package massive

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	massive "github.com/massive-com/client-go/v2/rest"
	v3gen "github.com/massive-com/client-go/v3/rest/gen"
)

// testTransport redirects SDK requests to a local test server.
type testTransport struct {
	server *httptest.Server
}

func (tt *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	serverURL, _ := url.Parse(tt.server.URL)
	clone := req.Clone(req.Context())
	clone.URL.Scheme = serverURL.Scheme
	clone.URL.Host = serverURL.Host
	return http.DefaultTransport.RoundTrip(clone)
}

// newTestClient creates a Client whose SDK is pointed at the given test server.
func newTestClient(ts *httptest.Server) *Client {
	httpClient := &http.Client{
		Transport: &testTransport{server: ts},
	}
	v3Client, _ := v3gen.NewClientWithResponses(ts.URL, v3gen.WithHTTPClient(httpClient))
	return &Client{
		sdk:        massive.NewWithClient("test-key", httpClient),
		tickerSDK:  v3Client,
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		maxRetries: 3,
		baseDelay:  time.Millisecond,
		maxDelay:   time.Millisecond,
		sleep:      func(d time.Duration) {},
	}
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{"zero value returns empty", time.Time{}, ""},
		{"formats date correctly", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC), "2024-06-15"},
		{"ignores time component", time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC), "2024-06-15"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDate(tt.time); got != tt.want {
				t.Errorf("formatDate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{"zero value returns empty", time.Time{}, ""},
		{"formats as RFC3339", time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC), "2024-06-15T14:30:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatTime(tt.time); got != tt.want {
				t.Errorf("formatTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDerefHelpers(t *testing.T) {
	t.Run("derefStr nil returns empty", func(t *testing.T) {
		if got := derefStr(nil); got != "" {
			t.Errorf("derefStr(nil) = %q, want empty", got)
		}
	})
	t.Run("derefStr non-nil", func(t *testing.T) {
		s := "hello"
		if got := derefStr(&s); got != "hello" {
			t.Errorf("derefStr(&s) = %q, want %q", got, "hello")
		}
	})
	t.Run("derefBool nil returns false", func(t *testing.T) {
		if got := derefBool(nil); got {
			t.Error("derefBool(nil) = true, want false")
		}
	})
	t.Run("derefBool non-nil", func(t *testing.T) {
		b := true
		if got := derefBool(&b); !got {
			t.Error("derefBool(&true) = false, want true")
		}
	})
	t.Run("derefTime nil returns zero", func(t *testing.T) {
		if got := derefTime(nil); !got.IsZero() {
			t.Errorf("derefTime(nil) = %v, want zero", got)
		}
	})
	t.Run("derefTime non-nil", func(t *testing.T) {
		tm := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		if got := derefTime(&tm); got != tm {
			t.Errorf("derefTime(&tm) = %v, want %v", got, tm)
		}
	})
}

func TestFetchActiveTickers_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "OK",
			"request_id": "test-123",
			"results": []map[string]any{
				{
					"ticker":           "AAPL",
					"name":             "Apple Inc.",
					"market":           "stocks",
					"primary_exchange": "XNAS",
					"type":             "CS",
					"active":           true,
					"currency_name":    "usd",
					"locale":           "us",
					"cik":              "0000320193",
				},
				{
					"ticker":           "MSFT",
					"name":             "Microsoft Corporation",
					"market":           "stocks",
					"primary_exchange": "XNAS",
					"type":             "CS",
					"active":           true,
					"currency_name":    "usd",
					"locale":           "us",
				},
			},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	tickers, err := c.FetchActiveTickers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tickers) != 2 {
		t.Fatalf("expected 2 tickers, got %d", len(tickers))
	}

	// Verify first ticker fields
	if tickers[0].Ticker != "AAPL" {
		t.Errorf("first ticker = %q, want %q", tickers[0].Ticker, "AAPL")
	}
	if tickers[0].Name != "Apple Inc." {
		t.Errorf("first name = %q, want %q", tickers[0].Name, "Apple Inc.")
	}
	if tickers[0].Market != "stocks" {
		t.Errorf("first market = %q, want %q", tickers[0].Market, "stocks")
	}
	if tickers[0].PrimaryExchange != "XNAS" {
		t.Errorf("first exchange = %q, want %q", tickers[0].PrimaryExchange, "XNAS")
	}
	if tickers[0].Type != "CS" {
		t.Errorf("first type = %q, want %q", tickers[0].Type, "CS")
	}
	if !tickers[0].Active {
		t.Error("first Active = false, want true")
	}
	if tickers[0].CurrencyName != "usd" {
		t.Errorf("first currency = %q, want %q", tickers[0].CurrencyName, "usd")
	}
	if tickers[0].Locale != "us" {
		t.Errorf("first locale = %q, want %q", tickers[0].Locale, "us")
	}
	if tickers[0].CIK != "0000320193" {
		t.Errorf("first CIK = %q, want %q", tickers[0].CIK, "0000320193")
	}

	// Verify second ticker
	if tickers[1].Ticker != "MSFT" {
		t.Errorf("second ticker = %q, want %q", tickers[1].Ticker, "MSFT")
	}
}

