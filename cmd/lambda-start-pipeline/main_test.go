package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/sfn"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/queue"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// --- stubs ---

type stubRunRepo struct {
	created *domain.PipelineRun
	err     error
}

func (s *stubRunRepo) Create(_ context.Context, run *domain.PipelineRun) (*domain.PipelineRun, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := *run
	out.ID = "run-001"
	s.created = &out
	return &out, nil
}

func (s *stubRunRepo) UpdateStatus(_ context.Context, id, status, errMsg string) error { return nil }
func (s *stubRunRepo) UpdateSFNArn(_ context.Context, id, arn string) error             { return nil }

type stubStageRepo struct {
	runningCalls   []stageCall
	completedCalls []stageCall
	failedCalls    []stageFailCall
	markRunningErr error
}

type stageCall struct {
	runID, tickerID, stage string
}

type stageFailCall struct {
	runID, tickerID, stage, errorMessage string
}

func (s *stubStageRepo) MarkRunning(_ context.Context, runID, tickerID, stage string) (string, error) {
	s.runningCalls = append(s.runningCalls, stageCall{runID, tickerID, stage})
	if s.markRunningErr != nil {
		return "", s.markRunningErr
	}
	return "stage-001", nil
}

func (s *stubStageRepo) MarkCompleted(_ context.Context, runID, tickerID, stage string) error {
	s.completedCalls = append(s.completedCalls, stageCall{runID, tickerID, stage})
	return nil
}

func (s *stubStageRepo) MarkFailed(_ context.Context, runID, tickerID, stage, errorMessage string) error {
	s.failedCalls = append(s.failedCalls, stageFailCall{runID, tickerID, stage, errorMessage})
	return nil
}

type stubSFNClient struct {
	input *sfn.StartExecutionInput
	err   error
}

func (s *stubSFNClient) StartExecution(_ context.Context, input *sfn.StartExecutionInput, _ ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error) {
	s.input = input
	if s.err != nil {
		return nil, s.err
	}
	arn := "arn:aws:states:us-east-1:123:execution:pipeline:run-001"
	return &sfn.StartExecutionOutput{ExecutionArn: &arn}, nil
}

// --- tracking stubs for assertion ---

type trackingRunRepo struct {
	stubRunRepo
	statusUpdates []statusUpdate
	arnUpdates    []string
}

type statusUpdate struct {
	id, status, errMsg string
}

func (t *trackingRunRepo) UpdateStatus(_ context.Context, id, status, errMsg string) error {
	t.statusUpdates = append(t.statusUpdates, statusUpdate{id, status, errMsg})
	return nil
}

func (t *trackingRunRepo) UpdateSFNArn(_ context.Context, id, arn string) error {
	t.arnUpdates = append(t.arnUpdates, arn)
	return nil
}

// --- helpers ---

func makeSQSEvent(messages ...queue.TickerMessage) events.SQSEvent {
	var records []events.SQSMessage
	for i, msg := range messages {
		body, _ := json.Marshal(msg)
		records = append(records, events.SQSMessage{
			MessageId: fmt.Sprintf("msg-%d", i),
			Body:      string(body),
		})
	}
	return events.SQSEvent{Records: records}
}

func validMessage() queue.TickerMessage {
	return queue.TickerMessage{
		Ticker: domain.Ticker{
			ID:     "ticker-uuid-1",
			Ticker: "AAPL",
			Name:   "Apple Inc.",
		},
		Date: "2026-06-12",
	}
}

// --- tests ---

func TestStartPipeline_HappyPath(t *testing.T) {
	runs := &trackingRunRepo{}
	stages := &stubStageRepo{}
	sfnClient := &stubSFNClient{}

	event := makeSQSEvent(validMessage())
	resp, err := startPipeline(context.Background(), event, runs, stages, sfnClient, "arn:sfn:test", discardLogger)
	if err != nil {
		t.Fatalf("startPipeline: %v", err)
	}

	if len(resp.BatchItemFailures) != 0 {
		t.Errorf("expected 0 failures, got %d", len(resp.BatchItemFailures))
	}

	// Run created with correct fields.
	if runs.created == nil {
		t.Fatal("expected run to be created")
	}
	if runs.created.TickerID != "ticker-uuid-1" {
		t.Errorf("TickerID = %q, want %q", runs.created.TickerID, "ticker-uuid-1")
	}
	if runs.created.Ticker != "AAPL" {
		t.Errorf("Ticker = %q, want %q", runs.created.Ticker, "AAPL")
	}
	if runs.created.Date != "2026-06-12" {
		t.Errorf("Date = %q, want %q", runs.created.Date, "2026-06-12")
	}

	// start_pipeline stage marked running then completed.
	if len(stages.runningCalls) != 1 {
		t.Fatalf("MarkRunning calls = %d, want 1", len(stages.runningCalls))
	}
	if stages.runningCalls[0].stage != domain.StageStartPipeline {
		t.Errorf("MarkRunning stage = %q, want %q", stages.runningCalls[0].stage, domain.StageStartPipeline)
	}
	if len(stages.completedCalls) != 1 {
		t.Fatalf("MarkCompleted calls = %d, want 1", len(stages.completedCalls))
	}
	if stages.completedCalls[0].stage != domain.StageStartPipeline {
		t.Errorf("MarkCompleted stage = %q, want %q", stages.completedCalls[0].stage, domain.StageStartPipeline)
	}

	// SFN started.
	if sfnClient.input == nil {
		t.Fatal("SFN StartExecution not called")
	}

	// ARN saved.
	if len(runs.arnUpdates) != 1 {
		t.Fatalf("arnUpdates = %d, want 1", len(runs.arnUpdates))
	}
	if runs.arnUpdates[0] != "arn:aws:states:us-east-1:123:execution:pipeline:run-001" {
		t.Errorf("saved ARN = %q", runs.arnUpdates[0])
	}
}

