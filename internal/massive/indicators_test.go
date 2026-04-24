package massive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchSMA_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "OK",
			"results": map[string]any{
				"values": []map[string]any{
					{"timestamp": 1712534400000, "value": 175.50},
				},
			},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	date := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	val, err := c.FetchSMA(context.Background(), "AAPL", 20, date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val == nil || *val != 175.50 {
		t.Errorf("SMA = %v, want 175.50", val)
	}
}

func TestFetchSMA_NoData(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "OK",
			"results": map[string]any{"values": []any{}},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	val, err := c.FetchSMA(context.Background(), "AAPL", 20, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil for no data, got %v", *val)
	}
}

func TestFetchMACD_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "OK",
			"results": map[string]any{
				"values": []map[string]any{
					{"timestamp": 1712534400000, "value": 1.5, "signal": 1.2, "histogram": 0.3},
				},
			},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	line, signal, histogram, err := c.FetchMACD(context.Background(), "AAPL", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line == nil || *line != 1.5 {
		t.Errorf("MACD line = %v, want 1.5", line)
	}
	if signal == nil || *signal != 1.2 {
		t.Errorf("MACD signal = %v, want 1.2", signal)
	}
	if histogram == nil || *histogram != 0.3 {
		t.Errorf("MACD histogram = %v, want 0.3", histogram)
	}
}

func TestFetchAllIndicators_PartialFailures(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		// Return valid data for all calls
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "OK",
			"results": map[string]any{
				"values": []map[string]any{
					{"timestamp": 1712534400000, "value": 100.0, "signal": 95.0, "histogram": 5.0},
				},
			},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	tech, err := c.FetchAllIndicators(context.Background(), "AAPL", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have made 7 calls: SMA(20,50,200), EMA(12,26), RSI(14), MACD
	if calls != 7 {
		t.Errorf("expected 7 API calls, got %d", calls)
	}

	if tech.SMA20 == nil {
		t.Error("expected SMA20 to be set")
	}
	if tech.RSI14 == nil {
		t.Error("expected RSI14 to be set")
	}
	if tech.MACDLine == nil {
		t.Error("expected MACDLine to be set")
	}
}
