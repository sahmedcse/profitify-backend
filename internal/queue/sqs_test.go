package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/profitify/profitify-backend/internal/domain"
)

// stubSQSClient captures SendMessageBatch calls for assertions.
type stubSQSClient struct {
	calls  []sqs.SendMessageBatchInput
	errIdx int // if >= 0, fail the batch at this index
}

func (s *stubSQSClient) SendMessageBatch(ctx context.Context, input *sqs.SendMessageBatchInput, _ ...func(*sqs.Options)) (*sqs.SendMessageBatchOutput, error) {
	idx := len(s.calls)
	s.calls = append(s.calls, *input)
	if s.errIdx >= 0 && idx == s.errIdx {
		return nil, fmt.Errorf("sqs send error")
	}
	return &sqs.SendMessageBatchOutput{}, nil
}

func TestSendBatch_Empty(t *testing.T) {
	client := &stubSQSClient{errIdx: -1}
	pub := &Publisher{client: client, queueURL: "https://sqs.us-east-1.amazonaws.com/123/test"}

	err := pub.SendBatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.calls) != 0 {
		t.Errorf("expected 0 API calls, got %d", len(client.calls))
	}
}

func TestSendBatch_SingleBatch(t *testing.T) {
	client := &stubSQSClient{errIdx: -1}
	pub := &Publisher{client: client, queueURL: "https://sqs.us-east-1.amazonaws.com/123/test"}

	messages := []TickerMessage{
		{Ticker: domain.Ticker{Ticker: "AAPL", Name: "Apple Inc."}, Date: "2026-06-07"},
		{Ticker: domain.Ticker{Ticker: "MSFT", Name: "Microsoft"}, Date: "2026-06-07"},
	}

	err := pub.SendBatch(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(client.calls) != 1 {
		t.Fatalf("expected 1 API call, got %d", len(client.calls))
	}

	entries := client.calls[0].Entries
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Verify first message body is valid JSON with correct ticker.
	var msg TickerMessage
	if err := json.Unmarshal([]byte(*entries[0].MessageBody), &msg); err != nil {
		t.Fatalf("failed to unmarshal message body: %v", err)
	}
	if msg.Ticker.Ticker != "AAPL" {
		t.Errorf("ticker = %q, want %q", msg.Ticker.Ticker, "AAPL")
	}
	if msg.Date != "2026-06-07" {
		t.Errorf("date = %q, want %q", msg.Date, "2026-06-07")
	}
}

func TestSendBatch_MultipleChunks(t *testing.T) {
	client := &stubSQSClient{errIdx: -1}
	pub := &Publisher{client: client, queueURL: "https://sqs.us-east-1.amazonaws.com/123/test"}

	// 25 messages should produce 3 batches (10 + 10 + 5).
	messages := make([]TickerMessage, 25)
	for i := range messages {
		messages[i] = TickerMessage{
			Ticker: domain.Ticker{Ticker: fmt.Sprintf("T%d", i)},
			Date:   "2026-06-07",
		}
	}

	err := pub.SendBatch(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(client.calls) != 3 {
		t.Fatalf("expected 3 API calls, got %d", len(client.calls))
	}
	if len(client.calls[0].Entries) != 10 {
		t.Errorf("first batch entries = %d, want 10", len(client.calls[0].Entries))
	}
	if len(client.calls[1].Entries) != 10 {
		t.Errorf("second batch entries = %d, want 10", len(client.calls[1].Entries))
	}
	if len(client.calls[2].Entries) != 5 {
		t.Errorf("third batch entries = %d, want 5", len(client.calls[2].Entries))
	}
}

func TestSendBatch_ExactlyTen(t *testing.T) {
	client := &stubSQSClient{errIdx: -1}
	pub := &Publisher{client: client, queueURL: "https://sqs.us-east-1.amazonaws.com/123/test"}

	messages := make([]TickerMessage, 10)
	for i := range messages {
		messages[i] = TickerMessage{
			Ticker: domain.Ticker{Ticker: fmt.Sprintf("T%d", i)},
			Date:   "2026-06-07",
		}
	}

	err := pub.SendBatch(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.calls) != 1 {
		t.Fatalf("expected 1 API call, got %d", len(client.calls))
	}
}

func TestSendBatch_APIError(t *testing.T) {
	client := &stubSQSClient{errIdx: 0}
	pub := &Publisher{client: client, queueURL: "https://sqs.us-east-1.amazonaws.com/123/test"}

	messages := []TickerMessage{
		{Ticker: domain.Ticker{Ticker: "AAPL"}, Date: "2026-06-07"},
	}

	err := pub.SendBatch(context.Background(), messages)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSendBatch_PartialFailure(t *testing.T) {
	// Simulate SQS returning failed entries in the response.
	client := &partialFailClient{}
	pub := &Publisher{client: client, queueURL: "https://sqs.us-east-1.amazonaws.com/123/test"}

	messages := []TickerMessage{
		{Ticker: domain.Ticker{Ticker: "AAPL"}, Date: "2026-06-07"},
		{Ticker: domain.Ticker{Ticker: "MSFT"}, Date: "2026-06-07"},
	}

	err := pub.SendBatch(context.Background(), messages)
	if err == nil {
		t.Fatal("expected error for partial failure, got nil")
	}
}

// partialFailClient returns a response with one failed entry.
type partialFailClient struct{}

func (p *partialFailClient) SendMessageBatch(_ context.Context, input *sqs.SendMessageBatchInput, _ ...func(*sqs.Options)) (*sqs.SendMessageBatchOutput, error) {
	return &sqs.SendMessageBatchOutput{
		Failed: []sqstypes.BatchResultErrorEntry{
			{
				Id:      input.Entries[0].Id,
				Code:    strPtr("InternalError"),
				Message: strPtr("something went wrong"),
			},
		},
	}, nil
}

func TestSendBatch_MessageContainsAllFields(t *testing.T) {
	client := &stubSQSClient{errIdx: -1}
	pub := &Publisher{client: client, queueURL: "https://sqs.us-east-1.amazonaws.com/123/test"}

	messages := []TickerMessage{
		{
			Ticker: domain.Ticker{
				Ticker:          "AAPL",
				Name:            "Apple Inc.",
				Market:          "stocks",
				PrimaryExchange: "XNAS",
				Type:            "CS",
				Active:          true,
				CurrencyName:    "usd",
				Locale:          "us",
				CIK:             "0000320193",
				ListDate:        "1980-12-12",
			},
			Date: "2026-06-07",
		},
	}

	err := pub.SendBatch(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(*client.calls[0].Entries[0].MessageBody), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	checks := map[string]string{
		"ticker":           "AAPL",
		"name":             "Apple Inc.",
		"market":           "stocks",
		"primary_exchange": "XNAS",
		"type":             "CS",
		"currency_name":    "usd",
		"locale":           "us",
		"cik":              "0000320193",
		"list_date":        "1980-12-12",
		"date":             "2026-06-07",
	}
	for key, want := range checks {
		got, ok := msg[key].(string)
		if !ok || got != want {
			t.Errorf("msg[%q] = %v, want %q", key, msg[key], want)
		}
	}

	active, ok := msg["active"].(bool)
	if !ok || !active {
		t.Errorf("msg[active] = %v, want true", msg["active"])
	}
}

func TestNewPublisher(t *testing.T) {
	// NewPublisher requires valid AWS config which we can't test here.
	// Verify it returns an error with invalid endpoint (no credentials).
	// This is mainly a smoke test that the constructor wires up correctly.
	// Skipped because it needs AWS credentials or endpoint override.
	t.Skip("requires AWS credentials or localstack")
}

func strPtr(s string) *string { return &s }
