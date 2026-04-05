package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/repository"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// testPool returns a pgxpool connected to the test database.
// Skips the test if DATABASE_URL is not set.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

// cleanTickers removes all rows from the tickers table.
func cleanTickers(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "DELETE FROM tickers")
	if err != nil {
		t.Fatalf("failed to clean tickers: %v", err)
	}
}

// stubFetcher returns a fixed list of tickers.
type stubFetcher struct {
	tickers []domain.Ticker
	err     error
}

func (f *stubFetcher) FetchActiveTickers(_ context.Context) ([]domain.Ticker, error) {
	return f.tickers, f.err
}

func TestFetchAndUpsert_Integration(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	repo := repository.NewTickerRepo(pool, discardLogger)
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL", Name: "Apple Inc.", Market: "stocks", Active: true, Type: "CS"},
			{Ticker: "MSFT", Name: "Microsoft Corporation", Market: "stocks", Active: true, Type: "CS"},
			{Ticker: "GOOG", Name: "Alphabet Inc.", Market: "stocks", Active: true, Type: "CS"},
		},
	}

	resp, err := fetchAndUpsert(context.Background(), fetcher, repo, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndUpsert: %v", err)
	}

	if resp.TickerCount != 3 {
		t.Errorf("TickerCount = %d, want 3", resp.TickerCount)
	}
	if resp.Date == "" {
		t.Error("Date should not be empty")
	}

	// Verify data persisted to DB
	got, err := repo.GetActive(context.Background())
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 tickers in DB, got %d", len(got))
	}
}

func TestFetchAndUpsert_UpsertOverwrite(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	repo := repository.NewTickerRepo(pool, discardLogger)
	ctx := context.Background()

	// First run: insert 2 tickers
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL", Name: "Apple Inc.", Market: "stocks", Active: true, Type: "CS"},
			{Ticker: "MSFT", Name: "Microsoft", Market: "stocks", Active: true, Type: "CS"},
		},
	}
	resp, err := fetchAndUpsert(ctx, fetcher, repo, discardLogger)
	if err != nil {
		t.Fatalf("first fetchAndUpsert: %v", err)
	}
	if resp.TickerCount != 2 {
		t.Errorf("first run TickerCount = %d, want 2", resp.TickerCount)
	}

	// Second run: update AAPL name, add GOOG
	fetcher.tickers = []domain.Ticker{
		{Ticker: "AAPL", Name: "Apple Inc. (Updated)", Market: "stocks", Active: true, Type: "CS"},
		{Ticker: "MSFT", Name: "Microsoft", Market: "stocks", Active: true, Type: "CS"},
		{Ticker: "GOOG", Name: "Alphabet", Market: "stocks", Active: true, Type: "CS"},
	}
	resp, err = fetchAndUpsert(ctx, fetcher, repo, discardLogger)
	if err != nil {
		t.Fatalf("second fetchAndUpsert: %v", err)
	}
	if resp.TickerCount != 3 {
		t.Errorf("second run TickerCount = %d, want 3", resp.TickerCount)
	}

	// Verify AAPL was updated, not duplicated
	got, err := repo.GetBySymbol(ctx, "AAPL")
	if err != nil {
		t.Fatalf("GetBySymbol: %v", err)
	}
	if got.Name != "Apple Inc. (Updated)" {
		t.Errorf("AAPL Name = %q, want %q", got.Name, "Apple Inc. (Updated)")
	}

	all, err := repo.GetActive(ctx)
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 total tickers, got %d", len(all))
	}
}

func TestFetchAndUpsert_EmptyFetch(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	repo := repository.NewTickerRepo(pool, discardLogger)
	fetcher := &stubFetcher{tickers: nil}

	resp, err := fetchAndUpsert(context.Background(), fetcher, repo, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndUpsert: %v", err)
	}
	if resp.TickerCount != 0 {
		t.Errorf("TickerCount = %d, want 0", resp.TickerCount)
	}
}

func TestFetchAndUpsert_FetchError(t *testing.T) {
	pool := testPool(t)

	repo := repository.NewTickerRepo(pool, discardLogger)
	fetcher := &stubFetcher{err: fmt.Errorf("api timeout")}

	_, err := fetchAndUpsert(context.Background(), fetcher, repo, discardLogger)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHandleRequest_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("MASSIVE_API_KEY", "test-key")

	_, err := handleRequest(context.Background())
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
}

func TestHandleRequest_MissingAPIKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("MASSIVE_API_KEY", "")

	_, err := handleRequest(context.Background())
	if err == nil {
		t.Fatal("expected error for missing MASSIVE_API_KEY")
	}
}
