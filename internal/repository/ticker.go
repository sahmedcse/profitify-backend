package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

// tickerRepo implements TickerRepository using pgx.
type tickerRepo struct {
	pool *pgxpool.Pool
}

// NewTickerRepo creates a new TickerRepository backed by the given connection pool.
func NewTickerRepo(pool *pgxpool.Pool) TickerRepository {
	return &tickerRepo{pool: pool}
}

func (r *tickerRepo) UpsertBatch(ctx context.Context, tickers []domain.Ticker) error {
	if len(tickers) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	const query = `
		INSERT INTO tickers (ticker, name, market, primary_exchange, type, active, currency_name, locale, cik, list_date, delisted_utc)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (ticker) DO UPDATE SET
			name             = EXCLUDED.name,
			market           = EXCLUDED.market,
			primary_exchange = EXCLUDED.primary_exchange,
			type             = EXCLUDED.type,
			active           = EXCLUDED.active,
			currency_name    = EXCLUDED.currency_name,
			locale           = EXCLUDED.locale,
			cik              = EXCLUDED.cik,
			list_date        = EXCLUDED.list_date,
			delisted_utc     = EXCLUDED.delisted_utc,
			updated_at       = NOW()`

	for _, t := range tickers {
		batch.Queue(query,
			t.Ticker, t.Name, t.Market, t.PrimaryExchange, t.Type,
			t.Active, t.CurrencyName, t.Locale, t.CIK, t.ListDate, t.DelistedUTC,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(tickers); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("tickerRepo.UpsertBatch: row %d (%s): %w", i, tickers[i].Ticker, err)
		}
	}
	return nil
}

func (r *tickerRepo) GetActive(ctx context.Context) ([]domain.Ticker, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, ticker, name, market, primary_exchange, type, active,
		       currency_name, locale, cik, list_date, delisted_utc,
		       created_at, updated_at
		FROM tickers
		WHERE active = TRUE
		ORDER BY ticker`)
	if err != nil {
		return nil, fmt.Errorf("tickerRepo.GetActive: %w", err)
	}
	defer rows.Close()

	return scanTickers(rows)
}

func (r *tickerRepo) GetBySymbol(ctx context.Context, symbol string) (*domain.Ticker, error) {
	var t domain.Ticker
	err := r.pool.QueryRow(ctx, `
		SELECT id, ticker, name, market, primary_exchange, type, active,
		       currency_name, locale, cik, list_date, delisted_utc,
		       created_at, updated_at
		FROM tickers
		WHERE ticker = $1`, symbol).
		Scan(&t.ID, &t.Ticker, &t.Name, &t.Market, &t.PrimaryExchange,
			&t.Type, &t.Active, &t.CurrencyName, &t.Locale, &t.CIK,
			&t.ListDate, &t.DelistedUTC, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("tickerRepo.GetBySymbol: %w", err)
	}
	return &t, nil
}

func scanTickers(rows pgx.Rows) ([]domain.Ticker, error) {
	var tickers []domain.Ticker
	for rows.Next() {
		var t domain.Ticker
		if err := rows.Scan(
			&t.ID, &t.Ticker, &t.Name, &t.Market, &t.PrimaryExchange,
			&t.Type, &t.Active, &t.CurrencyName, &t.Locale, &t.CIK,
			&t.ListDate, &t.DelistedUTC, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("tickerRepo.scanTickers: %w", err)
		}
		tickers = append(tickers, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("tickerRepo.scanTickers: %w", err)
	}
	return tickers, nil
}
