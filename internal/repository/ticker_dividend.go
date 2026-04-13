package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

type tickerDividendRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewTickerDividendRepo creates a new TickerDividendRepository backed by the given pool.
func NewTickerDividendRepo(pool *pgxpool.Pool, logger *slog.Logger) TickerDividendRepository {
	return &tickerDividendRepo{pool: pool, logger: logger}
}

func (r *tickerDividendRepo) UpsertBatch(ctx context.Context, dividends []domain.TickerDividend) error {
	if len(dividends) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, d := range dividends {
		batch.Queue(`
			INSERT INTO ticker_dividends (
				ticker_id, cash_amount, split_adjusted_cash_amount, currency,
				ex_dividend_date, declaration_date, record_date, pay_date,
				frequency, distribution_type, historical_adjustment_factor
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (ticker_id, ex_dividend_date) DO UPDATE SET
				cash_amount                  = EXCLUDED.cash_amount,
				split_adjusted_cash_amount   = EXCLUDED.split_adjusted_cash_amount,
				currency                     = EXCLUDED.currency,
				declaration_date             = EXCLUDED.declaration_date,
				record_date                  = EXCLUDED.record_date,
				pay_date                     = EXCLUDED.pay_date,
				frequency                    = EXCLUDED.frequency,
				distribution_type            = EXCLUDED.distribution_type,
				historical_adjustment_factor = EXCLUDED.historical_adjustment_factor`,
			d.TickerID, d.CashAmount, d.SplitAdjustedCashAmount, d.Currency,
			d.ExDividendDate, d.DeclarationDate, d.RecordDate, d.PayDate,
			d.Frequency, d.DistributionType, d.HistoricalAdjustmentFactor,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }()

	for range dividends {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("tickerDividendRepo.UpsertBatch: %w", err)
		}
	}
	return nil
}
