package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

// tickerStatsRepo implements TickerStatsRepository.
type tickerStatsRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewTickerStatsRepo creates a TickerStatsRepository.
func NewTickerStatsRepo(pool *pgxpool.Pool, logger *slog.Logger) TickerStatsRepository {
	return &tickerStatsRepo{pool: pool, logger: logger}
}

// GetLatest returns the most recent rolling stats row for tickerID.
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
