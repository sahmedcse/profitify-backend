package pipeline

import (
	"context"
	"fmt"
	"log/slog"
)

// StageUpdater is the minimal interface needed by StageTracker.
// Defined here (not in repository) to avoid import cycles.
// repository.PipelineTickerStageRepository satisfies this implicitly.
type StageUpdater interface {
	MarkRunning(ctx context.Context, runID, tickerID, stage string) (string, error)
	MarkCompleted(ctx context.Context, runID, tickerID, stage string) error
	MarkFailed(ctx context.Context, runID, tickerID, stage, errorMessage string) error
}

// StageTracker provides a lightweight interface for Lambdas to report
// their progress. If runID is empty, all operations are no-ops.
type StageTracker struct {
	updater  StageUpdater
	runID    string
	tickerID string
	stage    string
	logger   *slog.Logger
	active   bool
}

// NewStageTracker creates a tracker. If runID is empty, the tracker
// becomes a no-op (safe for local testing without a pipeline run).
func NewStageTracker(updater StageUpdater, runID, tickerID, stage string, logger *slog.Logger) *StageTracker {
	return &StageTracker{
		updater:  updater,
		runID:    runID,
		tickerID: tickerID,
		stage:    stage,
		logger:   logger,
	}
}

// Begin marks the stage as running. Call this before doing work.
func (t *StageTracker) Begin(ctx context.Context) error {
	if t.runID == "" {
		return nil
	}
	_, err := t.updater.MarkRunning(ctx, t.runID, t.tickerID, t.stage)
	if err != nil {
		t.logger.Warn("pipeline tracking: failed to mark running",
			"run_id", t.runID, "ticker_id", t.tickerID, "stage", t.stage, "error", err)
		return fmt.Errorf("stage tracker begin: %w", err)
	}
	t.active = true
	return nil
}

// End marks the stage as completed (if workErr is nil) or failed
// (if workErr is non-nil). Call this after doing work.
func (t *StageTracker) End(ctx context.Context, workErr error) {
	if !t.active {
		return
	}
	if workErr != nil {
		if err := t.updater.MarkFailed(ctx, t.runID, t.tickerID, t.stage, workErr.Error()); err != nil {
			t.logger.Warn("pipeline tracking: failed to mark failed",
				"run_id", t.runID, "ticker_id", t.tickerID, "stage", t.stage, "error", err)
		}
		return
	}
	if err := t.updater.MarkCompleted(ctx, t.runID, t.tickerID, t.stage); err != nil {
		t.logger.Warn("pipeline tracking: failed to mark completed",
			"run_id", t.runID, "ticker_id", t.tickerID, "stage", t.stage, "error", err)
	}
}
