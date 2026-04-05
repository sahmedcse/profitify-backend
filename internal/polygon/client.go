package polygon

import (
	"errors"
	"fmt"
	"math"
	"time"

	massive "github.com/massive-com/client-go/v2/rest"
	"github.com/massive-com/client-go/v2/rest/models"
)

const (
	defaultMaxRetries    = 3
	defaultBaseDelay     = 2 * time.Second
	defaultMaxDelay      = 30 * time.Second
	defaultBackoffFactor = 2.0
)

// sleepFunc is a function that pauses execution for the given duration.
type sleepFunc func(time.Duration)

// Client wraps the Massive/Polygon SDK client with retry logic for 429 and 5xx errors.
type Client struct {
	sdk        *massive.Client
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
	sleep      sleepFunc
}

// NewClient creates a new Polygon client wrapper with default retry settings.
func NewClient(apiKey string) *Client {
	return &Client{
		sdk:        massive.New(apiKey),
		maxRetries: defaultMaxRetries,
		baseDelay:  defaultBaseDelay,
		maxDelay:   defaultMaxDelay,
		sleep:      time.Sleep,
	}
}

// isRetryable checks if an error from the SDK is a retryable 429 or 5xx response.
func isRetryable(err error) (statusCode int, retryable bool) {
	var errResp *models.ErrorResponse
	if errors.As(err, &errResp) {
		if errResp.StatusCode == 429 || errResp.StatusCode >= 500 {
			return errResp.StatusCode, true
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
			return fmt.Errorf("%s: max retries (%d) exceeded for status %d: %w",
				operation, c.maxRetries, statusCode, lastErr)
		}

		c.sleep(backoffDelay(attempt, c.baseDelay, c.maxDelay))
	}
	return lastErr
}
