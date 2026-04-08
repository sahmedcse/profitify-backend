package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/profitify/profitify-backend/internal/domain"
)

func TestGetTechnicals_FromMaterializedStatuses(t *testing.T) {
	h, tr, _, pr, techR, _, _ := newTestHandlers(t)
	tr.bySymbol["AAPL"] = &domain.Ticker{ID: "id1", Ticker: "AAPL"}
	now := time.Now().UTC()
	pr.bars["id1"] = []domain.DailyPrice{
		{Time: now, Close: 152},
	}
	techR.rows["id1"] = &domain.TechnicalIndicators{
		Time:  now,
		RSI14: ptr(72),
		IndicatorStatuses: map[string]string{
			"rsi_14": "bullish",
		},
	}

	r := httptest.NewRequest("GET", "/v1/tickers/AAPL/technicals", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "AAPL")
	r = r.WithContext(contextWith(r, rctx))
	w := httptest.NewRecorder()
	h.GetTechnicals(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Symbol     string `json:"symbol"`
		Indicators []struct {
			Name   string  `json:"name"`
			Value  float64 `json:"value"`
			Status string  `json:"status"`
		} `json:"indicators"`
		BuyCount int `json:"buy_count"`
	}
	decodeJSON(t, w.Body, &resp)
	if resp.Symbol != "AAPL" {
		t.Errorf("symbol = %q, want AAPL", resp.Symbol)
	}
	if len(resp.Indicators) != 1 {
		t.Fatalf("indicators len = %d, want 1", len(resp.Indicators))
	}
	if resp.Indicators[0].Status != "bullish" {
		t.Errorf("status = %q, want bullish", resp.Indicators[0].Status)
	}
	if resp.BuyCount != 1 {
		t.Errorf("buy_count = %d, want 1", resp.BuyCount)
	}
}

func TestGetTechnicals_FallbackClassification(t *testing.T) {
	// No materialized indicator_statuses — handler should compute on the fly.
	h, tr, _, pr, techR, _, _ := newTestHandlers(t)
	tr.bySymbol["AAPL"] = &domain.Ticker{ID: "id1", Ticker: "AAPL"}
	now := time.Now().UTC()
	pr.bars["id1"] = []domain.DailyPrice{{Time: now, Close: 105}}
	techR.rows["id1"] = &domain.TechnicalIndicators{
		Time:       now,
		RSI14:      ptr(50),
		MACDLine:   ptr(1.5),
		MACDSignal: ptr(1.0),
		SMA20:      ptr(100), // close 105 > 100 → bullish
	}

	r := httptest.NewRequest("GET", "/v1/tickers/AAPL/technicals", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "AAPL")
	r = r.WithContext(contextWith(r, rctx))
	w := httptest.NewRecorder()
	h.GetTechnicals(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Indicators []struct {
			Name, Status string
		} `json:"indicators"`
		BuyCount int `json:"buy_count"`
	}
	decodeJSON(t, w.Body, &resp)
	if resp.BuyCount < 2 {
		t.Errorf("buy_count = %d, want ≥2 (MACD + SMA20)", resp.BuyCount)
	}
}
