package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

// tickerRepo implements TickerRepository using pgx.
type tickerRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewTickerRepo creates a new TickerRepository backed by the given connection pool.
func NewTickerRepo(pool *pgxpool.Pool, logger *slog.Logger) TickerRepository {
	return &tickerRepo{pool: pool, logger: logger}
}

func (r *tickerRepo) UpsertBatch(ctx context.Context, tickers []domain.Ticker) error {
	if len(tickers) == 0 {
		return nil
	}

	r.logger.Info("upserting tickers", "count", len(tickers))

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
	defer func() { _ = br.Close() }()

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
		       currency_name, locale, cik, list_date, delisted_utc, sector,
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
		       currency_name, locale, cik, list_date, delisted_utc, sector,
		       created_at, updated_at
		FROM tickers
		WHERE ticker = $1`, symbol).
		Scan(&t.ID, &t.Ticker, &t.Name, &t.Market, &t.PrimaryExchange,
			&t.Type, &t.Active, &t.CurrencyName, &t.Locale, &t.CIK,
			&t.ListDate, &t.DelistedUTC, &t.Sector, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("tickerRepo.GetBySymbol: %w", err)
	}
	return &t, nil
}

// ListForDashboard returns a sidebar-ready row per active ticker. It joins
// the latest ticker_stats row (by time DESC) so the sidebar gets the most
// recent close, change, and signal in a single query. The optional sector
// and search filters are applied in SQL.
func (r *tickerRepo) ListForDashboard(ctx context.Context, filter DashboardFilter) ([]DashboardTicker, error) {
	const baseQuery = `
		SELECT
			t.id,
			t.ticker,
			t.name,
			t.sector,
			COALESCE(latest_price.close, 0)::float8 AS latest_price,
			COALESCE(s.price_change, 0)            AS price_change,
			COALESCE(s.price_change_pct, 0)        AS price_change_pct,
			COALESCE(s.signal_label, '')           AS signal_label,
			COALESCE(s.signal_strength, 0)         AS signal_strength
		FROM tickers t
		LEFT JOIN LATERAL (
			SELECT close
			FROM daily_prices dp
			WHERE dp.ticker_id = t.id
			ORDER BY dp.time DESC
			LIMIT 1
		) latest_price ON TRUE
		LEFT JOIN LATERAL (
			SELECT price_change, price_change_pct, signal_label, signal_strength
			FROM ticker_stats ts
			WHERE ts.ticker_id = t.id
			ORDER BY ts.time DESC
			LIMIT 1
		) s ON TRUE
		WHERE t.active = TRUE`

	args := []any{}
	q := baseQuery
	if filter.Sector != "" {
		args = append(args, filter.Sector)
		q += fmt.Sprintf(" AND t.sector = $%d", len(args))
	}
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		q += fmt.Sprintf(" AND (t.ticker ILIKE $%d OR t.name ILIKE $%d)", len(args), len(args))
	}
	q += " ORDER BY t.ticker"
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		q += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("tickerRepo.ListForDashboard: %w", err)
	}
	defer rows.Close()

	out := []DashboardTicker{}
	for rows.Next() {
		var d DashboardTicker
		if err := rows.Scan(
			&d.ID, &d.Symbol, &d.Name, &d.Sector,
			&d.LatestPrice, &d.PriceChange, &d.PriceChangePct,
			&d.SignalLabel, &d.SignalStrength,
		); err != nil {
			return nil, fmt.Errorf("tickerRepo.ListForDashboard scan: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("tickerRepo.ListForDashboard: %w", err)
	}
	return out, nil
}

func scanTickers(rows pgx.Rows) ([]domain.Ticker, error) {
	var tickers []domain.Ticker
	for rows.Next() {
		var t domain.Ticker
		if err := rows.Scan(
			&t.ID, &t.Ticker, &t.Name, &t.Market, &t.PrimaryExchange,
			&t.Type, &t.Active, &t.CurrencyName, &t.Locale, &t.CIK,
			&t.ListDate, &t.DelistedUTC, &t.Sector, &t.CreatedAt, &t.UpdatedAt,
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
