package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

type tickerStatsRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewTickerStatsRepo creates a new TickerStatsRepository backed by the given pool.
func NewTickerStatsRepo(pool *pgxpool.Pool, logger *slog.Logger) TickerStatsRepository {
	return &tickerStatsRepo{pool: pool, logger: logger}
}

func (r *tickerStatsRepo) Upsert(ctx context.Context, s *domain.TickerStats) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ticker_stats (
			ticker_id, time,
			price_change, price_change_pct, volume_change_pct,
			day_range, gap_pct, relative_volume,
			price_return_7d, dividend_return_7d, total_return_7d,
			volatility_7d, avg_volume_7d, max_drawdown_7d,
			price_return_30d, dividend_return_30d, total_return_30d,
			volatility_30d, avg_volume_30d, max_drawdown_30d,
			price_return_90d, dividend_return_90d, total_return_90d,
			volatility_90d, avg_volume_90d, max_drawdown_90d,
			high_52w, low_52w, dist_from_high_52w_pct, dist_from_low_52w_pct,
			signal_label, signal_strength, pivot_levels
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26,
			$27, $28, $29, $30,
			$31, $32, $33
		)
		ON CONFLICT (ticker_id, time) DO UPDATE SET
			price_change            = EXCLUDED.price_change,
			price_change_pct        = EXCLUDED.price_change_pct,
			volume_change_pct       = EXCLUDED.volume_change_pct,
			day_range               = EXCLUDED.day_range,
			gap_pct                 = EXCLUDED.gap_pct,
			relative_volume         = EXCLUDED.relative_volume,
			price_return_7d         = EXCLUDED.price_return_7d,
			dividend_return_7d      = EXCLUDED.dividend_return_7d,
			total_return_7d         = EXCLUDED.total_return_7d,
			volatility_7d           = EXCLUDED.volatility_7d,
			avg_volume_7d           = EXCLUDED.avg_volume_7d,
			max_drawdown_7d         = EXCLUDED.max_drawdown_7d,
			price_return_30d        = EXCLUDED.price_return_30d,
			dividend_return_30d     = EXCLUDED.dividend_return_30d,
			total_return_30d        = EXCLUDED.total_return_30d,
			volatility_30d          = EXCLUDED.volatility_30d,
			avg_volume_30d          = EXCLUDED.avg_volume_30d,
			max_drawdown_30d        = EXCLUDED.max_drawdown_30d,
			price_return_90d        = EXCLUDED.price_return_90d,
			dividend_return_90d     = EXCLUDED.dividend_return_90d,
			total_return_90d        = EXCLUDED.total_return_90d,
			volatility_90d          = EXCLUDED.volatility_90d,
			avg_volume_90d          = EXCLUDED.avg_volume_90d,
			max_drawdown_90d        = EXCLUDED.max_drawdown_90d,
			high_52w                = EXCLUDED.high_52w,
			low_52w                 = EXCLUDED.low_52w,
			dist_from_high_52w_pct  = EXCLUDED.dist_from_high_52w_pct,
			dist_from_low_52w_pct   = EXCLUDED.dist_from_low_52w_pct,
			signal_label            = EXCLUDED.signal_label,
			signal_strength         = EXCLUDED.signal_strength,
			pivot_levels            = EXCLUDED.pivot_levels`,
		s.TickerID, s.Time,
		s.PriceChange, s.PriceChangePct, s.VolumeChangePct,
		s.DayRange, s.GapPct, s.RelativeVolume,
		s.PriceReturn7d, s.DividendReturn7d, s.TotalReturn7d,
		s.Volatility7d, s.AvgVolume7d, s.MaxDrawdown7d,
		s.PriceReturn30d, s.DividendReturn30d, s.TotalReturn30d,
		s.Volatility30d, s.AvgVolume30d, s.MaxDrawdown30d,
		s.PriceReturn90d, s.DividendReturn90d, s.TotalReturn90d,
		s.Volatility90d, s.AvgVolume90d, s.MaxDrawdown90d,
		s.High52w, s.Low52w, s.DistFromHigh52wPct, s.DistFromLow52wPct,
		s.SignalLabel, s.SignalStrength, s.PivotLevels,
	)
	if err != nil {
		return fmt.Errorf("tickerStatsRepo.Upsert: %w", err)
	}
	return nil
}

func (r *tickerStatsRepo) GetLatest(ctx context.Context, tickerID string) (*domain.TickerStats, error) {
	var s domain.TickerStats
	err := r.pool.QueryRow(ctx, `
		SELECT id, ticker_id, time,
		       price_change, price_change_pct, volume_change_pct,
		       day_range, gap_pct, relative_volume,
		       price_return_7d, dividend_return_7d, total_return_7d,
		       volatility_7d, avg_volume_7d, max_drawdown_7d,
		       price_return_30d, dividend_return_30d, total_return_30d,
		       volatility_30d, avg_volume_30d, max_drawdown_30d,
		       price_return_90d, dividend_return_90d, total_return_90d,
		       volatility_90d, avg_volume_90d, max_drawdown_90d,
		       high_52w, low_52w, dist_from_high_52w_pct, dist_from_low_52w_pct,
		       signal_label, signal_strength, pivot_levels
		FROM ticker_stats
		WHERE ticker_id = $1
		ORDER BY time DESC
		LIMIT 1`, tickerID).
		Scan(&s.ID, &s.TickerID, &s.Time,
			&s.PriceChange, &s.PriceChangePct, &s.VolumeChangePct,
			&s.DayRange, &s.GapPct, &s.RelativeVolume,
			&s.PriceReturn7d, &s.DividendReturn7d, &s.TotalReturn7d,
			&s.Volatility7d, &s.AvgVolume7d, &s.MaxDrawdown7d,
			&s.PriceReturn30d, &s.DividendReturn30d, &s.TotalReturn30d,
			&s.Volatility30d, &s.AvgVolume30d, &s.MaxDrawdown30d,
			&s.PriceReturn90d, &s.DividendReturn90d, &s.TotalReturn90d,
			&s.Volatility90d, &s.AvgVolume90d, &s.MaxDrawdown90d,
			&s.High52w, &s.Low52w, &s.DistFromHigh52wPct, &s.DistFromLow52wPct,
			&s.SignalLabel, &s.SignalStrength, &s.PivotLevels)
	if err != nil {
		return nil, fmt.Errorf("tickerStatsRepo.GetLatest: %w", err)
	}
	return &s, nil
}
