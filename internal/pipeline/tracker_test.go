package pipeline_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/profitify/profitify-backend/internal/pipeline"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

type mockUpdater struct {
	runningCalls   []string
	completedCalls []string
	failedCalls    []struct{ stage, msg string }
	markRunningErr   error
	markCompletedErr error
	markFailedErr    error
}

func (m *mockUpdater) MarkRunning(_ context.Context, _, _, stage string) (string, error) {
	m.runningCalls = append(m.runningCalls, stage)
	return "stage-id", m.markRunningErr
}

func (m *mockUpdater) MarkCompleted(_ context.Context, _, _, stage string) error {
	m.completedCalls = append(m.completedCalls, stage)
	return m.markCompletedErr
}

func (m *mockUpdater) MarkFailed(_ context.Context, _, _, stage, msg string) error {
	m.failedCalls = append(m.failedCalls, struct{ stage, msg string }{stage, msg})
	return m.markFailedErr
}

func TestStageTracker_HappyPath(t *testing.T) {
	m := &mockUpdater{}
	tracker := pipeline.NewStageTracker(m, "run-1", "ticker-1", "ingest_ohlcv", discardLogger)

	ctx := context.Background()
	if err := tracker.Begin(ctx); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	tracker.End(ctx, nil)

	if len(m.runningCalls) != 1 {
		t.Errorf("running calls = %d, want 1", len(m.runningCalls))
	}
	if len(m.completedCalls) != 1 {
		t.Errorf("completed calls = %d, want 1", len(m.completedCalls))
	}
	if len(m.failedCalls) != 0 {
		t.Errorf("failed calls = %d, want 0", len(m.failedCalls))
	}
}

func TestStageTracker_WorkFailed(t *testing.T) {
	m := &mockUpdater{}
	tracker := pipeline.NewStageTracker(m, "run-1", "ticker-1", "ingest_ohlcv", discardLogger)

	ctx := context.Background()
	_ = tracker.Begin(ctx)
	tracker.End(ctx, fmt.Errorf("api timeout"))

	if len(m.failedCalls) != 1 {
		t.Fatalf("failed calls = %d, want 1", len(m.failedCalls))
	}
	if m.failedCalls[0].msg != "api timeout" {
		t.Errorf("msg = %q, want %q", m.failedCalls[0].msg, "api timeout")
	}
	if len(m.completedCalls) != 0 {
		t.Errorf("completed calls = %d, want 0", len(m.completedCalls))
	}
}

func TestStageTracker_EmptyRunID_Noop(t *testing.T) {
	m := &mockUpdater{}
	tracker := pipeline.NewStageTracker(m, "", "ticker-1", "ingest_ohlcv", discardLogger)

	ctx := context.Background()
	if err := tracker.Begin(ctx); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	tracker.End(ctx, nil)

	if len(m.runningCalls) != 0 {
		t.Errorf("expected 0 calls with empty run_id, got %d", len(m.runningCalls))
	}
}

func TestStageTracker_BeginError(t *testing.T) {
	m := &mockUpdater{markRunningErr: fmt.Errorf("db error")}
	tracker := pipeline.NewStageTracker(m, "run-1", "ticker-1", "ingest_ohlcv", discardLogger)

	ctx := context.Background()
	if err := tracker.Begin(ctx); err == nil {
		t.Fatal("expected error from Begin")
	}
	// End should be no-op since Begin failed (active=false)
	tracker.End(ctx, nil)
	if len(m.completedCalls) != 0 {
		t.Errorf("expected 0 completed calls after failed Begin, got %d", len(m.completedCalls))
	}
}

func TestStageTracker_EndWithoutBegin(t *testing.T) {
	m := &mockUpdater{}
	tracker := pipeline.NewStageTracker(m, "run-1", "ticker-1", "ingest_ohlcv", discardLogger)

	tracker.End(context.Background(), nil)

	if len(m.completedCalls) != 0 {
		t.Errorf("expected 0 calls without Begin, got %d", len(m.completedCalls))
	}
}
