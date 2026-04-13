package massive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchDailyBars_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/v2/aggs/ticker/AAPL/range/1/day/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "OK",
			"resultsCount": 2,
			"results": []map[string]interface{}{
				{"o": 170.0, "h": 175.0, "l": 168.0, "c": 173.0, "v": 50000000.0, "t": 1712534400000},
				{"o": 173.0, "h": 178.0, "l": 172.0, "c": 177.0, "v": 48000000.0, "t": 1712620800000},
			},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	from := time.Date(2024, 4, 8, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 4, 9, 0, 0, 0, 0, time.UTC)
	prices, err := c.FetchDailyBars(context.Background(), "AAPL", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prices) != 2 {
		t.Fatalf("expected 2 bars, got %d", len(prices))
	}
	if prices[0].Open != 170.0 {
		t.Errorf("Open = %v, want 170.0", prices[0].Open)
	}
	if prices[1].Close != 177.0 {
		t.Errorf("Close = %v, want 177.0", prices[1].Close)
	}
	if prices[0].Volume != 50000000 {
		t.Errorf("Volume = %v, want 50000000", prices[0].Volume)
	}
}

func TestFetchDailyBars_Empty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "OK",
			"resultsCount": 0,
			"results":      []interface{}{},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	from := time.Date(2024, 4, 8, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 4, 9, 0, 0, 0, 0, time.UTC)
	prices, err := c.FetchDailyBars(context.Background(), "AAPL", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prices) != 0 {
		t.Errorf("expected 0 bars, got %d", len(prices))
	}
}

func TestFetchDailyOHLCV_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/open-close/AAPL/2026-04-08"
		if r.URL.Path != expectedPath {
			t.Errorf("unexpected path: got %s, want %s", r.URL.Path, expectedPath)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "OK",
			"symbol":     "AAPL",
			"from":       "2026-04-08",
			"open":       175.50,
			"high":       178.25,
			"low":        174.80,
			"close":      177.10,
			"volume":     52340000.0,
			"afterHours": 177.30,
			"preMarket":  175.20,
			"otc":        false,
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	date := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	price, err := c.FetchDailyOHLCV(context.Background(), "AAPL", date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if price.Open != 175.50 {
		t.Errorf("Open = %v, want 175.50", price.Open)
	}
	if price.High != 178.25 {
		t.Errorf("High = %v, want 178.25", price.High)
	}
	if price.Low != 174.80 {
		t.Errorf("Low = %v, want 174.80", price.Low)
	}
	if price.Close != 177.10 {
		t.Errorf("Close = %v, want 177.10", price.Close)
	}
	if price.Volume != 52340000 {
		t.Errorf("Volume = %v, want 52340000", price.Volume)
	}
	if price.AfterHours != 177.30 {
		t.Errorf("AfterHours = %v, want 177.30", price.AfterHours)
	}
	if price.PreMarket != 175.20 {
		t.Errorf("PreMarket = %v, want 175.20", price.PreMarket)
	}
}

func TestFetchDailyOHLCV_RetryOn429(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(429)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "ERROR",
				"error":      "rate limit exceeded",
				"request_id": "test",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "OK",
			"open":   100.0,
			"high":   105.0,
			"low":    99.0,
			"close":  103.0,
			"volume": 1000000.0,
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	date := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	price, err := c.FetchDailyOHLCV(context.Background(), "TEST", date)
	if err != nil {
		t.Fatalf("unexpected error after retry: %v", err)
	}
	if price.Close != 103.0 {
		t.Errorf("Close = %v, want 103.0", price.Close)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls (1 retry), got %d", calls)
	}
}
