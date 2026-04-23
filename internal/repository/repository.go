package repository

import (
	"context"
	"time"

	"github.com/profitify/profitify-backend/internal/domain"
)

// TickerRepository persists and retrieves ticker metadata.
type TickerRepository interface {
	UpsertBatch(ctx context.Context, tickers []domain.Ticker) error
	GetActive(ctx context.Context) ([]domain.Ticker, error)
	GetBySymbol(ctx context.Context, symbol string) (*domain.Ticker, error)
	UpdateSector(ctx context.Context, tickerID string, sector string) error
}

// DailyPriceRepository persists and retrieves OHLCV bars.
type DailyPriceRepository interface {
	Upsert(ctx context.Context, price *domain.DailyPrice) error
	GetByTickerAndDateRange(ctx context.Context, tickerID string, from, to time.Time) ([]domain.DailyPrice, error)
	GetLatest(ctx context.Context, tickerID string) (*domain.DailyPrice, error)
}

// TickerTechnicalsRepository persists and retrieves technical indicators.
type TickerTechnicalsRepository interface {
	Upsert(ctx context.Context, tech *domain.TechnicalIndicators) error
	GetLatest(ctx context.Context, tickerID string) (*domain.TechnicalIndicators, error)
	UpdateIndicatorStatuses(ctx context.Context, tickerID string, t time.Time, statuses map[string]string) error
}

// TickerFundamentalsRepository persists and retrieves company fundamentals.
type TickerFundamentalsRepository interface {
	Upsert(ctx context.Context, f *domain.TickerFundamentals) error
	GetLatest(ctx context.Context, tickerID string) (*domain.TickerFundamentals, error)
}

// TickerDividendRepository persists dividend events.
type TickerDividendRepository interface {
	UpsertBatch(ctx context.Context, dividends []domain.TickerDividend) error
}

// TickerDividendSummaryRepository persists and retrieves computed dividend summaries.
type TickerDividendSummaryRepository interface {
	Upsert(ctx context.Context, s *domain.TickerDividendSummary) error
	GetLatest(ctx context.Context, tickerID string) (*domain.TickerDividendSummary, error)
}

// TickerStatsRepository persists and retrieves rolling stats.
type TickerStatsRepository interface {
	Upsert(ctx context.Context, stats *domain.TickerStats) error
	GetLatest(ctx context.Context, tickerID string) (*domain.TickerStats, error)
}

// PipelineRunRepository persists and retrieves pipeline run records.
type PipelineRunRepository interface {
	Create(ctx context.Context, run *domain.PipelineRun) (*domain.PipelineRun, error)
	GetByID(ctx context.Context, id string) (*domain.PipelineRun, error)
	UpdateStatus(ctx context.Context, id string, status string, errorMessage string) error
	UpdateCounts(ctx context.Context, id string) error
	MarkCompleted(ctx context.Context, id string) error
}

// PipelineTickerStageRepository persists and retrieves per-ticker stage records.
type PipelineTickerStageRepository interface {
	BulkInsert(ctx context.Context, stages []domain.PipelineTickerStage) error
	MarkRunning(ctx context.Context, runID, tickerID, stage string) (string, error)
	MarkCompleted(ctx context.Context, runID, tickerID, stage string) error
	MarkFailed(ctx context.Context, runID, tickerID, stage, errorMessage string) error
	GetByRunAndTicker(ctx context.Context, runID, tickerID string) ([]domain.PipelineTickerStage, error)
}
