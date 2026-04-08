package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/profitify/profitify-backend/internal/domain"
)

func TestGetPrices(t *testing.T) {
	h, tr, _, pr, _, _, _ := newTestHandlers(t)
	tr.bySymbol["AAPL"] = &domain.Ticker{ID: "id1", Ticker: "AAPL"}
	now := time.Now().UTC()
	pr.bars["id1"] = []domain.DailyPrice{
		{Time: now.Add(-2 * 24 * time.Hour), Open: 100, High: 105, Low: 99, Close: 104, Volume: 1000},
		{Time: now.Add(-1 * 24 * time.Hour), Open: 104, High: 106, Low: 103, Close: 105, Volume: 1100},
		{Time: now, Open: 105, High: 108, Low: 104, Close: 107, Volume: 1200},
	}

	r := httptest.NewRequest("GET", "/v1/tickers/AAPL/prices?timeframe=1M", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "AAPL")
	r = r.WithContext(contextWith(r, rctx))
	w := httptest.NewRecorder()
	h.GetPrices(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Symbol string `json:"symbol"`
		Bars   []struct {
			Close float64 `json:"close"`
		} `json:"bars"`
	}
	decodeJSON(t, w.Body, &resp)
	if resp.Symbol != "AAPL" {
		t.Errorf("symbol = %q, want AAPL", resp.Symbol)
	}
	if len(resp.Bars) != 3 {
		t.Fatalf("bars len = %d, want 3", len(resp.Bars))
	}
	if resp.Bars[2].Close != 107 {
		t.Errorf("last close = %v, want 107", resp.Bars[2].Close)
	}
}

func TestGetPrices_InvalidTimeframe(t *testing.T) {
	h, tr, _, _, _, _, _ := newTestHandlers(t)
	tr.bySymbol["AAPL"] = &domain.Ticker{ID: "id1", Ticker: "AAPL"}

	r := httptest.NewRequest("GET", "/v1/tickers/AAPL/prices?timeframe=BOGUS", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "AAPL")
	r = r.WithContext(contextWith(r, rctx))
	w := httptest.NewRecorder()
	h.GetPrices(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestGetPrices_TickerNotFound(t *testing.T) {
	h, _, _, _, _, _, _ := newTestHandlers(t)
	r := httptest.NewRequest("GET", "/v1/tickers/NOPE/prices", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "NOPE")
	r = r.WithContext(contextWith(r, rctx))
	w := httptest.NewRecorder()
	h.GetPrices(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}
