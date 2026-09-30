package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
)

func TestFetchAndPublish_PublishesUpsertedIDs(t *testing.T) {
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL"},
			{Ticker: "MSFT"},
		},
	}
	pub := &stubPublisher{}
	upserter := &stubUpserter{}

	_, err := fetchAndPublish(context.Background(), Event{}, fetcher, upserter, pub, nil, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndPublish: %v", err)
	}

	if len(pub.published) != 2 {
		t.Fatalf("published %d messages, want 2", len(pub.published))
	}
	for _, msg := range pub.published {
		if msg.ID == "" {
			t.Fatalf("published message for %s has empty ID", msg.Ticker.Ticker)
		}
		want := "uuid-" + msg.Ticker.Ticker
		if msg.ID != want {
			t.Errorf("ID for %s = %q, want %q", msg.Ticker.Ticker, msg.ID, want)
		}
	}
}

func TestFetchAndPublish_UpsertError(t *testing.T) {
	fetcher := &stubFetcher{tickers: []domain.Ticker{{Ticker: "AAPL"}}}
	pub := &stubPublisher{}
	upserter := &stubUpserter{err: fmt.Errorf("db down")}

	_, err := fetchAndPublish(context.Background(), Event{}, fetcher, upserter, pub, nil, discardLogger)
	if err == nil {
		t.Fatal("expected error for upsert failure")
	}
	if len(pub.published) != 0 {
		t.Errorf("should not publish when upsert fails, got %d messages", len(pub.published))
	}
}

func TestFetchAndPublish_MissingIDAfterUpsert(t *testing.T) {
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL"},
			{Ticker: "MSFT"},
		},
	}
	pub := &stubPublisher{}
	upserter := &stubUpserter{skipIDFor: "MSFT"}

	_, err := fetchAndPublish(context.Background(), Event{}, fetcher, upserter, pub, nil, discardLogger)
	if err == nil {
		t.Fatal("expected error when upsert leaves an ID empty")
	}
	if len(pub.published) != 0 {
		t.Errorf("should not publish when an ID is missing, got %d messages", len(pub.published))
	}
}
