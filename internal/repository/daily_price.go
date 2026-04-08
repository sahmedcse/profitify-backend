package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

// dailyPriceRepo implements DailyPriceRepository using pgx.
type dailyPriceRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewDailyPriceRepo creates a new DailyPriceRepository backed by the
// given connection pool.
func NewDailyPriceRepo(pool *pgxpool.Pool, logger *slog.Logger) DailyPriceRepository {
	return &dailyPriceRepo{pool: pool, logger: logger}
}

// GetRange returns OHLCV bars for tickerID where time is in [from, to],
// ordered chronologically. The bounds are inclusive.
func (r *dailyPriceRepo) GetRange(ctx context.Context, tickerID string, from, to time.Time) ([]domain.DailyPrice, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, ticker_id, time,
		       COALESCE(open, 0)::float8,
		       COALESCE(high, 0)::float8,
		       COALESCE(low, 0)::float8,
		       COALESCE(close, 0)::float8,
		       COALESCE(volume, 0)::float8,
		       COALESCE(vwap, 0)::float8,
		       COALESCE(pre_market, 0)::float8,
		       COALESCE(after_hours, 0)::float8,
		       otc
		FROM daily_prices
		WHERE ticker_id = $1
		  AND time >= $2
		  AND time <= $3
		ORDER BY time ASC`, tickerID, from, to)
	if err != nil {
		return nil, fmt.Errorf("dailyPriceRepo.GetRange: %w", err)
	}
	defer rows.Close()

	out := []domain.DailyPrice{}
	for rows.Next() {
		var p domain.DailyPrice
		if err := rows.Scan(
			&p.ID, &p.TickerID, &p.Time,
			&p.Open, &p.High, &p.Low, &p.Close, &p.Volume,
			&p.VWAP, &p.PreMarket, &p.AfterHours, &p.OTC,
		); err != nil {
			return nil, fmt.Errorf("dailyPriceRepo.GetRange scan: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dailyPriceRepo.GetRange: %w", err)
	}
	return out, nil
}
