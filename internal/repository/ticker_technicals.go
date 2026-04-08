package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

// tickerTechnicalsRepo implements TickerTechnicalsRepository using pgx.
type tickerTechnicalsRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewTickerTechnicalsRepo creates a TickerTechnicalsRepository.
func NewTickerTechnicalsRepo(pool *pgxpool.Pool, logger *slog.Logger) TickerTechnicalsRepository {
	return &tickerTechnicalsRepo{pool: pool, logger: logger}
}

// GetLatest returns the most recent technical indicator snapshot for
// tickerID. Returns pgx.ErrNoRows wrapped if no row exists.
func (r *tickerTechnicalsRepo) GetLatest(ctx context.Context, tickerID string) (*domain.TechnicalIndicators, error) {
	var t domain.TechnicalIndicators
	var statusesRaw []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, ticker_id, time,
		       sma_20, sma_50, sma_200,
		       ema_12, ema_26,
		       rsi_14,
		       macd_line, macd_signal, macd_histogram,
		       bollinger_upper, bollinger_middle, bollinger_lower,
		       atr_14, obv,
		       indicator_statuses
		FROM ticker_technicals
		WHERE ticker_id = $1
		ORDER BY time DESC
		LIMIT 1`, tickerID).
		Scan(&t.ID, &t.TickerID, &t.Time,
			&t.SMA20, &t.SMA50, &t.SMA200,
			&t.EMA12, &t.EMA26,
			&t.RSI14,
			&t.MACDLine, &t.MACDSignal, &t.MACDHistogram,
			&t.BollingerUpper, &t.BollingerMiddle, &t.BollingerLower,
			&t.ATR14, &t.OBV,
			&statusesRaw)
	if err != nil {
		return nil, fmt.Errorf("tickerTechnicalsRepo.GetLatest: %w", err)
	}
	if len(statusesRaw) > 0 {
		if err := json.Unmarshal(statusesRaw, &t.IndicatorStatuses); err != nil {
			return nil, fmt.Errorf("tickerTechnicalsRepo.GetLatest decoding indicator_statuses: %w", err)
		}
	}
	return &t, nil
}
