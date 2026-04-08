package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/profitify/profitify-backend/internal/repository"
)

// ListTickers handles GET /v1/tickers.
//
// Query params:
//   - sector: optional canonical sector bucket (Technology, Financial, …)
//   - search: optional case-insensitive substring on symbol or name
//   - limit:  optional positive integer cap (default: no cap)
//
// Returns the sidebar-ready DashboardTicker rows from the materialized
// ticker_stats join.
func (h *Handlers) ListTickers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := repository.DashboardFilter{
		Sector: q.Get("sector"),
		Search: q.Get("search"),
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			h.writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		filter.Limit = n
	}

	rows, err := h.Tickers.ListForDashboard(r.Context(), filter)
	if err != nil {
		h.Logger.Error("ListForDashboard", "error", err.Error())
		h.writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"tickers": rows})
}

// GetTicker handles GET /v1/tickers/{symbol}.
//
// Returns the detail-header payload: ticker identity + latest stats
// joined inline. The frontend uses this for the page header (symbol,
// name, sector, price, change, signal, strength).
func (h *Handlers) GetTicker(w http.ResponseWriter, r *http.Request) {
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

	// Best-effort latest stats. We don't 404 the whole detail page just
	// because stats haven't been computed yet — return the identity row.
	stats, statsErr := h.Stats.GetLatest(r.Context(), t.ID)

	resp := map[string]any{
		"id":     t.ID,
		"symbol": t.Ticker,
		"name":   t.Name,
		"sector": t.Sector,
	}
	if statsErr == nil && stats != nil {
		resp["price_change"] = stats.PriceChange
		resp["price_change_pct"] = stats.PriceChangePct
		resp["signal_label"] = stats.SignalLabel
		resp["signal_strength"] = stats.SignalStrength
	}
	h.writeJSON(w, http.StatusOK, resp)
}