func TestStartPipeline_RunCreateFails(t *testing.T) {
	runs := &trackingRunRepo{stubRunRepo: stubRunRepo{err: fmt.Errorf("db down")}}
	stages := &stubStageRepo{}
	sfnClient := &stubSFNClient{}

	event := makeSQSEvent(validMessage())
	resp, err := startPipeline(context.Background(), event, runs, stages, sfnClient, "arn:sfn:test", discardLogger)
	if err != nil {
		t.Fatalf("startPipeline should not return top-level error: %v", err)
	}

	if len(resp.BatchItemFailures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(resp.BatchItemFailures))
	}

	// No stage or SFN calls.
	if len(stages.runningCalls) != 0 {
		t.Errorf("MarkRunning should not be called on run create failure")
	}
	if sfnClient.input != nil {
		t.Error("SFN should not be called on run create failure")
	}
}

func TestStartPipeline_SFNStartFails(t *testing.T) {
	runs := &trackingRunRepo{}
	stages := &stubStageRepo{}
	sfnClient := &stubSFNClient{err: fmt.Errorf("sfn throttle")}

	event := makeSQSEvent(validMessage())
	resp, err := startPipeline(context.Background(), event, runs, stages, sfnClient, "arn:sfn:test", discardLogger)
	if err != nil {
		t.Fatalf("startPipeline: %v", err)
	}

	if len(resp.BatchItemFailures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(resp.BatchItemFailures))
	}

	// Run should be marked failed.
	if len(runs.statusUpdates) == 0 {
		t.Fatal("expected UpdateStatus to be called")
	}
	if runs.statusUpdates[0].status != domain.PipelineStatusFailed {
		t.Errorf("status = %q, want %q", runs.statusUpdates[0].status, domain.PipelineStatusFailed)
	}

	// Stage should be marked failed.
	if len(stages.failedCalls) != 1 {
		t.Fatalf("MarkFailed calls = %d, want 1", len(stages.failedCalls))
	}
	if stages.failedCalls[0].stage != domain.StageStartPipeline {
		t.Errorf("MarkFailed stage = %q, want %q", stages.failedCalls[0].stage, domain.StageStartPipeline)
	}
}

func TestStartPipeline_MarkRunningFails(t *testing.T) {
	runs := &trackingRunRepo{}
	stages := &stubStageRepo{markRunningErr: fmt.Errorf("stage insert error")}
	sfnClient := &stubSFNClient{}

	event := makeSQSEvent(validMessage())
	resp, err := startPipeline(context.Background(), event, runs, stages, sfnClient, "arn:sfn:test", discardLogger)
	if err != nil {
		t.Fatalf("startPipeline: %v", err)
	}

	if len(resp.BatchItemFailures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(resp.BatchItemFailures))
	}

	// Run should be marked failed.
	if len(runs.statusUpdates) == 0 {
		t.Fatal("expected UpdateStatus to be called")
	}
	if runs.statusUpdates[0].status != domain.PipelineStatusFailed {
		t.Errorf("status = %q, want %q", runs.statusUpdates[0].status, domain.PipelineStatusFailed)
	}

	// SFN should not be called.
	if sfnClient.input != nil {
		t.Error("SFN should not be called when stage tracking fails")
	}
}

func TestStartPipeline_UpdateSFNArnFails(t *testing.T) {
	runs := &trackingRunRepo{}
	// Override UpdateSFNArn to fail.
	failArnRepo := &failArnRunRepo{trackingRunRepo: runs}
	stages := &stubStageRepo{}
	sfnClient := &stubSFNClient{}

	event := makeSQSEvent(validMessage())
	resp, err := startPipeline(context.Background(), event, failArnRepo, stages, sfnClient, "arn:sfn:test", discardLogger)
	if err != nil {
		t.Fatalf("startPipeline: %v", err)
	}

	// Non-fatal: no batch failures.
	if len(resp.BatchItemFailures) != 0 {
		t.Errorf("expected 0 failures (UpdateSFNArn is non-fatal), got %d", len(resp.BatchItemFailures))
	}

	// Stage still completed despite ARN failure.
	if len(stages.completedCalls) != 1 {
		t.Fatalf("MarkCompleted calls = %d, want 1", len(stages.completedCalls))
	}
}

