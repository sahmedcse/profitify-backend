package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/pipeline"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// stubs

type stubRunRepo struct {
	markCompletedErr error
	getByIDRun       *domain.PipelineRun
	getByIDErr       error
}

func (r *stubRunRepo) MarkCompleted(_ context.Context, _ string) error {
	return r.markCompletedErr
}

func (r *stubRunRepo) GetByID(_ context.Context, _ string) (*domain.PipelineRun, error) {
	return r.getByIDRun, r.getByIDErr
}

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

func TestClosePipeline_HappyPath(t *testing.T) {
	repo := &stubRunRepo{
		getByIDRun: &domain.PipelineRun{
			ID:     "run-123",
			Status: domain.PipelineStatusCompleted,
		},
	}

	event := pipeline.TickerEvent{
		Ticker:   "AAPL",
		TickerID: "uuid-123",
		Date:     "2026-04-08",
		RunID:    "run-123",
	}

	resp, err := closePipeline(context.Background(), event, repo, &stubStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ticker != "AAPL" {
		t.Errorf("Ticker = %q, want AAPL", resp.Ticker)
	}
	if resp.Date != "2026-04-08" {
		t.Errorf("Date = %q, want 2026-04-08", resp.Date)
	}
	if resp.PipelineStatus != domain.PipelineStatusCompleted {
		t.Errorf("PipelineStatus = %q, want %q", resp.PipelineStatus, domain.PipelineStatusCompleted)
	}
}

func TestClosePipeline_EmptyRunID(t *testing.T) {
	event := pipeline.TickerEvent{
		Ticker:   "AAPL",
		TickerID: "uuid-123",
		Date:     "2026-04-08",
		RunID:    "",
	}

	resp, err := closePipeline(context.Background(), event, nil, &stubStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PipelineStatus != "skipped" {
		t.Errorf("PipelineStatus = %q, want skipped", resp.PipelineStatus)
	}
}

func TestClosePipeline_MarkCompletedFails(t *testing.T) {
	repo := &stubRunRepo{
		markCompletedErr: fmt.Errorf("db connection lost"),
	}

	event := pipeline.TickerEvent{
		Ticker:   "AAPL",
		TickerID: "uuid-123",
		Date:     "2026-04-08",
		RunID:    "run-123",
	}

	_, err := closePipeline(context.Background(), event, repo, &stubStageTracker{}, discardLogger)
	if err == nil {
		t.Fatal("expected error for MarkCompleted failure")
	}
}

func TestClosePipeline_GetByIDFails(t *testing.T) {
	repo := &stubRunRepo{
		getByIDErr: fmt.Errorf("not found"),
	}

	event := pipeline.TickerEvent{
		Ticker:   "AAPL",
		TickerID: "uuid-123",
		Date:     "2026-04-08",
		RunID:    "run-123",
	}

	_, err := closePipeline(context.Background(), event, repo, &stubStageTracker{}, discardLogger)
	if err == nil {
		t.Fatal("expected error for GetByID failure")
	}
}

func TestClosePipeline_FailedPipelineStatus(t *testing.T) {
	repo := &stubRunRepo{
		getByIDRun: &domain.PipelineRun{
			ID:     "run-123",
			Status: domain.PipelineStatusFailed,
		},
	}

	event := pipeline.TickerEvent{
		Ticker:   "AAPL",
		TickerID: "uuid-123",
		Date:     "2026-04-08",
		RunID:    "run-123",
	}

	resp, err := closePipeline(context.Background(), event, repo, &stubStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PipelineStatus != domain.PipelineStatusFailed {
		t.Errorf("PipelineStatus = %q, want %q", resp.PipelineStatus, domain.PipelineStatusFailed)
	}
}

func TestClosePipeline_TrackingFailure_DoesNotAbort(t *testing.T) {
	repo := &stubRunRepo{
		getByIDRun: &domain.PipelineRun{
			ID:     "run-123",
			Status: domain.PipelineStatusCompleted,
		},
	}

	event := pipeline.TickerEvent{
		Ticker:   "AAPL",
		TickerID: "uuid-123",
		Date:     "2026-04-08",
		RunID:    "run-123",
	}

	resp, err := closePipeline(context.Background(), event, repo, &failingStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("tracking failure should not abort work: %v", err)
	}
	if resp.PipelineStatus != domain.PipelineStatusCompleted {
		t.Errorf("PipelineStatus = %q, want %q", resp.PipelineStatus, domain.PipelineStatusCompleted)
	}
}

func TestHandleRequest_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08", RunID: "run-123"}
	_, err := handleRequest(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
}
