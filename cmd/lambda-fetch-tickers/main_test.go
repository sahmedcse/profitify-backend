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

// stubRunCreator stubs pipeline run creation.
type stubRunCreator struct {
	created       *domain.PipelineRun
	createErr     error
	updatedStatus string
}

func (r *stubRunCreator) Create(_ context.Context, run *domain.PipelineRun) (*domain.PipelineRun, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	run.ID = "test-run-id"
	r.created = run
	return run, nil
}

func (r *stubRunCreator) UpdateStatus(_ context.Context, _ string, status string, _ string) error {
	r.updatedStatus = status
	return nil
}

// stubStageInserter stubs pipeline stage insertion.
type stubStageInserter struct {
	inserted []domain.PipelineTickerStage
	err      error
}

func (s *stubStageInserter) BulkInsert(_ context.Context, stages []domain.PipelineTickerStage) error {
	s.inserted = stages
	return s.err
}

// stubTickerRepo implements repository.TickerRepository for unit tests.
type stubTickerRepo struct {
	upserted []domain.Ticker
	active   []domain.Ticker
}

func (r *stubTickerRepo) UpsertBatch(_ context.Context, tickers []domain.Ticker) error {
	r.upserted = tickers
	return nil
}

func (r *stubTickerRepo) GetActive(_ context.Context) ([]domain.Ticker, error) {
	return r.active, nil
}

func (r *stubTickerRepo) GetBySymbol(_ context.Context, _ string) (*domain.Ticker, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *stubTickerRepo) UpdateSector(_ context.Context, _ string, _ string) error {
	return nil
}

func TestFetchAndUpsert_Unit_Success(t *testing.T) {
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL", Name: "Apple"},
			{Ticker: "MSFT", Name: "Microsoft"},
		},
	}
	repo := &stubTickerRepo{
		active: []domain.Ticker{
			{ID: "id-1", Ticker: "AAPL"},
			{ID: "id-2", Ticker: "MSFT"},
		},
	}
	runs := &stubRunCreator{}
	stages := &stubStageInserter{}

	resp, err := fetchAndUpsert(context.Background(), Event{}, fetcher, repo, runs, stages, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndUpsert: %v", err)
	}

	if resp.TickerCount != 2 {
		t.Errorf("TickerCount = %d, want 2", resp.TickerCount)
	}
	if resp.RunID != "test-run-id" {
		t.Errorf("RunID = %q, want %q", resp.RunID, "test-run-id")
	}
	if resp.Date == "" {
		t.Error("Date should not be empty")
	}
}

func TestFetchAndUpsert_Unit_CustomDate(t *testing.T) {
	fetcher := &stubFetcher{tickers: []domain.Ticker{{Ticker: "AAPL"}}}
	repo := &stubTickerRepo{active: []domain.Ticker{{ID: "id-1", Ticker: "AAPL"}}}
	runs := &stubRunCreator{}
	stages := &stubStageInserter{}

	event := Event{RunParams: &domain.PipelineRunParams{Date: "2026-01-15"}}
	resp, err := fetchAndUpsert(context.Background(), event, fetcher, repo, runs, stages, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndUpsert: %v", err)
	}

	if resp.Date != "2026-01-15" {
		t.Errorf("Date = %q, want %q", resp.Date, "2026-01-15")
	}
	if runs.created.RunParams.Date != "2026-01-15" {
		t.Errorf("RunParams.Date = %q, want %q", runs.created.RunParams.Date, "2026-01-15")
	}
}

func TestFetchAndUpsert_Unit_StagesCount(t *testing.T) {
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL"},
			{Ticker: "MSFT"},
			{Ticker: "GOOG"},
		},
	}
	repo := &stubTickerRepo{
		active: []domain.Ticker{
			{ID: "id-1", Ticker: "AAPL"},
			{ID: "id-2", Ticker: "MSFT"},
			{ID: "id-3", Ticker: "GOOG"},
		},
	}
	runs := &stubRunCreator{}
	stages := &stubStageInserter{}

	_, err := fetchAndUpsert(context.Background(), Event{}, fetcher, repo, runs, stages, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndUpsert: %v", err)
	}

	// 3 tickers x 5 stages = 15 stage rows
	if len(stages.inserted) != 15 {
		t.Errorf("stage count = %d, want 15", len(stages.inserted))
	}
}

