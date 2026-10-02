package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/profitify/profitify-backend/internal/config"
	"github.com/profitify/profitify-backend/internal/domain"
	lambdautil "github.com/profitify/profitify-backend/internal/lambda"
	"github.com/profitify/profitify-backend/internal/pipeline"
	"github.com/profitify/profitify-backend/internal/repository"
)

// runCompleter abstracts the pipeline run repository for testing.
type runCompleter interface {
	MarkCompleted(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*domain.PipelineRun, error)
	UpdateStatus(ctx context.Context, id, status, errMsg string) error
}

// stageTracker abstracts pipeline stage tracking for testing.
type stageTracker interface {
	MarkRunning(ctx context.Context, runID, tickerID, stage string) (string, error)
	MarkCompleted(ctx context.Context, runID, tickerID, stage string) error
	MarkFailed(ctx context.Context, runID, tickerID, stage, errorMessage string) error
}

// maxErrorMessageRunes bounds the error message written to pipeline_runs so
// a runaway Cause payload cannot blow out the column.
const maxErrorMessageRunes = 2000

// Event is the input payload for the ClosePipeline Lambda. It is used both
// on the success path (Error is nil) and as the MarkRunFailed task input,
// where Step Functions writes the Catch result at $.error. The state also
// carries stage results (ingestResult, fetchTechnicalsResult, …) that this
// Lambda ignores via encoding/json's default unknown-field behavior.
type Event struct {
	pipeline.TickerEvent
	Error *pipeline.StageError `json:"error,omitempty"`
}

// Response is the output payload for the ClosePipeline Lambda.
type Response struct {
	Ticker         string `json:"ticker"`
	Date           string `json:"date"`
	PipelineStatus string `json:"pipeline_status"`
}

// failureMessage builds the pipeline_runs.error_message for the failure
// path: "<Error>: <Cause>", or "unknown error" when both are empty,
// truncated to maxErrorMessageRunes runes.
func failureMessage(e *pipeline.StageError) string {
	if e.Error == "" && e.Cause == "" {
		return "unknown error"
	}
	msg := fmt.Sprintf("%s: %s", e.Error, e.Cause)
	runes := []rune(msg)
	if len(runes) > maxErrorMessageRunes {
		return string(runes[:maxErrorMessageRunes])
	}
	return msg
}

// closePipeline is the core logic.
func closePipeline(
	ctx context.Context,
	event Event,
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

	if event.Error != nil {
		msg := failureMessage(event.Error)
		if err := runs.UpdateStatus(ctx, event.RunID, domain.PipelineStatusFailed, msg); err != nil {
			return nil, fmt.Errorf("marking pipeline run failed: %w", err)
		}

		logger.Info("pipeline marked failed",
			"ticker", event.Ticker,
			"run_id", event.RunID,
			"error", msg,
		)

		return &Response{
			Ticker:         event.Ticker,
			Date:           event.Date,
			PipelineStatus: domain.PipelineStatusFailed,
		}, nil
	}

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

func handleRequest(ctx context.Context, event Event) (*Response, error) {
	logger := lambdautil.InitLogger()

	cfg, err := config.LoadClosePipeline()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	pool, err := lambdautil.ConnectDB(ctx, cfg.DatabaseURL, cfg.DBSecretARN, cfg.PoolMaxConns)
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
