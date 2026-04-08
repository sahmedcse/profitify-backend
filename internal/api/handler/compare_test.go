package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/profitify/profitify-backend/internal/domain"
)

func TestGetCompare(t *testing.T) {
	h, tr, _, pr, _, _, _ := newTestHandlers(t)
	tr.bySymbol["AAPL"] = &domain.Ticker{ID: "a", Ticker: "AAPL"}
	tr.bySymbol["MSFT"] = &domain.Ticker{ID: "b", Ticker: "MSFT"}

	now := time.Now().UTC()
	// Build 12 daily bars per ticker so the 5-step weekly sampler picks
	// up at least 3 points and we can verify the final summary uses the
	// last close.
	bars := func(start float64) []domain.DailyPrice {
		out := make([]domain.DailyPrice, 12)
		for i := range 12 {
			out[i] = domain.DailyPrice{
				Time:  now.Add(time.Duration(i-11) * 24 * time.Hour),
				Close: start + float64(i),
			}
		}
		return out
	}
	pr.bars["a"] = bars(100) // 100..111 → +11%
	pr.bars["b"] = bars(50)  // 50..61   → +22%

	r := httptest.NewRequest("GET", "/v1/compare?a=AAPL&b=MSFT&period=3M", nil)
	w := httptest.NewRecorder()
	h.GetCompare(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		A        string `json:"a"`
		B        string `json:"b"`
		Period   string `json:"period"`
		Series   []struct {
			A float64 `json:"a_pct"`
			B float64 `json:"b_pct"`
		} `json:"series"`
		SummaryA float64 `json:"summary_a_pct"`
		SummaryB float64 `json:"summary_b_pct"`
	}
	decodeJSON(t, w.Body, &resp)
	if resp.A != "AAPL" || resp.B != "MSFT" {
		t.Errorf("got a=%q b=%q, want AAPL/MSFT", resp.A, resp.B)
	}
	if resp.Period != "3M" {
		t.Errorf("period = %q, want 3M", resp.Period)
	}
	if len(resp.Series) == 0 {
		t.Fatalf("series empty")
	}
	// First sample is week-0 (index 0), so both series start at 0 pct.
	if resp.Series[0].A != 0 || resp.Series[0].B != 0 {
		t.Errorf("series[0] = (%v,%v), want (0,0)", resp.Series[0].A, resp.Series[0].B)
	}
	if resp.SummaryA <= 0 || resp.SummaryB <= 0 {
		t.Errorf("summary should be positive: a=%v b=%v", resp.SummaryA, resp.SummaryB)
	}
	// MSFT moved more (50→61 = 22%) than AAPL (100→111 = 11%).
	if resp.SummaryB <= resp.SummaryA {
		t.Errorf("expected MSFT > AAPL: a=%v b=%v", resp.SummaryA, resp.SummaryB)
	}
}

func TestGetCompare_MissingParams(t *testing.T) {
	h, _, _, _, _, _, _ := newTestHandlers(t)
	r := httptest.NewRequest("GET", "/v1/compare?a=AAPL", nil)
	w := httptest.NewRecorder()
	h.GetCompare(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestGetCompare_InvalidPeriod(t *testing.T) {
	h, tr, _, _, _, _, _ := newTestHandlers(t)
	tr.bySymbol["AAPL"] = &domain.Ticker{ID: "a", Ticker: "AAPL"}
	tr.bySymbol["MSFT"] = &domain.Ticker{ID: "b", Ticker: "MSFT"}

	r := httptest.NewRequest("GET", "/v1/compare?a=AAPL&b=MSFT&period=BOGUS", nil)
	w := httptest.NewRecorder()
	h.GetCompare(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestGetCompare_TickerNotFound(t *testing.T) {
	h, tr, _, _, _, _, _ := newTestHandlers(t)
	tr.bySymbol["AAPL"] = &domain.Ticker{ID: "a", Ticker: "AAPL"}

	r := httptest.NewRequest("GET", "/v1/compare?a=AAPL&b=NOPE&period=3M", nil)
	w := httptest.NewRecorder()
	h.GetCompare(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}
