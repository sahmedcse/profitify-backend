package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

// tickerFundamentalsRepo implements TickerFundamentalsRepository.
type tickerFundamentalsRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewTickerFundamentalsRepo creates a TickerFundamentalsRepository.
func NewTickerFundamentalsRepo(pool *pgxpool.Pool, logger *slog.Logger) TickerFundamentalsRepository {
	return &tickerFundamentalsRepo{pool: pool, logger: logger}
}

// GetLatest returns the most recent fundamentals row for tickerID.
// JSONB columns (address, branding) are decoded into their typed fields.
func (r *tickerFundamentalsRepo) GetLatest(ctx context.Context, tickerID string) (*domain.TickerFundamentals, error) {
	var f domain.TickerFundamentals
	var marketCap, sharesOut, weightedShares, employees *float64
	var addressRaw, brandingRaw []byte

	err := r.pool.QueryRow(ctx, `
		SELECT id, ticker_id, time,
		       market_cap,
		       shares_outstanding,
		       weighted_shares_outstanding,
		       sic_code, sic_description, description,
		       homepage_url, phone_number,
		       total_employees,
		       address, branding
		FROM ticker_fundamentals
		WHERE ticker_id = $1
		ORDER BY time DESC
		LIMIT 1`, tickerID).
		Scan(&f.ID, &f.TickerID, &f.Time,
			&marketCap,
			&sharesOut,
			&weightedShares,
			&f.SICCode, &f.SICDescription, &f.Description,
			&f.HomepageURL, &f.PhoneNumber,
			&employees,
			&addressRaw, &brandingRaw)
	if err != nil {
		return nil, fmt.Errorf("tickerFundamentalsRepo.GetLatest: %w", err)
	}
	if marketCap != nil {
		f.MarketCap = *marketCap
	}
	if sharesOut != nil {
		f.SharesOutstanding = int64(*sharesOut)
	}
	if weightedShares != nil {
		f.WeightedSharesOutstanding = int64(*weightedShares)
	}
	if employees != nil {
		f.TotalEmployees = int(*employees)
	}
	if len(addressRaw) > 0 {
		if err := json.Unmarshal(addressRaw, &f.Address); err != nil {
			return nil, fmt.Errorf("tickerFundamentalsRepo.GetLatest decoding address: %w", err)
		}
	}
	if len(brandingRaw) > 0 {
		if err := json.Unmarshal(brandingRaw, &f.Branding); err != nil {
			return nil, fmt.Errorf("tickerFundamentalsRepo.GetLatest decoding branding: %w", err)
		}
	}
	return &f, nil
}
