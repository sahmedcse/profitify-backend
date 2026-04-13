package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

type tickerTechnicalsRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewTickerTechnicalsRepo creates a new TickerTechnicalsRepository.
func NewTickerTechnicalsRepo(pool *pgxpool.Pool, logger *slog.Logger) TickerTechnicalsRepository {
	return &tickerTechnicalsRepo{pool: pool, logger: logger}
}

func (r *tickerTechnicalsRepo) Upsert(ctx context.Context, tech *domain.TechnicalIndicators) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ticker_technicals (
			ticker_id, time,
			sma_20, sma_50, sma_200,
			ema_12, ema_26,
			rsi_14,
			macd_line, macd_signal, macd_histogram,
			bollinger_upper, bollinger_middle, bollinger_lower,
			atr_14, obv
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (ticker_id, time) DO UPDATE SET
			sma_20            = EXCLUDED.sma_20,
			sma_50            = EXCLUDED.sma_50,
			sma_200           = EXCLUDED.sma_200,
			ema_12            = EXCLUDED.ema_12,
			ema_26            = EXCLUDED.ema_26,
			rsi_14            = EXCLUDED.rsi_14,
			macd_line         = EXCLUDED.macd_line,
			macd_signal       = EXCLUDED.macd_signal,
			macd_histogram    = EXCLUDED.macd_histogram,
			bollinger_upper   = EXCLUDED.bollinger_upper,
			bollinger_middle  = EXCLUDED.bollinger_middle,
			bollinger_lower   = EXCLUDED.bollinger_lower,
			atr_14            = EXCLUDED.atr_14,
			obv               = EXCLUDED.obv`,
		tech.TickerID, tech.Time,
		tech.SMA20, tech.SMA50, tech.SMA200,
		tech.EMA12, tech.EMA26,
		tech.RSI14,
		tech.MACDLine, tech.MACDSignal, tech.MACDHistogram,
		tech.BollingerUpper, tech.BollingerMiddle, tech.BollingerLower,
		tech.ATR14, tech.OBV,
	)
	if err != nil {
		return fmt.Errorf("tickerTechnicalsRepo.Upsert: %w", err)
	}
	return nil
}

func (r *tickerTechnicalsRepo) GetLatest(ctx context.Context, tickerID string) (*domain.TechnicalIndicators, error) {
	var tech domain.TechnicalIndicators
	var statusesJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, ticker_id, time,
			sma_20, sma_50, sma_200,
			ema_12, ema_26,
			rsi_14,
			macd_line, macd_signal, macd_histogram,
			bollinger_upper, bollinger_middle, bollinger_lower,
			atr_14, obv, indicator_statuses
		FROM ticker_technicals
		WHERE ticker_id = $1
		ORDER BY time DESC
		LIMIT 1`, tickerID).
		Scan(&tech.ID, &tech.TickerID, &tech.Time,
			&tech.SMA20, &tech.SMA50, &tech.SMA200,
			&tech.EMA12, &tech.EMA26,
			&tech.RSI14,
			&tech.MACDLine, &tech.MACDSignal, &tech.MACDHistogram,
			&tech.BollingerUpper, &tech.BollingerMiddle, &tech.BollingerLower,
			&tech.ATR14, &tech.OBV, &statusesJSON)
	if err != nil {
		return nil, fmt.Errorf("tickerTechnicalsRepo.GetLatest: %w", err)
	}
	if len(statusesJSON) > 0 {
		_ = json.Unmarshal(statusesJSON, &tech.IndicatorStatuses)
	}
	return &tech, nil
}

func (r *tickerTechnicalsRepo) UpdateIndicatorStatuses(ctx context.Context, tickerID string, t time.Time, statuses map[string]string) error {
	statusesJSON, err := json.Marshal(statuses)
	if err != nil {
		return fmt.Errorf("tickerTechnicalsRepo.UpdateIndicatorStatuses: marshaling: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		UPDATE ticker_technicals
		SET indicator_statuses = $1
		WHERE ticker_id = $2 AND time = $3`,
		statusesJSON, tickerID, t)
	if err != nil {
		return fmt.Errorf("tickerTechnicalsRepo.UpdateIndicatorStatuses: %w", err)
	}
	return nil
}
