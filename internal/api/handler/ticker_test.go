package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/profitify/profitify-backend/internal/api/handler"
	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/repository"
)

func newTestHandlers(t *testing.T) (*handler.Handlers, *fakeTickerRepo, *fakeStatsRepo, *fakePriceRepo, *fakeTechnicalsRepo, *fakeFundamentalsRepo, *fakeDividendRepo) {
	t.Helper()
	tr := &fakeTickerRepo{bySymbol: map[string]*domain.Ticker{}}
	sr := &fakeStatsRepo{rows: map[string]*domain.TickerStats{}}
	pr := &fakePriceRepo{bars: map[string][]domain.DailyPrice{}}
	techR := &fakeTechnicalsRepo{rows: map[string]*domain.TechnicalIndicators{}}
	fr := &fakeFundamentalsRepo{rows: map[string]*domain.TickerFundamentals{}}
	dr := &fakeDividendRepo{rows: map[string]*domain.TickerDividendSummary{}}
	h := &handler.Handlers{
		Tickers: tr, Stats: sr, Prices: pr, Technicals: techR,
		Fundamentals: fr, DividendSummaries: dr, Logger: discardLogger,
	}
	return h, tr, sr, pr, techR, fr, dr
}

func TestListTickers(t *testing.T) {
	h, tr, _, _, _, _, _ := newTestHandlers(t)
	tr.dashboard = []repository.DashboardTicker{
		{Symbol: "AAPL", Name: "Apple", Sector: domain.SectorTechnology, LatestPrice: 150, SignalStrength: 70},
		{Symbol: "JPM", Name: "JPMorgan", Sector: domain.SectorFinancial, LatestPrice: 200, SignalStrength: 50},
	}

	r := httptest.NewRequest("GET", "/v1/tickers", nil)
	w := httptest.NewRecorder()
	h.ListTickers(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Tickers []repository.DashboardTicker `json:"tickers"`
	}
	decodeJSON(t, w.Body, &resp)
	if len(resp.Tickers) != 2 {
		t.Errorf("len = %d, want 2", len(resp.Tickers))
	}
}

func TestListTickers_FilterBySector(t *testing.T) {
	h, tr, _, _, _, _, _ := newTestHandlers(t)
	tr.dashboard = []repository.DashboardTicker{
		{Symbol: "AAPL", Sector: domain.SectorTechnology},
		{Symbol: "JPM", Sector: domain.SectorFinancial},
	}

	r := httptest.NewRequest("GET", "/v1/tickers?sector=Financial", nil)
	w := httptest.NewRecorder()
	h.ListTickers(w, r)

	var resp struct {
		Tickers []repository.DashboardTicker `json:"tickers"`
	}
	decodeJSON(t, w.Body, &resp)
	if len(resp.Tickers) != 1 || resp.Tickers[0].Symbol != "JPM" {
		t.Errorf("got %+v, want [JPM]", resp.Tickers)
	}
}

func TestListTickers_InvalidLimit(t *testing.T) {
	h, _, _, _, _, _, _ := newTestHandlers(t)
	r := httptest.NewRequest("GET", "/v1/tickers?limit=oops", nil)
	w := httptest.NewRecorder()
	h.ListTickers(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestGetTicker(t *testing.T) {
	h, tr, sr, _, _, _, _ := newTestHandlers(t)
	tr.bySymbol["AAPL"] = &domain.Ticker{ID: "ticker-1", Ticker: "AAPL", Name: "Apple", Sector: domain.SectorTechnology}
	sr.rows["ticker-1"] = &domain.TickerStats{
		PriceChange: 1.5, PriceChangePct: 1.0, SignalLabel: "Bullish", SignalStrength: 65,
	}

	r := httptest.NewRequest("GET", "/v1/tickers/AAPL", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "AAPL")
	r = r.WithContext(contextWith(r, rctx))
	w := httptest.NewRecorder()
	h.GetTicker(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	decodeJSON(t, w.Body, &resp)
	if resp["symbol"] != "AAPL" {
		t.Errorf("symbol = %v, want AAPL", resp["symbol"])
	}
	if resp["signal_label"] != "Bullish" {
		t.Errorf("signal_label = %v, want Bullish", resp["signal_label"])
	}
}

func TestGetTicker_NotFound(t *testing.T) {
	h, _, _, _, _, _, _ := newTestHandlers(t)
	r := httptest.NewRequest("GET", "/v1/tickers/NOPE", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "NOPE")
	r = r.WithContext(contextWith(r, rctx))
	w := httptest.NewRecorder()
	h.GetTicker(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}
