package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

// dailyPriceRepo implements DailyPriceRepository using pgx.
type dailyPriceRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewDailyPriceRepo creates a new DailyPriceRepository backed by the given pool.
func NewDailyPriceRepo(pool *pgxpool.Pool, logger *slog.Logger) DailyPriceRepository {
	return &dailyPriceRepo{pool: pool, logger: logger}
}

func (r *dailyPriceRepo) Upsert(ctx context.Context, price *domain.DailyPrice) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO daily_prices (ticker_id, time, open, high, low, close, volume, vwap, pre_market, after_hours, otc)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (ticker_id, time) DO UPDATE SET
			open        = EXCLUDED.open,
			high        = EXCLUDED.high,
			low         = EXCLUDED.low,
			close       = EXCLUDED.close,
			volume      = EXCLUDED.volume,
			vwap        = EXCLUDED.vwap,
			pre_market  = EXCLUDED.pre_market,
			after_hours = EXCLUDED.after_hours,
			otc         = EXCLUDED.otc`,
		price.TickerID, price.Time,
		price.Open, price.High, price.Low, price.Close, price.Volume,
		price.VWAP, price.PreMarket, price.AfterHours, price.OTC,
	)
	if err != nil {
		return fmt.Errorf("dailyPriceRepo.Upsert: %w", err)
	}
	return nil
}

func (r *dailyPriceRepo) GetByTickerAndDateRange(ctx context.Context, tickerID string, from, to time.Time) ([]domain.DailyPrice, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, ticker_id, time, open, high, low, close, volume, vwap, pre_market, after_hours, otc
		FROM daily_prices
		WHERE ticker_id = $1
		  AND time >= $2
		  AND time <= $3
		ORDER BY time ASC`, tickerID, from, to)
	if err != nil {
		return nil, fmt.Errorf("dailyPriceRepo.GetByTickerAndDateRange: %w", err)
	}
	defer rows.Close()
	return scanDailyPrices(rows)
}

func (r *dailyPriceRepo) GetLatest(ctx context.Context, tickerID string) (*domain.DailyPrice, error) {
	var p domain.DailyPrice
	err := r.pool.QueryRow(ctx, `
		SELECT id, ticker_id, time, open, high, low, close, volume, vwap, pre_market, after_hours, otc
		FROM daily_prices
		WHERE ticker_id = $1
		ORDER BY time DESC
		LIMIT 1`, tickerID).
		Scan(&p.ID, &p.TickerID, &p.Time,
			&p.Open, &p.High, &p.Low, &p.Close, &p.Volume,
			&p.VWAP, &p.PreMarket, &p.AfterHours, &p.OTC)
	if err != nil {
		return nil, fmt.Errorf("dailyPriceRepo.GetLatest: %w", err)
	}
	return &p, nil
}

func scanDailyPrices(rows pgx.Rows) ([]domain.DailyPrice, error) {
	var out []domain.DailyPrice
	for rows.Next() {
		var p domain.DailyPrice
		if err := rows.Scan(
			&p.ID, &p.TickerID, &p.Time,
			&p.Open, &p.High, &p.Low, &p.Close, &p.Volume,
			&p.VWAP, &p.PreMarket, &p.AfterHours, &p.OTC,
		); err != nil {
			return nil, fmt.Errorf("dailyPriceRepo.scan: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dailyPriceRepo.scan: %w", err)
	}
	return out, nil
}
