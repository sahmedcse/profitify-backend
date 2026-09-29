package massive

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/massive-com/client-go/v2/rest/models"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantCode  int
		wantRetry bool
	}{
		{
			name:      "429 is retryable",
			err:       &models.ErrorResponse{StatusCode: 429},
			wantCode:  429,
			wantRetry: true,
		},
		{
			name:      "500 is retryable",
			err:       &models.ErrorResponse{StatusCode: 500},
			wantCode:  500,
			wantRetry: true,
		},
		{
			name:      "502 is retryable",
			err:       &models.ErrorResponse{StatusCode: 502},
			wantCode:  502,
			wantRetry: true,
		},
		{
			name:      "503 is retryable",
			err:       &models.ErrorResponse{StatusCode: 503},
			wantCode:  503,
			wantRetry: true,
		},
		{
			name:      "400 is not retryable",
			err:       &models.ErrorResponse{StatusCode: 400},
			wantCode:  0,
			wantRetry: false,
		},
		{
			name:      "401 is not retryable",
			err:       &models.ErrorResponse{StatusCode: 401},
			wantCode:  0,
			wantRetry: false,
		},
		{
			name:      "404 is not retryable",
			err:       &models.ErrorResponse{StatusCode: 404},
			wantCode:  0,
			wantRetry: false,
		},
		{
			name:      "non-ErrorResponse is not retryable",
			err:       errors.New("connection refused"),
			wantCode:  0,
			wantRetry: false,
		},
		{
			name:      "wrapped ErrorResponse is retryable",
			err:       fmt.Errorf("iterating: %w", &models.ErrorResponse{StatusCode: 429}),
			wantCode:  429,
			wantRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, retryable := isRetryable(tt.err)
			if retryable != tt.wantRetry {
				t.Errorf("isRetryable() retryable = %v, want %v", retryable, tt.wantRetry)
			}
			if code != tt.wantCode {
				t.Errorf("isRetryable() code = %v, want %v", code, tt.wantCode)
			}
		})
	}
}