func TestFetchAndUpsert_Unit_FetchError(t *testing.T) {
	fetcher := &stubFetcher{err: fmt.Errorf("api timeout")}
	repo := &stubTickerRepo{}
	runs := &stubRunCreator{}
	stages := &stubStageInserter{}

	_, err := fetchAndUpsert(context.Background(), Event{}, fetcher, repo, runs, stages, discardLogger)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFetchAndUpsert_Unit_CreateRunError(t *testing.T) {
	fetcher := &stubFetcher{tickers: []domain.Ticker{{Ticker: "AAPL"}}}
	repo := &stubTickerRepo{active: []domain.Ticker{{ID: "id-1", Ticker: "AAPL"}}}
	runs := &stubRunCreator{createErr: fmt.Errorf("db connection failed")}
	stages := &stubStageInserter{}

	_, err := fetchAndUpsert(context.Background(), Event{}, fetcher, repo, runs, stages, discardLogger)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFetchAndUpsert_Unit_StageInsertError(t *testing.T) {
	fetcher := &stubFetcher{tickers: []domain.Ticker{{Ticker: "AAPL"}}}
	repo := &stubTickerRepo{active: []domain.Ticker{{ID: "id-1", Ticker: "AAPL"}}}
	runs := &stubRunCreator{}
	stages := &stubStageInserter{err: fmt.Errorf("batch insert failed")}

	_, err := fetchAndUpsert(context.Background(), Event{}, fetcher, repo, runs, stages, discardLogger)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Run should be marked as failed.
	if runs.updatedStatus != domain.PipelineStatusFailed {
		t.Errorf("run status = %q, want %q", runs.updatedStatus, domain.PipelineStatusFailed)
	}
}

func TestFetchAndUpsert_Integration(t *testing.T) {
	pool := testPool(t)
	cleanTickers(t, pool)

	tickerRepo := repository.NewTickerRepo(pool, discardLogger)
	runRepo := repository.NewPipelineRunRepo(pool, discardLogger)
	stageRepo := repository.NewPipelineTickerStageRepo(pool, discardLogger)
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL", Name: "Apple Inc.", Market: "stocks", Active: true, Type: "CS"},
			{Ticker: "MSFT", Name: "Microsoft Corporation", Market: "stocks", Active: true, Type: "CS"},
			{Ticker: "GOOG", Name: "Alphabet Inc.", Market: "stocks", Active: true, Type: "CS"},
		},
	}

	resp, err := fetchAndUpsert(context.Background(), Event{}, fetcher, tickerRepo, runRepo, stageRepo, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndUpsert: %v", err)
	}

	if resp.TickerCount != 3 {
		t.Errorf("TickerCount = %d, want 3", resp.TickerCount)
	}
	if resp.RunID == "" {
		t.Error("RunID should not be empty")
	}
	if resp.Date == "" {
		t.Error("Date should not be empty")
	}

	// Verify pipeline run was created.
	run, err := runRepo.GetByID(context.Background(), resp.RunID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if run.TickerCount != 3 {
		t.Errorf("run.TickerCount = %d, want 3", run.TickerCount)
	}
	if run.Status != domain.PipelineStatusRunning {
		t.Errorf("run.Status = %q, want %q", run.Status, domain.PipelineStatusRunning)
	}

	// Clean up.
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM pipeline_ticker_stages")
		_, _ = pool.Exec(context.Background(), "DELETE FROM pipeline_runs")
		_, _ = pool.Exec(context.Background(), "DELETE FROM tickers")
	})
}

func TestHandleRequest_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("MASSIVE_API_KEY", "test-key")

	_, err := handleRequest(context.Background(), Event{})
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
}

func TestHandleRequest_MissingAPIKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("MASSIVE_API_KEY", "")

	_, err := handleRequest(context.Background(), Event{})
	if err == nil {
		t.Fatal("expected error for missing MASSIVE_API_KEY")
	}
}
