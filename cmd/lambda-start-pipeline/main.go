package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sfn"

	"github.com/profitify/profitify-backend/internal/config"
	"github.com/profitify/profitify-backend/internal/db"
	"github.com/profitify/profitify-backend/internal/domain"
	lambdautil "github.com/profitify/profitify-backend/internal/lambda"
	"github.com/profitify/profitify-backend/internal/pipeline"
	"github.com/profitify/profitify-backend/internal/queue"
	"github.com/profitify/profitify-backend/internal/repository"
)

// runCreator abstracts the pipeline run repository for testing.
type runCreator interface {
	Create(ctx context.Context, run *domain.PipelineRun) (*domain.PipelineRun, error)
	UpdateStatus(ctx context.Context, id, status, errMsg string) error
	UpdateSFNArn(ctx context.Context, id, arn string) error
}

// stageTracker abstracts the stage repository for testing.
// Matches pipeline.StageUpdater so the real repo satisfies it.
type stageTracker interface {
	MarkRunning(ctx context.Context, runID, tickerID, stage string) (string, error)
	MarkCompleted(ctx context.Context, runID, tickerID, stage string) error
	MarkFailed(ctx context.Context, runID, tickerID, stage, errorMessage string) error
}

// sfnStarter abstracts the Step Functions client for testing.
type sfnStarter interface {
	StartExecution(ctx context.Context, input *sfn.StartExecutionInput, opts ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error)
}

// startPipeline processes an SQS event containing ticker messages.
// For each message it creates a pipeline run, tracks its own stage,
// and starts a Step Function execution.
func startPipeline(
	ctx context.Context,
	event events.SQSEvent,
	runs runCreator,
	stages stageTracker,
	sfnClient sfnStarter,
	sfnArn string,
	logger *slog.Logger,
) (*events.SQSEventResponse, error) {
	var failures []events.SQSBatchItemFailure

	for _, record := range event.Records {
		var msg queue.TickerMessage
		if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
			logger.Error("failed to parse SQS message", "messageId", record.MessageId, "error", err)
			failures = append(failures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
			continue
		}

		if err := processTicker(ctx, msg, runs, stages, sfnClient, sfnArn, logger); err != nil {
			logger.Error("failed to process ticker", "ticker", msg.Ticker.Ticker, "messageId", record.MessageId, "error", err)
			failures = append(failures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
		}
	}

	return &events.SQSEventResponse{BatchItemFailures: failures}, nil
}

func processTicker(
	ctx context.Context,
	msg queue.TickerMessage,
	runs runCreator,
	stages stageTracker,
	sfnClient sfnStarter,
	sfnArn string,
	logger *slog.Logger,
) error {
	// 1. Create pipeline run.
	run, err := runs.Create(ctx, &domain.PipelineRun{
		TickerID: msg.Ticker.ID,
		Ticker:   msg.Ticker.Ticker,
		Date:     msg.Date,
		Status:   domain.PipelineStatusPending,
	})
	if err != nil {
		return fmt.Errorf("creating pipeline run: %w", err)
	}

	logger.Info("pipeline run created", "runId", run.ID, "ticker", msg.Ticker.Ticker)

	// 2. Track start_pipeline stage.
	st := pipeline.NewStageTracker(stages, run.ID, msg.Ticker.ID, domain.StageStartPipeline, logger)
	if err := st.Begin(ctx); err != nil {
		_ = runs.UpdateStatus(ctx, run.ID, domain.PipelineStatusFailed, err.Error())
		return fmt.Errorf("tracking start_pipeline stage: %w", err)
	}

	// 3. Start Step Function execution.
	tickerEvent := pipeline.TickerEvent{
		Ticker:   msg.Ticker.Ticker,
		TickerID: msg.Ticker.ID,
		Date:     msg.Date,
		RunID:    run.ID,
	}

	sfnInput, err := json.Marshal(tickerEvent)
	if err != nil {
		st.End(ctx, err)
		_ = runs.UpdateStatus(ctx, run.ID, domain.PipelineStatusFailed, err.Error())
		return fmt.Errorf("marshaling SFN input: %w", err)
	}

	inputStr := string(sfnInput)
	out, err := sfnClient.StartExecution(ctx, &sfn.StartExecutionInput{
		StateMachineArn: &sfnArn,
		Input:           &inputStr,
	})
	if err != nil {
		st.End(ctx, err)
		_ = runs.UpdateStatus(ctx, run.ID, domain.PipelineStatusFailed, err.Error())
		return fmt.Errorf("starting SFN execution: %w", err)
	}

	// 4. Save execution ARN (non-fatal if this fails).
	if err := runs.UpdateSFNArn(ctx, run.ID, *out.ExecutionArn); err != nil {
		logger.Warn("failed to save SFN ARN", "runId", run.ID, "error", err)
	}

	// 5. Mark start_pipeline stage completed.
	st.End(ctx, nil)

	logger.Info("SFN execution started", "runId", run.ID, "ticker", msg.Ticker.Ticker, "executionArn", *out.ExecutionArn)
	return nil
}

func handleRequest(ctx context.Context, event events.SQSEvent) (*events.SQSEventResponse, error) {
	logger := lambdautil.InitLogger()

	cfg, err := config.LoadStartPipeline()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	awsCfg, err := awscfg.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	runRepo := repository.NewPipelineRunRepo(pool, logger)
	stageRepo := repository.NewPipelineTickerStageRepo(pool, logger)
	sfnClient := sfn.NewFromConfig(awsCfg)

	return startPipeline(ctx, event, runRepo, stageRepo, sfnClient, cfg.SFNArn, logger)
}

func main() {
	lambda.Start(handleRequest)
}