func TestBackoffDelay(t *testing.T) {
	base := 2 * time.Second
	max := 30 * time.Second

	tests := []struct {
		name    string
		attempt int
		want    time.Duration
	}{
		{"attempt 0", 0, 2 * time.Second},
		{"attempt 1", 1, 4 * time.Second},
		{"attempt 2", 2, 8 * time.Second},
		{"attempt 3", 3, 16 * time.Second},
		{"attempt 4 capped at max", 4, 30 * time.Second},
		{"attempt 10 capped at max", 10, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := backoffDelay(tt.attempt, base, max)
			if got != tt.want {
				t.Errorf("backoffDelay(%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestRetry_SucceedsImmediately(t *testing.T) {
	c := &Client{
		logger:     discardLogger,
		maxRetries: 3,
		baseDelay:  time.Millisecond,
		maxDelay:   time.Millisecond,
		sleep:      func(d time.Duration) {},
	}

	calls := 0
	err := c.retry("test", func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetry_RetriesOnRetryableError(t *testing.T) {
	c := &Client{
		logger:     discardLogger,
		maxRetries: 3,
		baseDelay:  time.Millisecond,
		maxDelay:   time.Millisecond,
		sleep:      func(d time.Duration) {},
	}

	calls := 0
	err := c.retry("test", func() error {
		calls++
		if calls < 3 {
			return &models.ErrorResponse{StatusCode: 429}
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_StopsOnNonRetryableError(t *testing.T) {
	c := &Client{
		logger:     discardLogger,
		maxRetries: 3,
		baseDelay:  time.Millisecond,
		maxDelay:   time.Millisecond,
		sleep:      func(d time.Duration) {},
	}

	calls := 0
	err := c.retry("test", func() error {
		calls++
		return &models.ErrorResponse{StatusCode: 400}
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry for 400), got %d", calls)
	}
}

func TestRetry_ExhaustsMaxRetries(t *testing.T) {
	c := &Client{
		logger:     discardLogger,
		maxRetries: 2,
		baseDelay:  time.Millisecond,
		maxDelay:   time.Millisecond,
		sleep:      func(d time.Duration) {},
	}

	calls := 0
	err := c.retry("test-op", func() error {
		calls++
		return &models.ErrorResponse{StatusCode: 500}
	})

	if err == nil {
		t.Fatal("expected error after max retries, got nil")
	}
	// 1 initial + 2 retries = 3 total
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
	if !errors.Is(err, &models.ErrorResponse{}) {
		// Check the error message contains our context
		want := "test-op: max retries (2) exceeded"
		if got := err.Error(); len(got) < len(want) {
			t.Errorf("error message too short: %s", got)
		}
	}
}

func TestRetry_CallsSleepBetweenAttempts(t *testing.T) {
	var sleepDurations []time.Duration
	c := &Client{
		logger:     discardLogger,
		maxRetries: 2,
		baseDelay:  2 * time.Second,
		maxDelay:   30 * time.Second,
		sleep: func(d time.Duration) {
			sleepDurations = append(sleepDurations, d)
		},
	}

	_ = c.retry("test", func() error {
		return &models.ErrorResponse{StatusCode: 429}
	})

	if len(sleepDurations) != 2 {
		t.Fatalf("expected 2 sleeps, got %d", len(sleepDurations))
	}
	if sleepDurations[0] != 2*time.Second {
		t.Errorf("first sleep = %v, want 2s", sleepDurations[0])
	}
	if sleepDurations[1] != 4*time.Second {
		t.Errorf("second sleep = %v, want 4s", sleepDurations[1])
	}
}

func TestNewClient_Defaults(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	c := NewClient("test-key", logger)

	if c == nil {
		t.Fatal("NewClient() returned nil")
	}
	if c.sdk == nil {
		t.Error("sdk client was not initialised")
	}
	if c.tickerSDK == nil {
		t.Error("v3 ticker client was not initialised")
	}
	if c.logger != logger {
		t.Error("logger was not stored on the client")
	}
	if c.maxRetries != defaultMaxRetries {
		t.Errorf("maxRetries = %d, want %d", c.maxRetries, defaultMaxRetries)
	}
	if c.maxTickers != 0 {
		t.Errorf("maxTickers = %d, want 0 (no limit)", c.maxTickers)
	}
	if c.baseDelay != defaultBaseDelay {
		t.Errorf("baseDelay = %v, want %v", c.baseDelay, defaultBaseDelay)
	}
	if c.maxDelay != defaultMaxDelay {
		t.Errorf("maxDelay = %v, want %v", c.maxDelay, defaultMaxDelay)
	}
	if c.sleep == nil {
		t.Error("sleep func was not set")
	}
}

func TestNewClient_AppliesOptions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name string
		opts []Option
		want int
	}{
		{name: "no options keeps default", opts: nil, want: 0},
		{name: "single option", opts: []Option{WithMaxTickers(25)}, want: 25},
		{name: "last option wins", opts: []Option{WithMaxTickers(5), WithMaxTickers(11)}, want: 11},
		{name: "zero is no limit", opts: []Option{WithMaxTickers(0)}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient("test-key", logger, tt.opts...)
			if c.maxTickers != tt.want {
				t.Errorf("maxTickers = %d, want %d", c.maxTickers, tt.want)
			}
		})
	}
}

func TestHTTPError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *httpError
		want string
	}{
		{name: "rate limited", err: &httpError{statusCode: 429, body: "slow down"}, want: "HTTP 429: slow down"},
		{name: "server error", err: &httpError{statusCode: 500, body: "boom"}, want: "HTTP 500: boom"},
		{name: "empty body", err: &httpError{statusCode: 404, body: ""}, want: "HTTP 404: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHTTPError_SatisfiesErrorInterface(t *testing.T) {
	var err error = &httpError{statusCode: 503, body: "unavailable"}
	var target *httpError
	if !errors.As(err, &target) {
		t.Fatal("httpError should be matchable with errors.As")
	}
	if target.statusCode != 503 {
		t.Errorf("statusCode = %d, want 503", target.statusCode)
	}
}
