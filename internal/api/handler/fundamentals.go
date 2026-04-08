package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// fundamentalsResponse is the slim payload for the dashboard's
// "Key Data / Fundamentals" panel. We deliberately omit fields the
// Massive feed doesn't carry (P/E, EPS, Beta) — they ship in Phase 2.
type fundamentalsResponse struct {
	Symbol        string   `json:"symbol"`
	MarketCap     float64  `json:"market_cap"`
	DividendYield *float64 `json:"dividend_yield"`
	High52w       *float64 `json:"high_52w"`
	Low52w        *float64 `json:"low_52w"`
	AvgVolume30d  *float64 `json:"avg_volume_30d"`
}

// GetFundamentals handles GET /v1/tickers/{symbol}/fundamentals.
//
// Joins the latest fundamentals row with the latest stats row (52w
// extremes + avg volume live in stats, not fundamentals) and the
// dividend summary. The whole panel renders from one response.
func (h *Handlers) GetFundamentals(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		h.writeError(w, http.StatusBadRequest, "symbol required")
		return
	}
	t, err := h.Tickers.GetBySymbol(r.Context(), symbol)
	if err != nil {
		h.notFoundOrInternal(w, err, "ticker")
		return
	}

	resp := fundamentalsResponse{Symbol: t.Ticker}

	if f, err := h.Fundamentals.GetLatest(r.Context(), t.ID); err == nil {
		resp.MarketCap = f.MarketCap
	} else if !errors.Is(err, pgx.ErrNoRows) {
		h.Logger.Error("fundamentals.GetLatest", "error", err.Error())
	}

	if s, err := h.Stats.GetLatest(r.Context(), t.ID); err == nil {
		resp.High52w = s.High52w
		resp.Low52w = s.Low52w
		resp.AvgVolume30d = s.AvgVolume30d
	} else if !errors.Is(err, pgx.ErrNoRows) {
		h.Logger.Error("stats.GetLatest", "error", err.Error())
	}

	if d, err := h.DividendSummaries.GetLatest(r.Context(), t.ID); err == nil {
		resp.DividendYield = d.CurrentYield
	} else if !errors.Is(err, pgx.ErrNoRows) {
		h.Logger.Error("dividend.GetLatest", "error", err.Error())
	}

	h.writeJSON(w, http.StatusOK, resp)
}
