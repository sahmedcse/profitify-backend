package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

type tickerFundamentalsRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewTickerFundamentalsRepo creates a new TickerFundamentalsRepository backed by the given pool.
func NewTickerFundamentalsRepo(pool *pgxpool.Pool, logger *slog.Logger) TickerFundamentalsRepository {
	return &tickerFundamentalsRepo{pool: pool, logger: logger}
}

func (r *tickerFundamentalsRepo) Upsert(ctx context.Context, f *domain.TickerFundamentals) error {
	addressJSON, err := json.Marshal(f.Address)
	if err != nil {
		return fmt.Errorf("tickerFundamentalsRepo.Upsert: marshalling address: %w", err)
	}
	brandingJSON, err := json.Marshal(f.Branding)
	if err != nil {
		return fmt.Errorf("tickerFundamentalsRepo.Upsert: marshalling branding: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO ticker_fundamentals (
			ticker_id, time, market_cap, shares_outstanding,
			weighted_shares_outstanding, sic_code, sic_description,
			description, homepage_url, phone_number, total_employees,
			address, branding
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (ticker_id, time) DO UPDATE SET
			market_cap                  = EXCLUDED.market_cap,
			shares_outstanding          = EXCLUDED.shares_outstanding,
			weighted_shares_outstanding = EXCLUDED.weighted_shares_outstanding,
			sic_code                    = EXCLUDED.sic_code,
			sic_description             = EXCLUDED.sic_description,
			description                 = EXCLUDED.description,
			homepage_url                = EXCLUDED.homepage_url,
			phone_number                = EXCLUDED.phone_number,
			total_employees             = EXCLUDED.total_employees,
			address                     = EXCLUDED.address,
			branding                    = EXCLUDED.branding`,
		f.TickerID, f.Time, f.MarketCap, f.SharesOutstanding,
		f.WeightedSharesOutstanding, f.SICCode, f.SICDescription,
		f.Description, f.HomepageURL, f.PhoneNumber, f.TotalEmployees,
		addressJSON, brandingJSON,
	)
	if err != nil {
		return fmt.Errorf("tickerFundamentalsRepo.Upsert: %w", err)
	}
	return nil
}

func (r *tickerFundamentalsRepo) GetLatest(ctx context.Context, tickerID string) (*domain.TickerFundamentals, error) {
	var f domain.TickerFundamentals
	var addressJSON, brandingJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, ticker_id, time, market_cap, shares_outstanding,
		       weighted_shares_outstanding, sic_code, sic_description,
		       description, homepage_url, phone_number, total_employees,
		       address, branding
		FROM ticker_fundamentals
		WHERE ticker_id = $1
		ORDER BY time DESC
		LIMIT 1`, tickerID).
		Scan(&f.ID, &f.TickerID, &f.Time, &f.MarketCap, &f.SharesOutstanding,
			&f.WeightedSharesOutstanding, &f.SICCode, &f.SICDescription,
			&f.Description, &f.HomepageURL, &f.PhoneNumber, &f.TotalEmployees,
			&addressJSON, &brandingJSON)
	if err != nil {
		return nil, fmt.Errorf("tickerFundamentalsRepo.GetLatest: %w", err)
	}
	if err := json.Unmarshal(addressJSON, &f.Address); err != nil {
		return nil, fmt.Errorf("tickerFundamentalsRepo.GetLatest: unmarshalling address: %w", err)
	}
	if err := json.Unmarshal(brandingJSON, &f.Branding); err != nil {
		return nil, fmt.Errorf("tickerFundamentalsRepo.GetLatest: unmarshalling branding: %w", err)
	}
	return &f, nil
}
