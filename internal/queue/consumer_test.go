package queue

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/profitify/profitify-backend/internal/domain"
)

func TestParseSQSEvent_Success(t *testing.T) {
	msg1 := TickerMessage{
		Ticker: domain.Ticker{Ticker: "AAPL", Name: "Apple Inc."},
		Date:   "2026-06-07",
	}
	msg2 := TickerMessage{
		Ticker: domain.Ticker{Ticker: "MSFT", Name: "Microsoft"},
		Date:   "2026-06-07",
	}
	body1, _ := json.Marshal(msg1)
	body2, _ := json.Marshal(msg2)

	event := events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "id-1", Body: string(body1)},
			{MessageId: "id-2", Body: string(body2)},
		},
	}

	result := ParseSQSEvent(event)

	if len(result.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(result.Messages))
	}
	if len(result.Failures) != 0 {
		t.Fatalf("failures = %d, want 0", len(result.Failures))
	}
	if result.Messages[0].Ticker.Ticker != "AAPL" {
		t.Errorf("messages[0].Ticker = %q, want %q", result.Messages[0].Ticker.Ticker, "AAPL")
	}
	if result.Messages[1].Ticker.Ticker != "MSFT" {
		t.Errorf("messages[1].Ticker = %q, want %q", result.Messages[1].Ticker.Ticker, "MSFT")
	}
}

func TestParseSQSEvent_EmptyEvent(t *testing.T) {
	event := events.SQSEvent{Records: []events.SQSMessage{}}

	result := ParseSQSEvent(event)

	if len(result.Messages) != 0 {
		t.Errorf("messages = %d, want 0", len(result.Messages))
	}
	if len(result.Failures) != 0 {
		t.Errorf("failures = %d, want 0", len(result.Failures))
	}
}

func TestParseSQSEvent_InvalidJSON(t *testing.T) {
	event := events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "bad-1", Body: "not-json"},
		},
	}

	result := ParseSQSEvent(event)

	if len(result.Messages) != 0 {
		t.Errorf("messages = %d, want 0", len(result.Messages))
	}
	if len(result.Failures) != 1 {
		t.Fatalf("failures = %d, want 1", len(result.Failures))
	}
	if result.Failures[0].ItemIdentifier != "bad-1" {
		t.Errorf("failure ItemIdentifier = %q, want %q", result.Failures[0].ItemIdentifier, "bad-1")
	}
}

func TestParseSQSEvent_PartialFailure(t *testing.T) {
	msg1 := TickerMessage{
		Ticker: domain.Ticker{Ticker: "AAPL"},
		Date:   "2026-06-07",
	}
	msg2 := TickerMessage{
		Ticker: domain.Ticker{Ticker: "GOOG"},
		Date:   "2026-06-07",
	}
	body1, _ := json.Marshal(msg1)
	body2, _ := json.Marshal(msg2)

	event := events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "ok-1", Body: string(body1)},
			{MessageId: "bad-1", Body: "{invalid"},
			{MessageId: "ok-2", Body: string(body2)},
		},
	}

	result := ParseSQSEvent(event)

	if len(result.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(result.Messages))
	}
	if len(result.Failures) != 1 {
		t.Fatalf("failures = %d, want 1", len(result.Failures))
	}
	if result.Failures[0].ItemIdentifier != "bad-1" {
		t.Errorf("failure ItemIdentifier = %q, want %q", result.Failures[0].ItemIdentifier, "bad-1")
	}
	if result.Messages[0].Ticker.Ticker != "AAPL" {
		t.Errorf("messages[0].Ticker = %q, want %q", result.Messages[0].Ticker.Ticker, "AAPL")
	}
	if result.Messages[1].Ticker.Ticker != "GOOG" {
		t.Errorf("messages[1].Ticker = %q, want %q", result.Messages[1].Ticker.Ticker, "GOOG")
	}
}

func TestParseSQSEvent_PreservesAllFields(t *testing.T) {
	msg := TickerMessage{
		Ticker: domain.Ticker{
			ID:              "abc-123",
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
			DelistedUTC:     "",
			Sector:          "Technology",
		},
		Date: "2026-06-07",
	}
	body, _ := json.Marshal(msg)

	event := events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "id-1", Body: string(body)},
		},
	}

	result := ParseSQSEvent(event)

	if len(result.Messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(result.Messages))
	}

	got := result.Messages[0]
	if got.Ticker.ID != "abc-123" {
		t.Errorf("ID = %q, want %q", got.Ticker.ID, "abc-123")
	}
	if got.Ticker.Ticker != "AAPL" {
		t.Errorf("Ticker = %q, want %q", got.Ticker.Ticker, "AAPL")
	}
	if got.Ticker.Name != "Apple Inc." {
		t.Errorf("Name = %q, want %q", got.Ticker.Name, "Apple Inc.")
	}
	if got.Ticker.Market != "stocks" {
		t.Errorf("Market = %q, want %q", got.Ticker.Market, "stocks")
	}
	if got.Ticker.PrimaryExchange != "XNAS" {
		t.Errorf("PrimaryExchange = %q, want %q", got.Ticker.PrimaryExchange, "XNAS")
	}
	if got.Ticker.Type != "CS" {
		t.Errorf("Type = %q, want %q", got.Ticker.Type, "CS")
	}
	if !got.Ticker.Active {
		t.Error("Active = false, want true")
	}
	if got.Ticker.CurrencyName != "usd" {
		t.Errorf("CurrencyName = %q, want %q", got.Ticker.CurrencyName, "usd")
	}
	if got.Ticker.Locale != "us" {
		t.Errorf("Locale = %q, want %q", got.Ticker.Locale, "us")
	}
	if got.Ticker.CIK != "0000320193" {
		t.Errorf("CIK = %q, want %q", got.Ticker.CIK, "0000320193")
	}
	if got.Ticker.ListDate != "1980-12-12" {
		t.Errorf("ListDate = %q, want %q", got.Ticker.ListDate, "1980-12-12")
	}
	if got.Ticker.Sector != "Technology" {
		t.Errorf("Sector = %q, want %q", got.Ticker.Sector, "Technology")
	}
	if got.Date != "2026-06-07" {
		t.Errorf("Date = %q, want %q", got.Date, "2026-06-07")
	}
}
