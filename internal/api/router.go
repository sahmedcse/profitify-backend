package api

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/profitify/profitify-backend/internal/api/handler"
	"github.com/profitify/profitify-backend/internal/middleware"
)

// NewRouter creates and configures a Chi router with standard middleware and routes.
func NewRouter(logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.RequestLogger(logger))
	r.Use(chimw.Recoverer)

	r.Get("/health", handler.Health)

	return r
}
