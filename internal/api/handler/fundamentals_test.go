package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/profitify/profitify-backend/internal/domain"
)

func TestGetFundamentals(t *testing.T) {
	h, tr, sr, _, _, fr, dr := newTestHandlers(t)
	tr.bySymbol["AAPL"] = &domain.Ticker{ID: "id1", Ticker: "AAPL"}
	fr.rows["id1"] = &domain.TickerFundamentals{MarketCap: 3_000_000_000_000}
	sr.rows["id1"] = &domain.TickerStats{
		High52w:      ptr(200),
		Low52w:       ptr(120),
		AvgVolume30d: ptr(50_000_000),
	}
	dr.rows["id1"] = &domain.TickerDividendSummary{CurrentYield: ptr(0.55)}

	r := httptest.NewRequest("GET", "/v1/tickers/AAPL/fundamentals", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "AAPL")
	r = r.WithContext(contextWith(r, rctx))
	w := httptest.NewRecorder()
	h.GetFundamentals(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Symbol        string   `json:"symbol"`
		MarketCap     float64  `json:"market_cap"`
		DividendYield *float64 `json:"dividend_yield"`
		High52w       *float64 `json:"high_52w"`
		Low52w        *float64 `json:"low_52w"`
		AvgVolume30d  *float64 `json:"avg_volume_30d"`
	}
	decodeJSON(t, w.Body, &resp)
	if resp.Symbol != "AAPL" {
		t.Errorf("symbol = %q, want AAPL", resp.Symbol)
	}
	if resp.MarketCap != 3_000_000_000_000 {
		t.Errorf("market_cap = %v, want 3T", resp.MarketCap)
	}
	if resp.High52w == nil || *resp.High52w != 200 {
		t.Errorf("high_52w = %v, want 200", resp.High52w)
	}
	if resp.DividendYield == nil || *resp.DividendYield != 0.55 {
		t.Errorf("dividend_yield = %v, want 0.55", resp.DividendYield)
	}
}

// GetFundamentals is intentionally tolerant of partial data: a ticker
// with no fundamentals/stats/dividend rows still returns 200 with the
// symbol set and the rest of the fields zero/nil. This protects the
// dashboard from one missing source breaking the whole panel.
func TestGetFundamentals_TolerantOfMissingPieces(t *testing.T) {
	h, tr, _, _, _, _, _ := newTestHandlers(t)
	tr.bySymbol["AAPL"] = &domain.Ticker{ID: "id1", Ticker: "AAPL"}

	r := httptest.NewRequest("GET", "/v1/tickers/AAPL/fundamentals", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "AAPL")
	r = r.WithContext(contextWith(r, rctx))
	w := httptest.NewRecorder()
	h.GetFundamentals(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Symbol    string  `json:"symbol"`
		MarketCap float64 `json:"market_cap"`
	}
	decodeJSON(t, w.Body, &resp)
	if resp.Symbol != "AAPL" {
		t.Errorf("symbol = %q, want AAPL", resp.Symbol)
	}
	if resp.MarketCap != 0 {
		t.Errorf("market_cap = %v, want 0", resp.MarketCap)
	}
}

func TestGetFundamentals_TickerNotFound(t *testing.T) {
	h, _, _, _, _, _, _ := newTestHandlers(t)
	r := httptest.NewRequest("GET", "/v1/tickers/NOPE/fundamentals", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "NOPE")
	r = r.WithContext(contextWith(r, rctx))
	w := httptest.NewRecorder()
	h.GetFundamentals(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}
