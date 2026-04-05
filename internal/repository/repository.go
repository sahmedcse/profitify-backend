package repository

import (
	"context"

	"github.com/profitify/profitify-backend/internal/domain"
)

// TickerRepository persists and retrieves ticker metadata.
type TickerRepository interface {
	UpsertBatch(ctx context.Context, tickers []domain.Ticker) error
	GetActive(ctx context.Context) ([]domain.Ticker, error)
	GetBySymbol(ctx context.Context, symbol string) (*domain.Ticker, error)
}
