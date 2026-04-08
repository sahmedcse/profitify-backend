package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/repository"
)

// seedTicker inserts a single row into tickers and returns its id.
func seedTicker(t *testing.T, pool *pgxpool.Pool, symbol, name, sector string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO tickers (ticker, name, market, type, active, sector)
		VALUES ($1, $2, 'stocks', 'CS', TRUE, $3)
		RETURNING id`, symbol, name, sector).Scan(&id)
	if err != nil {
		t.Fatalf("seedTicker(%s): %v", symbol, err)
	}
	return id
}

func seedDailyPrice(t *testing.T, pool *pgxpool.Pool, tickerID string, ts time.Time, open, high, low, close, vol float64) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO daily_prices (ticker_id, time, open, high, low, close, volume)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		tickerID, ts, open, high, low, close, vol)
	if err != nil {
		t.Fatalf("seedDailyPrice: %v", err)
	}
}

func seedTechnicals(t *testing.T, pool *pgxpool.Pool, tickerID string, ts time.Time, rsi float64, statuses string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ticker_technicals (ticker_id, time, rsi_14, sma_20, ema_12, macd_line, macd_signal,
		                               bollinger_upper, bollinger_middle, bollinger_lower, indicator_statuses)
		VALUES ($1, $2, $3, 100, 102, 1.5, 1.0, 110, 100, 90, $4::jsonb)`,
		tickerID, ts, rsi, statuses)
	if err != nil {
		t.Fatalf("seedTechnicals: %v", err)
	}
}

func seedStats(t *testing.T, pool *pgxpool.Pool, tickerID string, ts time.Time, change, changePct float64, label string, strength int, pivots string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ticker_stats (ticker_id, time, price_change, price_change_pct,
		                          signal_label, signal_strength, pivot_levels)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)`,
		tickerID, ts, change, changePct, label, strength, pivots)
	if err != nil {
		t.Fatalf("seedStats: %v", err)
	}
}

func seedFundamentals(t *testing.T, pool *pgxpool.Pool, tickerID string, ts time.Time, marketCap float64, sicCode string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ticker_fundamentals (ticker_id, time, market_cap, sic_code, sic_description, description, total_employees)
		VALUES ($1, $2, $3, $4, 'desc', 'company', 1000)`,
		tickerID, ts, marketCap, sicCode)
	if err != nil {
		t.Fatalf("seedFundamentals: %v", err)
	}
}

