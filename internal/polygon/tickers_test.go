package polygon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	massive "github.com/massive-com/client-go/v2/rest"
	"github.com/massive-com/client-go/v2/rest/models"
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
	return &Client{
		sdk:        massive.NewWithClient("test-key", httpClient),
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

func TestMapTicker(t *testing.T) {
	sdkTicker := models.Ticker{
		Ticker:          "AAPL",
		Name:            "Apple Inc.",
		Market:          "stocks",
		PrimaryExchange: "XNAS",
		Type:            "CS",
		Active:          true,
		CurrencyName:    "usd",
		Locale:          "us",
		CIK:             "0000320193",
		ListDate:        models.Date(time.Date(1980, 12, 12, 0, 0, 0, 0, time.UTC)),
	}

	got := mapTicker(sdkTicker)

	if got.Ticker != "AAPL" {
		t.Errorf("Ticker = %q, want %q", got.Ticker, "AAPL")
	}
	if got.Name != "Apple Inc." {
		t.Errorf("Name = %q, want %q", got.Name, "Apple Inc.")
	}
	if got.Market != "stocks" {
		t.Errorf("Market = %q, want %q", got.Market, "stocks")
	}
	if got.PrimaryExchange != "XNAS" {
		t.Errorf("PrimaryExchange = %q, want %q", got.PrimaryExchange, "XNAS")
	}
	if got.Type != "CS" {
		t.Errorf("Type = %q, want %q", got.Type, "CS")
	}
	if !got.Active {
		t.Error("Active = false, want true")
	}
	if got.CurrencyName != "usd" {
		t.Errorf("CurrencyName = %q, want %q", got.CurrencyName, "usd")
	}
	if got.Locale != "us" {
		t.Errorf("Locale = %q, want %q", got.Locale, "us")
	}
	if got.CIK != "0000320193" {
		t.Errorf("CIK = %q, want %q", got.CIK, "0000320193")
	}
	if got.ListDate != "1980-12-12" {
		t.Errorf("ListDate = %q, want %q", got.ListDate, "1980-12-12")
	}
	if got.DelistedUTC != "" {
		t.Errorf("DelistedUTC = %q, want empty", got.DelistedUTC)
	}
}

func TestMapTicker_ZeroDates(t *testing.T) {
	sdkTicker := models.Ticker{
		Ticker: "TEST",
		Name:   "Test Corp",
		Active: true,
	}

	got := mapTicker(sdkTicker)

	if got.ListDate != "" {
		t.Errorf("ListDate = %q, want empty for zero value", got.ListDate)
	}
	if got.DelistedUTC != "" {
		t.Errorf("DelistedUTC = %q, want empty for zero value", got.DelistedUTC)
	}
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
					"primary_exchange":  "XNAS",
					"type":             "CS",
					"active":           true,
					"currency_name":    "usd",
					"locale":           "us",
				},
				{
					"ticker":           "MSFT",
					"name":             "Microsoft Corporation",
					"market":           "stocks",
					"primary_exchange":  "XNAS",
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
	if tickers[0].Ticker != "AAPL" {
		t.Errorf("first ticker = %q, want %q", tickers[0].Ticker, "AAPL")
	}
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
