package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/profitify/profitify-backend/internal/config"
	"github.com/profitify/profitify-backend/internal/db"
	"github.com/profitify/profitify-backend/internal/domain"
	lambdautil "github.com/profitify/profitify-backend/internal/lambda"
	"github.com/profitify/profitify-backend/internal/pipeline"
	"github.com/profitify/profitify-backend/internal/repository"
)

// runCompleter abstracts the pipeline run repository for testing.
type runCompleter interface {
	MarkCompleted(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*domain.PipelineRun, error)
}

// stageTracker abstracts pipeline stage tracking for testing.
type stageTracker interface {
	MarkRunning(ctx context.Context, runID, tickerID, stage string) (string, error)
	MarkCompleted(ctx context.Context, runID, tickerID, stage string) error
	MarkFailed(ctx context.Context, runID, tickerID, stage, errorMessage string) error
}

// Response is the output payload for the ClosePipeline Lambda.
type Response struct {
	Ticker         string `json:"ticker"`
	Date           string `json:"date"`
	PipelineStatus string `json:"pipeline_status"`
}

// closePipeline is the core logic.
func closePipeline(
	ctx context.Context,
	event pipeline.TickerEvent,
	runs runCompleter,
	tracker stageTracker,
	logger *slog.Logger,
) (_ *Response, retErr error) {
	if event.RunID == "" {
		logger.Info("no run_id, skipping pipeline close", "ticker", event.Ticker)
		return &Response{
			Ticker:         event.Ticker,
			Date:           event.Date,
			PipelineStatus: "skipped",
		}, nil
	}

	st := pipeline.NewStageTracker(tracker, event.RunID, event.TickerID, domain.StageClosePipeline, logger)
	_ = st.Begin(ctx)
	defer func() { st.End(ctx, retErr) }()

	if err := runs.MarkCompleted(ctx, event.RunID); err != nil {
		return nil, fmt.Errorf("marking pipeline run completed: %w", err)
	}

	run, err := runs.GetByID(ctx, event.RunID)
	if err != nil {
		return nil, fmt.Errorf("reading final pipeline status: %w", err)
	}

	logger.Info("pipeline closed",
		"ticker", event.Ticker,
		"run_id", event.RunID,
		"status", run.Status,
	)

	return &Response{
		Ticker:         event.Ticker,
		Date:           event.Date,
		PipelineStatus: run.Status,
	}, nil
}

func handleRequest(ctx context.Context, event pipeline.TickerEvent) (*Response, error) {
	logger := lambdautil.InitLogger()

	cfg, err := config.LoadClosePipeline()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	runRepo := repository.NewPipelineRunRepo(pool, logger)
	stageRepo := repository.NewPipelineTickerStageRepo(pool, logger)

	return closePipeline(ctx, event, runRepo, stageRepo, logger)
}

func main() {
	lambda.Start(handleRequest)
}
