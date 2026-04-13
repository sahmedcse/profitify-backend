package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

type tickerDividendSummaryRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewTickerDividendSummaryRepo creates a new TickerDividendSummaryRepository backed by the given pool.
func NewTickerDividendSummaryRepo(pool *pgxpool.Pool, logger *slog.Logger) TickerDividendSummaryRepository {
	return &tickerDividendSummaryRepo{pool: pool, logger: logger}
}

func (r *tickerDividendSummaryRepo) Upsert(ctx context.Context, s *domain.TickerDividendSummary) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ticker_dividend_summaries (
			ticker_id, time, current_yield, forward_yield, trailing_yield_12m,
			dividend_growth_rate_1y, dividend_growth_rate_3y, dividend_growth_rate_5y,
			consecutive_increases, next_ex_dividend_date, days_until_ex_dividend,
			payout_frequency, latest_distribution_type
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (ticker_id, time) DO UPDATE SET
			current_yield            = EXCLUDED.current_yield,
			forward_yield            = EXCLUDED.forward_yield,
			trailing_yield_12m       = EXCLUDED.trailing_yield_12m,
			dividend_growth_rate_1y  = EXCLUDED.dividend_growth_rate_1y,
			dividend_growth_rate_3y  = EXCLUDED.dividend_growth_rate_3y,
			dividend_growth_rate_5y  = EXCLUDED.dividend_growth_rate_5y,
			consecutive_increases    = EXCLUDED.consecutive_increases,
			next_ex_dividend_date    = EXCLUDED.next_ex_dividend_date,
			days_until_ex_dividend   = EXCLUDED.days_until_ex_dividend,
			payout_frequency         = EXCLUDED.payout_frequency,
			latest_distribution_type = EXCLUDED.latest_distribution_type`,
		s.TickerID, s.Time, s.CurrentYield, s.ForwardYield, s.TrailingYield12m,
		s.DividendGrowthRate1y, s.DividendGrowthRate3y, s.DividendGrowthRate5y,
		s.ConsecutiveIncreases, s.NextExDividendDate, s.DaysUntilExDividend,
		s.PayoutFrequency, s.LatestDistributionType,
	)
	if err != nil {
		return fmt.Errorf("tickerDividendSummaryRepo.Upsert: %w", err)
	}
	return nil
}

func (r *tickerDividendSummaryRepo) GetLatest(ctx context.Context, tickerID string) (*domain.TickerDividendSummary, error) {
	var s domain.TickerDividendSummary
	err := r.pool.QueryRow(ctx, `
		SELECT id, ticker_id, time, current_yield, forward_yield, trailing_yield_12m,
		       dividend_growth_rate_1y, dividend_growth_rate_3y, dividend_growth_rate_5y,
		       consecutive_increases, next_ex_dividend_date, days_until_ex_dividend,
		       payout_frequency, latest_distribution_type
		FROM ticker_dividend_summaries
		WHERE ticker_id = $1
		ORDER BY time DESC
		LIMIT 1`, tickerID).
		Scan(&s.ID, &s.TickerID, &s.Time, &s.CurrentYield, &s.ForwardYield, &s.TrailingYield12m,
			&s.DividendGrowthRate1y, &s.DividendGrowthRate3y, &s.DividendGrowthRate5y,
			&s.ConsecutiveIncreases, &s.NextExDividendDate, &s.DaysUntilExDividend,
			&s.PayoutFrequency, &s.LatestDistributionType)
	if err != nil {
		return nil, fmt.Errorf("tickerDividendSummaryRepo.GetLatest: %w", err)
	}
	return &s, nil
}
