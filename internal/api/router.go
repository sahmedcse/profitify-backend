package api

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/profitify/profitify-backend/internal/api/handler"
	"github.com/profitify/profitify-backend/internal/middleware"
)

// NewRouter creates and configures a Chi router with standard middleware,
// the health endpoint, and the dashboard API mounted under /v1.
//
// All dashboard endpoints are thin reads against the materialized
// dashboard columns — no per-request computation. The Handlers struct
// holds repository dependencies which are injected from main.go for
// trivial unit-test substitution.
func NewRouter(logger *slog.Logger, h *handler.Handlers) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.RequestLogger(logger))
	r.Use(chimw.Recoverer)

	r.Get("/health", handler.Health)

	r.Route("/v1", func(r chi.Router) {
		r.Get("/tickers", h.ListTickers)
		r.Get("/tickers/{symbol}", h.GetTicker)
		r.Get("/tickers/{symbol}/prices", h.GetPrices)
		r.Get("/tickers/{symbol}/technicals", h.GetTechnicals)
		r.Get("/tickers/{symbol}/fundamentals", h.GetFundamentals)
		r.Get("/tickers/{symbol}/levels", h.GetLevels)
		r.Get("/compare", h.GetCompare)
	})

	return r
}