func seedDividendSummary(t *testing.T, pool *pgxpool.Pool, tickerID string, ts time.Time, yield float64) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ticker_dividend_summaries (ticker_id, time, current_yield)
		VALUES ($1, $2, $3)`, tickerID, ts, yield)
	if err != nil {
		t.Fatalf("seedDividendSummary: %v", err)
	}
}

func TestListForDashboard(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	aaplID := seedTicker(t, pool, "AAPL", "Apple Inc.", domain.SectorTechnology)
	jpmID := seedTicker(t, pool, "JPM", "JPMorgan", domain.SectorFinancial)

	now := time.Now().UTC().Truncate(time.Hour)
	seedDailyPrice(t, pool, aaplID, now, 150, 155, 149, 152, 1000000)
	seedDailyPrice(t, pool, jpmID, now, 200, 205, 199, 201, 800000)

	seedStats(t, pool, aaplID, now, 2.5, 1.6, "Bullish", 70, `{}`)
	seedStats(t, pool, jpmID, now, -0.8, -0.4, "Neutral", 50, `{}`)

	repo := repository.NewTickerRepo(pool, discardLogger)
	ctx := context.Background()

	t.Run("returns all tickers with joined stats", func(t *testing.T) {
		got, err := repo.ListForDashboard(ctx, repository.DashboardFilter{})
		if err != nil {
			t.Fatalf("ListForDashboard: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(got))
		}
		// Sorted by symbol: AAPL first
		if got[0].Symbol != "AAPL" {
			t.Errorf("first symbol = %q, want AAPL", got[0].Symbol)
		}
		if got[0].LatestPrice != 152 {
			t.Errorf("AAPL latest_price = %v, want 152", got[0].LatestPrice)
		}
		if got[0].SignalLabel != "Bullish" {
			t.Errorf("AAPL signal_label = %q, want Bullish", got[0].SignalLabel)
		}
		if got[0].SignalStrength != 70 {
			t.Errorf("AAPL signal_strength = %d, want 70", got[0].SignalStrength)
		}
		if got[0].Sector != domain.SectorTechnology {
			t.Errorf("AAPL sector = %q, want Technology", got[0].Sector)
		}
	})

	t.Run("filters by sector", func(t *testing.T) {
		got, err := repo.ListForDashboard(ctx, repository.DashboardFilter{Sector: domain.SectorFinancial})
		if err != nil {
			t.Fatalf("ListForDashboard: %v", err)
		}
		if len(got) != 1 || got[0].Symbol != "JPM" {
			t.Errorf("got %+v, want [JPM]", got)
		}
	})

	t.Run("filters by search substring", func(t *testing.T) {
		got, err := repo.ListForDashboard(ctx, repository.DashboardFilter{Search: "apple"})
		if err != nil {
			t.Fatalf("ListForDashboard: %v", err)
		}
		if len(got) != 1 || got[0].Symbol != "AAPL" {
			t.Errorf("got %+v, want [AAPL]", got)
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		got, err := repo.ListForDashboard(ctx, repository.DashboardFilter{Limit: 1})
		if err != nil {
			t.Fatalf("ListForDashboard: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("len = %d, want 1", len(got))
		}
	})
}

func TestDailyPriceRepo_GetRange(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	id := seedTicker(t, pool, "AAPL", "Apple", domain.SectorTechnology)
	day := func(d int) time.Time {
		return time.Date(2026, 4, d, 0, 0, 0, 0, time.UTC)
	}
	seedDailyPrice(t, pool, id, day(1), 150, 155, 148, 152, 1000)
	seedDailyPrice(t, pool, id, day(2), 152, 158, 151, 156, 1100)
	seedDailyPrice(t, pool, id, day(3), 156, 160, 155, 159, 1200)
	seedDailyPrice(t, pool, id, day(4), 159, 162, 157, 161, 1300)

	repo := repository.NewDailyPriceRepo(pool, discardLogger)
	got, err := repo.GetRange(context.Background(), id, day(2), day(3))
	if err != nil {
		t.Fatalf("GetRange: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Close != 156 || got[1].Close != 159 {
		t.Errorf("closes = [%v %v], want [156 159]", got[0].Close, got[1].Close)
	}
}

func TestTickerTechnicalsRepo_GetLatest(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	id := seedTicker(t, pool, "AAPL", "Apple", domain.SectorTechnology)
	now := time.Now().UTC().Truncate(time.Hour)
	seedTechnicals(t, pool, id, now.Add(-24*time.Hour), 50, `{"rsi_14":"neutral"}`)
	seedTechnicals(t, pool, id, now, 72, `{"rsi_14":"bullish","macd":"bullish"}`)

	repo := repository.NewTickerTechnicalsRepo(pool, discardLogger)
	got, err := repo.GetLatest(context.Background(), id)
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if got.RSI14 == nil || *got.RSI14 != 72 {
		t.Errorf("RSI14 = %v, want 72", got.RSI14)
	}
	if got.IndicatorStatuses["rsi_14"] != "bullish" {
		t.Errorf("indicator_statuses[rsi_14] = %q, want bullish", got.IndicatorStatuses["rsi_14"])
	}
	if got.IndicatorStatuses["macd"] != "bullish" {
		t.Errorf("indicator_statuses[macd] = %q, want bullish", got.IndicatorStatuses["macd"])
	}
}

func TestTickerTechnicalsRepo_GetLatest_NotFound(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	id := seedTicker(t, pool, "AAPL", "Apple", "")
	repo := repository.NewTickerTechnicalsRepo(pool, discardLogger)
	_, err := repo.GetLatest(context.Background(), id)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("err = %v, want pgx.ErrNoRows", err)
	}
}

func TestTickerStatsRepo_GetLatest(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	id := seedTicker(t, pool, "AAPL", "Apple", domain.SectorTechnology)
	now := time.Now().UTC().Truncate(time.Hour)
	seedStats(t, pool, id, now, 1.0, 0.5, "Bullish", 65,
		`{"R1":{"price":110,"strength":"strong"}}`)

	repo := repository.NewTickerStatsRepo(pool, discardLogger)
	got, err := repo.GetLatest(context.Background(), id)
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if got.SignalLabel != "Bullish" {
		t.Errorf("SignalLabel = %q, want Bullish", got.SignalLabel)
	}
	if got.SignalStrength != 65 {
		t.Errorf("SignalStrength = %d, want 65", got.SignalStrength)
	}
	if len(got.PivotLevels) == 0 {
		t.Error("PivotLevels is empty, want JSON content")
	}
}

func TestTickerFundamentalsRepo_GetLatest(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	id := seedTicker(t, pool, "AAPL", "Apple", domain.SectorTechnology)
	now := time.Now().UTC().Truncate(time.Hour)
	seedFundamentals(t, pool, id, now, 3.5e12, "7372")

	repo := repository.NewTickerFundamentalsRepo(pool, discardLogger)
	got, err := repo.GetLatest(context.Background(), id)
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if got.MarketCap != 3.5e12 {
		t.Errorf("MarketCap = %v, want 3.5e12", got.MarketCap)
	}
	if got.SICCode != "7372" {
		t.Errorf("SICCode = %q, want 7372", got.SICCode)
	}
}

func TestTickerDividendSummaryRepo_GetLatest(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	id := seedTicker(t, pool, "AAPL", "Apple", domain.SectorTechnology)
	now := time.Now().UTC().Truncate(time.Hour)
	seedDividendSummary(t, pool, id, now, 0.55)

	repo := repository.NewTickerDividendSummaryRepo(pool, discardLogger)
	got, err := repo.GetLatest(context.Background(), id)
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if got.CurrentYield == nil || *got.CurrentYield != 0.55 {
		t.Errorf("CurrentYield = %v, want 0.55", got.CurrentYield)
	}
}
