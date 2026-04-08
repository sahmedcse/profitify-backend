package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/profitify/profitify-backend/internal/repository"
)

// Handlers bundles every read-side repository the dashboard API needs.
// It is constructed once in main.go and its methods are mounted on the
// chi router. Keeping the dependencies on a struct makes the handlers
// trivial to mock in unit tests — supply fakes that satisfy the
// repository interfaces.
type Handlers struct {
	Tickers           repository.TickerRepository
	Prices            repository.DailyPriceRepository
	Technicals        repository.TickerTechnicalsRepository
	Fundamentals      repository.TickerFundamentalsRepository
	Stats             repository.TickerStatsRepository
	DividendSummaries repository.TickerDividendSummaryRepository
	Logger            *slog.Logger
}

// writeJSON serializes v as JSON and writes it with the given status.
// On encoder failure it logs but the response is already partially
// written so we can't recover with a different status.
func (h *Handlers) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.Logger.Error("encoding response", slog.String("error", err.Error()))
	}
}

// writeError writes a uniform JSON error envelope.
func (h *Handlers) writeError(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, map[string]string{"error": msg})
}

// notFoundOrInternal classifies a repository error: pgx.ErrNoRows
// becomes 404, anything else becomes 500.
func (h *Handlers) notFoundOrInternal(w http.ResponseWriter, err error, what string) {
	if errors.Is(err, pgx.ErrNoRows) {
		h.writeError(w, http.StatusNotFound, what+" not found")
		return
	}
	h.Logger.Error(what, slog.String("error", err.Error()))
	h.writeError(w, http.StatusInternalServerError, "internal error")
}
