package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/pipeline"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// stubs

type updateStatusCall struct {
	id, status, errMsg string
}

type stubRunRepo struct {
	markCompletedErr   error
	markCompletedCalls int
	getByIDRun         *domain.PipelineRun
	getByIDErr         error
	updateStatusErr    error
	updateStatusCalls  []updateStatusCall
}

func (r *stubRunRepo) MarkCompleted(_ context.Context, _ string) error {
	r.markCompletedCalls++
	return r.markCompletedErr
}

func (r *stubRunRepo) GetByID(_ context.Context, _ string) (*domain.PipelineRun, error) {
	return r.getByIDRun, r.getByIDErr
}

func (r *stubRunRepo) UpdateStatus(_ context.Context, id, status, errMsg string) error {
	r.updateStatusCalls = append(r.updateStatusCalls, updateStatusCall{id, status, errMsg})
	return r.updateStatusErr
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

	event := Event{TickerEvent: pipeline.TickerEvent{
		Ticker:   "AAPL",
		TickerID: "uuid-123",
		Date:     "2026-04-08",
		RunID:    "run-123",
	}}

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
	event := Event{TickerEvent: pipeline.TickerEvent{
		Ticker:   "AAPL",
		TickerID: "uuid-123",
		Date:     "2026-04-08",
		RunID:    "",
	}}

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

	event := Event{TickerEvent: pipeline.TickerEvent{
		Ticker:   "AAPL",
		TickerID: "uuid-123",
		Date:     "2026-04-08",
		RunID:    "run-123",
	}}

	_, err := closePipeline(context.Background(), event, repo, &stubStageTracker{}, discardLogger)
	if err == nil {
		t.Fatal("expected error for MarkCompleted failure")
	}
}

func TestClosePipeline_GetByIDFails(t *testing.T) {
	repo := &stubRunRepo{
		getByIDErr: fmt.Errorf("not found"),
	}

	event := Event{TickerEvent: pipeline.TickerEvent{
		Ticker:   "AAPL",
		TickerID: "uuid-123",
		Date:     "2026-04-08",
		RunID:    "run-123",
	}}

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

	event := Event{TickerEvent: pipeline.TickerEvent{
		Ticker:   "AAPL",
		TickerID: "uuid-123",
		Date:     "2026-04-08",
		RunID:    "run-123",
	}}

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

	event := Event{TickerEvent: pipeline.TickerEvent{
		Ticker:   "AAPL",
		TickerID: "uuid-123",
		Date:     "2026-04-08",
		RunID:    "run-123",
	}}

	resp, err := closePipeline(context.Background(), event, repo, &failingStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("tracking failure should not abort work: %v", err)
	}
	if resp.PipelineStatus != domain.PipelineStatusCompleted {
		t.Errorf("PipelineStatus = %q, want %q", resp.PipelineStatus, domain.PipelineStatusCompleted)
	}
}

func TestClosePipeline_DecodesSFNCatchPayload(t *testing.T) {
	const raw = `{"ticker":"AAPL","ticker_id":"uuid-1","date":"2026-09-29","run_id":"run-1","ingestResult":{"Payload":{}},"error":{"Error":"States.TaskFailed","Cause":"{\"errorMessage\":\"boom\"}"}}`

	var event Event
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if event.RunID != "run-1" {
		t.Errorf("RunID = %q, want %q", event.RunID, "run-1")
	}
	if event.Error == nil {
		t.Fatal("Error = nil, want non-nil")
	}
	if event.Error.Error != "States.TaskFailed" {
		t.Errorf("Error.Error = %q, want %q", event.Error.Error, "States.TaskFailed")
	}
	if event.Error.Cause != `{"errorMessage":"boom"}` {
		t.Errorf("Error.Cause = %q, want %q", event.Error.Cause, `{"errorMessage":"boom"}`)
	}
}

