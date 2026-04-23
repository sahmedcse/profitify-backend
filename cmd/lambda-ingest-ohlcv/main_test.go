package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/pipeline"
	"github.com/profitify/profitify-backend/internal/repository"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// stubOHLCVFetcher returns a fixed DailyPrice.
type stubOHLCVFetcher struct {
	price *domain.DailyPrice
	err   error
}

func (f *stubOHLCVFetcher) FetchDailyOHLCV(_ context.Context, _ string, _ time.Time) (*domain.DailyPrice, error) {
	return f.price, f.err
}

// stubPriceWriter records the upserted price.
type stubPriceWriter struct {
	upserted *domain.DailyPrice
	err      error
}

func (w *stubPriceWriter) Upsert(_ context.Context, price *domain.DailyPrice) error {
	w.upserted = price
	return w.err
}

// stubStageTracker is a no-op tracker for tests.
type stubStageTracker struct{}

func (s *stubStageTracker) MarkRunning(_ context.Context, _, _, _ string) (string, error) {
	return "stage-id", nil
}

func (s *stubStageTracker) MarkCompleted(_ context.Context, _, _, _ string) error {
	return nil
}

func (s *stubStageTracker) MarkFailed(_ context.Context, _, _, _, _ string) error {
	return nil
}

// failingStageTracker always returns errors (verifies tracking failures don't abort work).
type failingStageTracker struct{}

func (s *failingStageTracker) MarkRunning(_ context.Context, _, _, _ string) (string, error) {
	return "", fmt.Errorf("tracking unavailable")
}

func (s *failingStageTracker) MarkCompleted(_ context.Context, _, _, _ string) error {
	return fmt.Errorf("tracking unavailable")
}

func (s *failingStageTracker) MarkFailed(_ context.Context, _, _, _, _ string) error {
	return fmt.Errorf("tracking unavailable")
}

func TestIngestOHLCV_HappyPath(t *testing.T) {
	fetcher := &stubOHLCVFetcher{
		price: &domain.DailyPrice{
			Open: 175.50, High: 178.25, Low: 174.80, Close: 177.10,
			Volume: 52340000, PreMarket: 175.20, AfterHours: 177.30,
		},
	}
	writer := &stubPriceWriter{}

	event := pipeline.TickerEvent{
		Ticker:   "AAPL",
		TickerID: "uuid-123",
		Date:     "2026-04-08",
	}

	resp, err := ingestOHLCV(context.Background(), event, fetcher, writer, &stubStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Ticker != "AAPL" {
		t.Errorf("Ticker = %q, want AAPL", resp.Ticker)
	}
	if resp.Status != "ok" {
		t.Errorf("Status = %q, want ok", resp.Status)
	}
	if writer.upserted == nil {
		t.Fatal("expected price to be upserted")
	}
	if writer.upserted.TickerID != "uuid-123" {
		t.Errorf("TickerID = %q, want uuid-123", writer.upserted.TickerID)
	}
}

func TestIngestOHLCV_MissingTickerID(t *testing.T) {
	event := pipeline.TickerEvent{Ticker: "AAPL", Date: "2026-04-08"}
	_, err := ingestOHLCV(context.Background(), event, nil, nil, &stubStageTracker{}, discardLogger)
	if err == nil {
		t.Fatal("expected error for missing ticker_id")
	}
}

func TestIngestOHLCV_MissingTicker(t *testing.T) {
	event := pipeline.TickerEvent{TickerID: "uuid-123", Date: "2026-04-08"}
	_, err := ingestOHLCV(context.Background(), event, nil, nil, &stubStageTracker{}, discardLogger)
	if err == nil {
		t.Fatal("expected error for missing ticker")
	}
}

func TestIngestOHLCV_InvalidDate(t *testing.T) {
	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "not-a-date"}
	_, err := ingestOHLCV(context.Background(), event, nil, nil, &stubStageTracker{}, discardLogger)
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestIngestOHLCV_FetchError(t *testing.T) {
	fetcher := &stubOHLCVFetcher{err: fmt.Errorf("api timeout")}
	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}

	_, err := ingestOHLCV(context.Background(), event, fetcher, nil, &stubStageTracker{}, discardLogger)
	if err == nil {
		t.Fatal("expected error for fetch failure")
	}
}

func TestIngestOHLCV_UpsertError(t *testing.T) {
	fetcher := &stubOHLCVFetcher{
		price: &domain.DailyPrice{Close: 100},
	}
	writer := &stubPriceWriter{err: fmt.Errorf("db connection error")}
	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}

	_, err := ingestOHLCV(context.Background(), event, fetcher, writer, &stubStageTracker{}, discardLogger)
	if err == nil {
		t.Fatal("expected error for upsert failure")
	}
}

func TestIngestOHLCV_TrackingFailure_DoesNotAbort(t *testing.T) {
	fetcher := &stubOHLCVFetcher{
		price: &domain.DailyPrice{Close: 100},
	}
	writer := &stubPriceWriter{}
	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08", RunID: "run-123"}

	resp, err := ingestOHLCV(context.Background(), event, fetcher, writer, &failingStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("tracking failure should not abort work: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("Status = %q, want ok", resp.Status)
	}
}

func TestHandleRequest_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("MASSIVE_API_KEY", "test-key")

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	_, err := handleRequest(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
}

func TestHandleRequest_MissingAPIKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("MASSIVE_API_KEY", "")

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	_, err := handleRequest(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for missing MASSIVE_API_KEY")
	}
}

// Integration test: stub fetcher + real DB
func TestIngestOHLCV_Integration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Clean and insert a test ticker
	_, _ = pool.Exec(ctx, "DELETE FROM daily_prices WHERE ticker_id IN (SELECT id FROM tickers WHERE ticker = 'INGEST_TEST')")
	_, _ = pool.Exec(ctx, "DELETE FROM tickers WHERE ticker = 'INGEST_TEST'")
	var tickerID string
	err = pool.QueryRow(ctx,
		"INSERT INTO tickers (ticker, name, market, type, active) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		"INGEST_TEST", "Ingest Test Corp", "stocks", "CS", true).Scan(&tickerID)
	if err != nil {
		t.Fatalf("failed to insert test ticker: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM daily_prices WHERE ticker_id = $1", tickerID)
		_, _ = pool.Exec(ctx, "DELETE FROM tickers WHERE id = $1", tickerID)
	})

	fetcher := &stubOHLCVFetcher{
		price: &domain.DailyPrice{
			Open: 100, High: 105, Low: 99, Close: 103,
			Volume: 1000000, VWAP: 102, PreMarket: 99.5, AfterHours: 103.5,
		},
	}
	repo := repository.NewDailyPriceRepo(pool, discardLogger)
	event := pipeline.TickerEvent{Ticker: "INGEST_TEST", TickerID: tickerID, Date: "2026-04-08"}

	resp, err := ingestOHLCV(ctx, event, fetcher, repo, &stubStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("ingestOHLCV: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("Status = %q, want ok", resp.Status)
	}

	// Verify persisted
	latest, err := repo.GetLatest(ctx, tickerID)
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if latest.Close != 103 {
		t.Errorf("Close = %v, want 103", latest.Close)
	}
}
