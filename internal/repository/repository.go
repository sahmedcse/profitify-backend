package repository

import (
	"context"
	"time"

	"github.com/profitify/profitify-backend/internal/domain"
)

// DashboardFilter narrows the result set returned by
// TickerRepository.ListForDashboard. All fields are optional.
type DashboardFilter struct {
	// Sector restricts results to a single canonical sector bucket
	// (see internal/domain.Sector* constants). Empty = no filter.
	Sector string
	// Search restricts results to tickers whose symbol or name matches
	// the substring (case-insensitive). Empty = no filter.
	Search string
	// Limit caps the number of returned rows. <=0 means no cap.
	Limit int
}

// DashboardTicker is the row shape returned by ListForDashboard. It
// joins ticker identity with the most recent stats so the sidebar can
// render in a single SQL query.
type DashboardTicker struct {
	ID             string  `json:"id"`
	Symbol         string  `json:"symbol"`
	Name           string  `json:"name"`
	Sector         string  `json:"sector"`
	LatestPrice    float64 `json:"latest_price"`
	PriceChange    float64 `json:"price_change"`
	PriceChangePct float64 `json:"price_change_pct"`
	SignalLabel    string  `json:"signal_label"`
	SignalStrength int     `json:"signal_strength"`
}

// TickerRepository persists and retrieves ticker metadata and the
// dashboard rollup view.
type TickerRepository interface {
	UpsertBatch(ctx context.Context, tickers []domain.Ticker) error
	GetActive(ctx context.Context) ([]domain.Ticker, error)
	GetBySymbol(ctx context.Context, symbol string) (*domain.Ticker, error)
	ListForDashboard(ctx context.Context, filter DashboardFilter) ([]DashboardTicker, error)
}

// DailyPriceRepository reads OHLCV bars from the daily_prices hypertable.
type DailyPriceRepository interface {
	GetRange(ctx context.Context, tickerID string, from, to time.Time) ([]domain.DailyPrice, error)
}

// TickerTechnicalsRepository reads the latest technical indicator snapshot.
type TickerTechnicalsRepository interface {
	GetLatest(ctx context.Context, tickerID string) (*domain.TechnicalIndicators, error)
}

// TickerFundamentalsRepository reads the latest fundamentals row.
type TickerFundamentalsRepository interface {
	GetLatest(ctx context.Context, tickerID string) (*domain.TickerFundamentals, error)
}

// TickerStatsRepository reads the latest rolling stats row.
type TickerStatsRepository interface {
	GetLatest(ctx context.Context, tickerID string) (*domain.TickerStats, error)
}

// TickerDividendSummaryRepository reads the latest dividend summary row.
type TickerDividendSummaryRepository interface {
	GetLatest(ctx context.Context, tickerID string) (*domain.TickerDividendSummary, error)
}
