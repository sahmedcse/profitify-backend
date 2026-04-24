package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/profitify/profitify-backend/internal/db"
	"github.com/profitify/profitify-backend/internal/domain"
	lambdautil "github.com/profitify/profitify-backend/internal/lambda"
	"github.com/profitify/profitify-backend/internal/massive"
	"github.com/profitify/profitify-backend/internal/repository"
)

// tickerFetcher abstracts the Massive client for testing.
type tickerFetcher interface {
	FetchActiveTickers(ctx context.Context) ([]domain.Ticker, error)
}

// runCreator abstracts pipeline run persistence for testing.
type runCreator interface {
	Create(ctx context.Context, run *domain.PipelineRun) (*domain.PipelineRun, error)
	UpdateStatus(ctx context.Context, id string, status string, errorMessage string) error
}

// stageInserter abstracts pipeline stage persistence for testing.
type stageInserter interface {
	BulkInsert(ctx context.Context, stages []domain.PipelineTickerStage) error
}

// Event is the optional input payload for the FetchTickers Lambda.
type Event struct {
	RunParams *domain.PipelineRunParams `json:"run_params"`
}

// Response is the output payload for the FetchTickers Lambda.
type Response struct {
	TickerCount int    `json:"ticker_count"`
	Date        string `json:"date"`
	RunID       string `json:"run_id"`
}

// fetchAndUpsert is the core logic: fetch tickers, upsert to DB, create pipeline run + stages.
func fetchAndUpsert(
	ctx context.Context,
	event Event,
	fetcher tickerFetcher,
	repo repository.TickerRepository,
	runs runCreator,
	stages stageInserter,
	logger *slog.Logger,
) (*Response, error) {
	// Determine date from run_params or default to today.
	date := time.Now().UTC().Format("2006-01-02")
	if event.RunParams != nil && event.RunParams.Date != "" {
		date = event.RunParams.Date
	}

	// 1. Fetch tickers from Massive.
	logger.Info("fetching active tickers from Massive")
	tickers, err := fetcher.FetchActiveTickers(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching tickers: %w", err)
	}

	// 2. Upsert tickers to DB.
	logger.Info("upserting tickers to database", "count", len(tickers))
	if err := repo.UpsertBatch(ctx, tickers); err != nil {
		return nil, fmt.Errorf("upserting tickers: %w", err)
	}

	// 3. Re-read active tickers from DB to get IDs.
	activeTickers, err := repo.GetActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading active tickers: %w", err)
	}

	// 4. Create pipeline run.
	run, err := runs.Create(ctx, &domain.PipelineRun{
		RunParams:   domain.PipelineRunParams{Date: date},
		Status:      domain.PipelineStatusRunning,
		TickerCount: len(activeTickers),
	})
	if err != nil {
		return nil, fmt.Errorf("creating pipeline run: %w", err)
	}

	// 5. Bulk-insert pending stages for all tickers x all stages.
	var allStages []domain.PipelineTickerStage
	for _, t := range activeTickers {
		for _, stage := range domain.AllStages {
			allStages = append(allStages, domain.PipelineTickerStage{
				RunID:    run.ID,
				TickerID: t.ID,
				Ticker:   t.Ticker,
				Stage:    stage,
				Status:   domain.PipelineStatusPending,
			})
		}
	}

	logger.Info("inserting pipeline stages", "total", len(allStages))
	if err := stages.BulkInsert(ctx, allStages); err != nil {
		_ = runs.UpdateStatus(ctx, run.ID, domain.PipelineStatusFailed, err.Error())
		return nil, fmt.Errorf("inserting pipeline stages: %w", err)
	}

	logger.Info("fetch-tickers complete",
		"ticker_count", len(activeTickers), "run_id", run.ID, "date", date)
	return &Response{
		TickerCount: len(activeTickers),
		Date:        date,
		RunID:       run.ID,
	}, nil
}

func handleRequest(ctx context.Context, event Event) (*Response, error) {
	logger := lambdautil.InitLogger()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	apiKey := os.Getenv("MASSIVE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("MASSIVE_API_KEY is required")
	}

	pool, err := db.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	client := massive.NewClient(apiKey, logger)
	tickerRepo := repository.NewTickerRepo(pool, logger)
	runRepo := repository.NewPipelineRunRepo(pool, logger)
	stageRepo := repository.NewPipelineTickerStageRepo(pool, logger)

	return fetchAndUpsert(ctx, event, client, tickerRepo, runRepo, stageRepo, logger)
}

func main() {
	lambda.Start(handleRequest)
}
