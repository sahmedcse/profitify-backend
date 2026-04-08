package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/profitify/profitify-backend/internal/stats"
)

// GetLevels handles GET /v1/tickers/{symbol}/levels.
//
// Returns the materialized pivot_levels JSONB from ticker_stats. The
// payload shape matches internal/stats.PivotLevels exactly.
func (h *Handlers) GetLevels(w http.ResponseWriter, r *http.Request) {
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
	s, err := h.Stats.GetLatest(r.Context(), t.ID)
	if err != nil {
		h.notFoundOrInternal(w, err, "stats")
		return
	}

	var levels stats.PivotLevels
	if len(s.PivotLevels) > 0 {
		if err := json.Unmarshal(s.PivotLevels, &levels); err != nil {
			h.Logger.Error("decoding pivot_levels", "error", err.Error())
			h.writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"symbol": t.Ticker,
		"levels": levels,
	})
}
