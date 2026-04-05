package repository_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
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

func TestUpsertBatch_InsertsNewTickers(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	repo := repository.NewTickerRepo(pool, discardLogger)
	ctx := context.Background()

	tickers := []domain.Ticker{
		{Ticker: "AAPL", Name: "Apple Inc.", Market: "stocks", Active: true, Type: "CS"},
		{Ticker: "MSFT", Name: "Microsoft Corporation", Market: "stocks", Active: true, Type: "CS"},
	}

	err := repo.UpsertBatch(ctx, tickers)
	if err != nil {
		t.Fatalf("UpsertBatch: %v", err)
	}

	// Verify rows exist
	got, err := repo.GetActive(ctx)
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tickers, got %d", len(got))
	}
	if got[0].Ticker != "AAPL" {
		t.Errorf("first ticker = %q, want AAPL", got[0].Ticker)
	}
	if got[1].Ticker != "MSFT" {
		t.Errorf("second ticker = %q, want MSFT", got[1].Ticker)
	}
}

func TestUpsertBatch_UpdatesOnConflict(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	repo := repository.NewTickerRepo(pool, discardLogger)
	ctx := context.Background()

	// Insert initial
	err := repo.UpsertBatch(ctx, []domain.Ticker{
		{Ticker: "AAPL", Name: "Apple Inc.", Market: "stocks", Active: true},
	})
	if err != nil {
		t.Fatalf("first UpsertBatch: %v", err)
	}

	// Upsert with updated name
	err = repo.UpsertBatch(ctx, []domain.Ticker{
		{Ticker: "AAPL", Name: "Apple Inc. (Updated)", Market: "stocks", Active: true},
	})
	if err != nil {
		t.Fatalf("second UpsertBatch: %v", err)
	}

	got, err := repo.GetBySymbol(ctx, "AAPL")
	if err != nil {
		t.Fatalf("GetBySymbol: %v", err)
	}
	if got.Name != "Apple Inc. (Updated)" {
		t.Errorf("Name = %q, want updated name", got.Name)
	}
}

func TestUpsertBatch_EmptySlice(t *testing.T) {
	pool := testPool(t)

	repo := repository.NewTickerRepo(pool, discardLogger)
	err := repo.UpsertBatch(context.Background(), nil)
	if err != nil {
		t.Errorf("UpsertBatch(nil) should be no-op, got: %v", err)
	}
}

func TestGetActive_FiltersInactiveTickers(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	repo := repository.NewTickerRepo(pool, discardLogger)
	ctx := context.Background()

	err := repo.UpsertBatch(ctx, []domain.Ticker{
		{Ticker: "AAPL", Name: "Apple", Active: true},
		{Ticker: "DEAD", Name: "Dead Corp", Active: false},
		{Ticker: "MSFT", Name: "Microsoft", Active: true},
	})
	if err != nil {
		t.Fatalf("UpsertBatch: %v", err)
	}

	got, err := repo.GetActive(ctx)
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 active tickers, got %d", len(got))
	}
	for _, tk := range got {
		if tk.Ticker == "DEAD" {
			t.Error("GetActive returned inactive ticker DEAD")
		}
	}
}

func TestGetBySymbol_NotFound(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	repo := repository.NewTickerRepo(pool, discardLogger)
	_, err := repo.GetBySymbol(context.Background(), "NONEXIST")
	if err == nil {
		t.Fatal("expected error for non-existent ticker, got nil")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected pgx.ErrNoRows wrapped, got: %v", err)
	}
}

func TestGetBySymbol_ReturnsFull(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	repo := repository.NewTickerRepo(pool, discardLogger)
	ctx := context.Background()

	err := repo.UpsertBatch(ctx, []domain.Ticker{
		{
			Ticker:          "GOOG",
			Name:            "Alphabet Inc.",
			Market:          "stocks",
			PrimaryExchange: "XNAS",
			Type:            "CS",
			Active:          true,
			CurrencyName:    "usd",
			Locale:          "us",
			CIK:             "0001652044",
			ListDate:        "2004-08-19",
		},
	})
	if err != nil {
		t.Fatalf("UpsertBatch: %v", err)
	}

	got, err := repo.GetBySymbol(ctx, "GOOG")
	if err != nil {
		t.Fatalf("GetBySymbol: %v", err)
	}
	if got.ID == "" {
		t.Error("expected non-empty ID")
	}
	if got.Name != "Alphabet Inc." {
		t.Errorf("Name = %q, want %q", got.Name, "Alphabet Inc.")
	}
	if got.CIK != "0001652044" {
		t.Errorf("CIK = %q, want %q", got.CIK, "0001652044")
	}
	if got.ListDate != "2004-08-19" {
		t.Errorf("ListDate = %q, want %q", got.ListDate, "2004-08-19")
	}
	if got.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}