// failArnRunRepo wraps trackingRunRepo but fails on UpdateSFNArn.
type failArnRunRepo struct {
	*trackingRunRepo
}

func (f *failArnRunRepo) UpdateSFNArn(_ context.Context, id, arn string) error {
	return fmt.Errorf("arn update failed")
}

func TestStartPipeline_MalformedSQSBody(t *testing.T) {
	event := events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "bad-msg", Body: "not json"},
		},
	}

	runs := &trackingRunRepo{}
	stages := &stubStageRepo{}
	sfnClient := &stubSFNClient{}

	resp, err := startPipeline(context.Background(), event, runs, stages, sfnClient, "arn:sfn:test", discardLogger)
	if err != nil {
		t.Fatalf("startPipeline: %v", err)
	}

	if len(resp.BatchItemFailures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(resp.BatchItemFailures))
	}
	if resp.BatchItemFailures[0].ItemIdentifier != "bad-msg" {
		t.Errorf("failed message ID = %q, want %q", resp.BatchItemFailures[0].ItemIdentifier, "bad-msg")
	}

	// No run created for malformed message.
	if runs.created != nil {
		t.Error("run should not be created for malformed message")
	}
}

func TestStartPipeline_BatchItemFailures(t *testing.T) {
	// First message succeeds, second fails (run create error).
	conditionalRepo := &conditionalRunRepo{
		failOn: 1, // fail on second call (0-indexed)
	}
	stages := &stubStageRepo{}
	sfnClient := &stubSFNClient{}

	msg1 := validMessage()
	msg2 := validMessage()
	msg2.Ticker.ID = "ticker-uuid-2"
	msg2.Ticker.Ticker = "MSFT"

	event := makeSQSEvent(msg1, msg2)
	resp, err := startPipeline(context.Background(), event, conditionalRepo, stages, sfnClient, "arn:sfn:test", discardLogger)
	if err != nil {
		t.Fatalf("startPipeline: %v", err)
	}

	// Only the second message should fail.
	if len(resp.BatchItemFailures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(resp.BatchItemFailures))
	}
	if resp.BatchItemFailures[0].ItemIdentifier != "msg-1" {
		t.Errorf("failed message ID = %q, want %q", resp.BatchItemFailures[0].ItemIdentifier, "msg-1")
	}

	// First message's stage should be tracked.
	if len(stages.runningCalls) != 1 {
		t.Errorf("MarkRunning calls = %d, want 1 (only from successful message)", len(stages.runningCalls))
	}
}

// conditionalRunRepo fails Create on a specific call index.
type conditionalRunRepo struct {
	callCount int
	failOn    int
	created   *domain.PipelineRun
}

func (c *conditionalRunRepo) Create(_ context.Context, run *domain.PipelineRun) (*domain.PipelineRun, error) {
	idx := c.callCount
	c.callCount++
	if idx == c.failOn {
		return nil, fmt.Errorf("deliberate failure on call %d", idx)
	}
	out := *run
	out.ID = fmt.Sprintf("run-%03d", idx)
	c.created = &out
	return &out, nil
}

func (c *conditionalRunRepo) UpdateStatus(_ context.Context, id, status, errMsg string) error {
	return nil
}

func (c *conditionalRunRepo) UpdateSFNArn(_ context.Context, id, arn string) error {
	return nil
}

func TestStartPipeline_SFNInput(t *testing.T) {
	runs := &trackingRunRepo{}
	stages := &stubStageRepo{}
	sfnClient := &stubSFNClient{}

	event := makeSQSEvent(validMessage())
	_, err := startPipeline(context.Background(), event, runs, stages, sfnClient, "arn:sfn:my-pipeline", discardLogger)
	if err != nil {
		t.Fatalf("startPipeline: %v", err)
	}

	if sfnClient.input == nil {
		t.Fatal("SFN not called")
	}
	if *sfnClient.input.StateMachineArn != "arn:sfn:my-pipeline" {
		t.Errorf("StateMachineArn = %q, want %q", *sfnClient.input.StateMachineArn, "arn:sfn:my-pipeline")
	}

	// Verify the SFN input is a valid TickerEvent JSON.
	var tickerEvent map[string]string
	if err := json.Unmarshal([]byte(*sfnClient.input.Input), &tickerEvent); err != nil {
		t.Fatalf("SFN input is not valid JSON: %v", err)
	}
	if tickerEvent["ticker"] != "AAPL" {
		t.Errorf("SFN input ticker = %q, want %q", tickerEvent["ticker"], "AAPL")
	}
	if tickerEvent["ticker_id"] != "ticker-uuid-1" {
		t.Errorf("SFN input ticker_id = %q, want %q", tickerEvent["ticker_id"], "ticker-uuid-1")
	}
	if tickerEvent["run_id"] != "run-001" {
		t.Errorf("SFN input run_id = %q, want %q", tickerEvent["run_id"], "run-001")
	}
	if tickerEvent["date"] != "2026-06-12" {
		t.Errorf("SFN input date = %q, want %q", tickerEvent["date"], "2026-06-12")
	}
}
