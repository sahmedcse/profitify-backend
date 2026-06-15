package massive

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"

	massive "github.com/massive-com/client-go/v2/rest"
	"github.com/massive-com/client-go/v2/rest/models"
	v3gen "github.com/massive-com/client-go/v3/rest/gen"
)

const (
	defaultMaxRetries    = 3
	defaultBaseDelay     = 2 * time.Second
	defaultMaxDelay      = 30 * time.Second
	defaultBackoffFactor = 2.0
	defaultTickerLimit   = 100
	massiveBaseURL       = "https://api.massive.com"
)

// sleepFunc is a function that pauses execution for the given duration.
type sleepFunc func(time.Duration)

// Option configures optional Client settings.
type Option func(*Client)

// Client wraps the Massive SDK client with retry logic for 429 and 5xx errors.
type Client struct {
	sdk         *massive.Client            // v2 — used by indicators, dividends, aggregates, fundamentals
	tickerSDK   *v3gen.ClientWithResponses // v3 — used by FetchActiveTickers (paginated)
	logger      *slog.Logger
	maxRetries  int
	tickerLimit int
	baseDelay   time.Duration
	maxDelay    time.Duration
	sleep       sleepFunc
}

// WithTickerLimit sets the page size for ListTickers API requests.
func WithTickerLimit(n int) Option {
	return func(c *Client) {
		c.tickerLimit = n
	}
}

// httpError represents an HTTP error with a status code for retry logic.
type httpError struct {
	statusCode int
	body       string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.statusCode, e.body)
}

// NewClient creates a new Massive client wrapper with default retry settings.
func NewClient(apiKey string, logger *slog.Logger, opts ...Option) *Client {
	v3Client, _ := v3gen.NewClientWithResponses(massiveBaseURL,
		v3gen.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+apiKey)
			return nil
		}),
	)

	c := &Client{
		sdk:         massive.New(apiKey),
		tickerSDK:   v3Client,
		logger:      logger,
		maxRetries:  defaultMaxRetries,
		tickerLimit: defaultTickerLimit,
		baseDelay:   defaultBaseDelay,
		maxDelay:    defaultMaxDelay,
		sleep:       time.Sleep,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// isRetryable checks if an error from the SDK is a retryable 429 or 5xx response.
func isRetryable(err error) (statusCode int, retryable bool) {
	var errResp *models.ErrorResponse
	if errors.As(err, &errResp) {
		if errResp.StatusCode == 429 || errResp.StatusCode >= 500 {
			return errResp.StatusCode, true
		}
	}
	var httpErr *httpError
	if errors.As(err, &httpErr) {
		if httpErr.statusCode == 429 || httpErr.statusCode >= 500 {
			return httpErr.statusCode, true
		}
	}
	return 0, false
}

// backoffDelay calculates exponential backoff delay capped at maxDelay.
func backoffDelay(attempt int, base, max time.Duration) time.Duration {
	delay := time.Duration(float64(base) * math.Pow(defaultBackoffFactor, float64(attempt)))
	if delay > max {
		return max
	}
	return delay
}

// retry executes fn with exponential backoff on retryable errors (429, 5xx).
func (c *Client) retry(operation string, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		statusCode, retryable := isRetryable(lastErr)
		if !retryable {
			return lastErr
		}

		if attempt == c.maxRetries {
			c.logger.Error("max retries exceeded",
				"operation", operation,
				"retries", c.maxRetries,
				"status", statusCode,
			)
			return fmt.Errorf("%s: max retries (%d) exceeded for status %d: %w",
				operation, c.maxRetries, statusCode, lastErr)
		}

		delay := backoffDelay(attempt, c.baseDelay, c.maxDelay)
		c.logger.Warn("retryable error, backing off",
			"operation", operation,
			"attempt", attempt+1,
			"max_retries", c.maxRetries,
			"status", statusCode,
			"delay", delay,
		)
		c.sleep(delay)
	}
	return lastErr
}
