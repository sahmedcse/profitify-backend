package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

// tickerDividendSummaryRepo implements TickerDividendSummaryRepository.
type tickerDividendSummaryRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewTickerDividendSummaryRepo creates a TickerDividendSummaryRepository.
func NewTickerDividendSummaryRepo(pool *pgxpool.Pool, logger *slog.Logger) TickerDividendSummaryRepository {
	return &tickerDividendSummaryRepo{pool: pool, logger: logger}
}

// GetLatest returns the most recent dividend summary row for tickerID.
// Date columns are formatted as YYYY-MM-DD strings to keep the JSON
// payload uniform with the existing domain shape.
func (r *tickerDividendSummaryRepo) GetLatest(ctx context.Context, tickerID string) (*domain.TickerDividendSummary, error) {
	var s domain.TickerDividendSummary
	var nextEx *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT id, ticker_id, time,
		       current_yield, forward_yield, trailing_yield_12m,
		       dividend_growth_rate_1y, dividend_growth_rate_3y, dividend_growth_rate_5y,
		       consecutive_increases,
		       next_ex_dividend_date, days_until_ex_dividend,
		       payout_frequency, latest_distribution_type
		FROM ticker_dividend_summaries
		WHERE ticker_id = $1
		ORDER BY time DESC
		LIMIT 1`, tickerID).
		Scan(&s.ID, &s.TickerID, &s.Time,
			&s.CurrentYield, &s.ForwardYield, &s.TrailingYield12m,
			&s.DividendGrowthRate1y, &s.DividendGrowthRate3y, &s.DividendGrowthRate5y,
			&s.ConsecutiveIncreases,
			&nextEx, &s.DaysUntilExDividend,
			&s.PayoutFrequency, &s.LatestDistributionType)
	if err != nil {
		return nil, fmt.Errorf("tickerDividendSummaryRepo.GetLatest: %w", err)
	}
	if nextEx != nil {
		formatted := nextEx.Format("2006-01-02")
		s.NextExDividendDate = &formatted
	}
	return &s, nil
}