func TestClosePipeline_FailurePath_MarksRunFailed(t *testing.T) {
	repo := &stubRunRepo{}

	event := Event{
		TickerEvent: pipeline.TickerEvent{
			Ticker:   "AAPL",
			TickerID: "uuid-123",
			Date:     "2026-04-08",
			RunID:    "run-123",
		},
		Error: &pipeline.StageError{Error: "States.TaskFailed", Cause: "boom"},
	}

	resp, err := closePipeline(context.Background(), event, repo, &stubStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.updateStatusCalls) != 1 {
		t.Fatalf("UpdateStatus calls = %d, want 1", len(repo.updateStatusCalls))
	}
	call := repo.updateStatusCalls[0]
	if call.id != "run-123" {
		t.Errorf("UpdateStatus id = %q, want %q", call.id, "run-123")
	}
	if call.status != domain.PipelineStatusFailed {
		t.Errorf("UpdateStatus status = %q, want %q", call.status, domain.PipelineStatusFailed)
	}
	if !strings.Contains(call.errMsg, "States.TaskFailed") || !strings.Contains(call.errMsg, "boom") {
		t.Errorf("UpdateStatus errMsg = %q, want it to contain Error and Cause", call.errMsg)
	}

	if repo.markCompletedCalls != 0 {
		t.Errorf("MarkCompleted calls = %d, want 0", repo.markCompletedCalls)
	}

	if resp.PipelineStatus != domain.PipelineStatusFailed {
		t.Errorf("PipelineStatus = %q, want %q", resp.PipelineStatus, domain.PipelineStatusFailed)
	}
}

func TestClosePipeline_FailurePath_EmptyErrorObject(t *testing.T) {
	repo := &stubRunRepo{}

	event := Event{
		TickerEvent: pipeline.TickerEvent{
			Ticker:   "AAPL",
			TickerID: "uuid-123",
			Date:     "2026-04-08",
			RunID:    "run-123",
		},
		Error: &pipeline.StageError{},
	}

	resp, err := closePipeline(context.Background(), event, repo, &stubStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.updateStatusCalls) != 1 {
		t.Fatalf("UpdateStatus calls = %d, want 1", len(repo.updateStatusCalls))
	}
	if repo.updateStatusCalls[0].errMsg != "unknown error" {
		t.Errorf("errMsg = %q, want %q", repo.updateStatusCalls[0].errMsg, "unknown error")
	}
	if resp.PipelineStatus != domain.PipelineStatusFailed {
		t.Errorf("PipelineStatus = %q, want %q", resp.PipelineStatus, domain.PipelineStatusFailed)
	}
}

func TestClosePipeline_FailurePath_TruncatesMessage(t *testing.T) {
	repo := &stubRunRepo{}

	longCause := strings.Repeat("x", 3000)
	event := Event{
		TickerEvent: pipeline.TickerEvent{
			Ticker:   "AAPL",
			TickerID: "uuid-123",
			Date:     "2026-04-08",
			RunID:    "run-123",
		},
		Error: &pipeline.StageError{Error: "States.TaskFailed", Cause: longCause},
	}

	_, err := closePipeline(context.Background(), event, repo, &stubStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.updateStatusCalls) != 1 {
		t.Fatalf("UpdateStatus calls = %d, want 1", len(repo.updateStatusCalls))
	}
	got := []rune(repo.updateStatusCalls[0].errMsg)
	if len(got) != 2000 {
		t.Errorf("errMsg length = %d, want 2000", len(got))
	}
}

func TestClosePipeline_FailurePath_UpdateStatusFails(t *testing.T) {
	repo := &stubRunRepo{updateStatusErr: fmt.Errorf("db down")}

	event := Event{
		TickerEvent: pipeline.TickerEvent{
			Ticker:   "AAPL",
			TickerID: "uuid-123",
			Date:     "2026-04-08",
			RunID:    "run-123",
		},
		Error: &pipeline.StageError{Error: "States.TaskFailed", Cause: "boom"},
	}

	_, err := closePipeline(context.Background(), event, repo, &stubStageTracker{}, discardLogger)
	if err == nil {
		t.Fatal("expected error when UpdateStatus fails")
	}
}

func TestClosePipeline_FailurePath_EmptyRunID(t *testing.T) {
	event := Event{
		TickerEvent: pipeline.TickerEvent{
			Ticker:   "AAPL",
			TickerID: "uuid-123",
			Date:     "2026-04-08",
			RunID:    "",
		},
		Error: &pipeline.StageError{Error: "States.TaskFailed", Cause: "boom"},
	}

	resp, err := closePipeline(context.Background(), event, nil, &stubStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PipelineStatus != "skipped" {
		t.Errorf("PipelineStatus = %q, want skipped", resp.PipelineStatus)
	}
}

func TestHandleRequest_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	event := Event{TickerEvent: pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08", RunID: "run-123"}}
	_, err := handleRequest(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
}