func TestFetchActiveTickers_EmptyResults(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "OK",
			"request_id": "test-123",
			"results":    []any{},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	tickers, err := c.FetchActiveTickers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tickers) != 0 {
		t.Errorf("expected 0 tickers, got %d", len(tickers))
	}
}

func TestFetchActiveTickers_UsesPageLimit(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		limit := r.URL.Query().Get("limit")
		if limit != "100" {
			t.Errorf("expected limit=100 in query, got %q", limit)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "OK",
			"request_id": "test-123",
			"results": []map[string]any{
				{"ticker": "AAPL", "name": "Apple Inc.", "market": "stocks", "locale": "us", "active": true},
				{"ticker": "MSFT", "name": "Microsoft Corporation", "market": "stocks", "locale": "us", "active": true},
			},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)

	tickers, err := c.FetchActiveTickers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tickers) != 2 {
		t.Fatalf("expected 2 tickers, got %d", len(tickers))
	}
	if calls != 1 {
		t.Errorf("expected exactly 1 API call, got %d", calls)
	}
}

func TestFetchActiveTickers_Pagination(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")

		if calls == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":     "OK",
				"request_id": "test-page1",
				"next_url":   "https://api.massive.com/v3/reference/tickers?cursor=page2",
				"results": []map[string]any{
					{"ticker": "AAPL", "name": "Apple Inc.", "market": "stocks", "locale": "us", "active": true},
					{"ticker": "AMZN", "name": "Amazon.com Inc.", "market": "stocks", "locale": "us", "active": true},
				},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "OK",
			"request_id": "test-page2",
			"results": []map[string]any{
				{"ticker": "GOOG", "name": "Alphabet Inc.", "market": "stocks", "locale": "us", "active": true},
				{"ticker": "MSFT", "name": "Microsoft Corporation", "market": "stocks", "locale": "us", "active": true},
			},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)

	tickers, err := c.FetchActiveTickers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tickers) != 4 {
		t.Fatalf("expected 4 tickers across 2 pages, got %d", len(tickers))
	}
	if calls != 2 {
		t.Errorf("expected 2 API calls, got %d", calls)
	}

	// Verify order preserved across pages
	want := []string{"AAPL", "AMZN", "GOOG", "MSFT"}
	for i, w := range want {
		if tickers[i].Ticker != w {
			t.Errorf("ticker[%d] = %q, want %q", i, tickers[i].Ticker, w)
		}
	}
}

func TestFetchActiveTickers_MaxTickersStopsPagination(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")

		if calls == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":     "OK",
				"request_id": "test-page1",
				"next_url":   "https://api.massive.com/v3/reference/tickers?cursor=page2",
				"results": []map[string]any{
					{"ticker": "AAPL", "name": "Apple Inc.", "market": "stocks", "locale": "us", "active": true},
					{"ticker": "AMZN", "name": "Amazon.com Inc.", "market": "stocks", "locale": "us", "active": true},
					{"ticker": "GOOG", "name": "Alphabet Inc.", "market": "stocks", "locale": "us", "active": true},
				},
			})
			return
		}

		// Should NOT be reached — maxTickers=3 met on first page
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "OK",
			"request_id": "test-page2",
			"results": []map[string]any{
				{"ticker": "MSFT", "name": "Microsoft Corporation", "market": "stocks", "locale": "us", "active": true},
			},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	c.maxTickers = 3

	tickers, err := c.FetchActiveTickers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tickers) != 3 {
		t.Fatalf("expected 3 tickers (maxTickers=3), got %d", len(tickers))
	}
	if calls != 1 {
		t.Errorf("expected 1 API call (maxTickers met on first page), got %d", calls)
	}
}

func TestFetchActiveTickers_RetriesOn429(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ERROR",
				"error":  "rate limited",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "OK",
			"request_id": "test-123",
			"results": []map[string]any{
				{
					"ticker": "AAPL",
					"name":   "Apple Inc.",
					"market": "stocks",
					"locale": "us",
					"active": true,
				},
			},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	tickers, err := c.FetchActiveTickers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tickers) != 1 {
		t.Fatalf("expected 1 ticker after retry, got %d", len(tickers))
	}
	if calls < 2 {
		t.Errorf("expected at least 2 API calls (retry on 429), got %d", calls)
	}
}

func TestFetchActiveTickers_NilResultsField(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "OK",
			"request_id": "test-123",
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	tickers, err := c.FetchActiveTickers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tickers) != 0 {
		t.Errorf("expected 0 tickers for nil results, got %d", len(tickers))
	}
}
