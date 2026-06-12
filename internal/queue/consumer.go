package queue

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

// ParseResult holds the output of parsing an SQS event.
type ParseResult struct {
	Messages []TickerMessage
	Failures []events.SQSBatchItemFailure // MessageIDs that failed to parse
}

// ParseSQSEvent extracts TickerMessages from an SQS Lambda trigger event.
// Successfully parsed messages are returned in Messages.
// Records that fail to unmarshal are returned as Failures (for partial batch failure reporting).
func ParseSQSEvent(event events.SQSEvent) ParseResult {
	var result ParseResult
	for _, record := range event.Records {
		var msg TickerMessage
		if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
			result.Failures = append(result.Failures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
			continue
		}
		result.Messages = append(result.Messages, msg)
	}
	return result
}
