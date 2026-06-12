package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/profitify/profitify-backend/internal/domain"
)

const maxBatchSize = 10

// TickerMessage is the SQS message payload for a single ticker.
type TickerMessage struct {
	domain.Ticker
	Date string `json:"date"`
}

// sqsClient abstracts the SQS API for testing.
type sqsClient interface {
	SendMessageBatch(ctx context.Context, input *sqs.SendMessageBatchInput, opts ...func(*sqs.Options)) (*sqs.SendMessageBatchOutput, error)
}

// Publisher sends ticker messages to an SQS queue.
type Publisher struct {
	client   sqsClient
	queueURL string
}

// NewPublisher creates a Publisher with a real SQS client using default AWS config.
func NewPublisher(ctx context.Context, queueURL string) (*Publisher, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	return &Publisher{
		client:   sqs.NewFromConfig(cfg),
		queueURL: queueURL,
	}, nil
}

// SendBatch publishes messages to SQS in chunks of 10 (the SQS maximum).
func (p *Publisher) SendBatch(ctx context.Context, messages []TickerMessage) error {
	for i := 0; i < len(messages); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(messages) {
			end = len(messages)
		}
		chunk := messages[i:end]

		entries := make([]sqstypes.SendMessageBatchRequestEntry, len(chunk))
		for j, msg := range chunk {
			body, err := json.Marshal(msg)
			if err != nil {
				return fmt.Errorf("marshaling message for %s: %w", msg.Ticker.Ticker, err)
			}
			bodyStr := string(body)
			id := strconv.Itoa(i + j)
			entries[j] = sqstypes.SendMessageBatchRequestEntry{
				Id:          &id,
				MessageBody: &bodyStr,
			}
		}

		out, err := p.client.SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
			QueueUrl: &p.queueURL,
			Entries:  entries,
		})
		if err != nil {
			return fmt.Errorf("sending SQS batch at offset %d: %w", i, err)
		}
		if len(out.Failed) > 0 {
			return fmt.Errorf("SQS batch had %d failures, first: %s",
				len(out.Failed), *out.Failed[0].Message)
		}
	}
	return nil
}
