package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/queue"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// stubFetcher returns a fixed list of tickers.
type stubFetcher struct {
	tickers []domain.Ticker
	err     error
}

func (f *stubFetcher) FetchActiveTickers(_ context.Context) ([]domain.Ticker, error) {
	return f.tickers, f.err
}

// stubPublisher captures published messages for assertions.
type stubPublisher struct {
	published []queue.TickerMessage
	err       error
}

func (p *stubPublisher) SendBatch(_ context.Context, messages []queue.TickerMessage) error {
	if p.err != nil {
		return p.err
	}
	p.published = append(p.published, messages...)
	return nil
}

func TestFetchAndPublish_Success(t *testing.T) {
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL", Name: "Apple Inc.", Market: "stocks", Active: true},
			{Ticker: "MSFT", Name: "Microsoft", Market: "stocks", Active: true},
		},
	}
	pub := &stubPublisher{}

	resp, err := fetchAndPublish(context.Background(), Event{}, fetcher, pub, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndPublish: %v", err)
	}

	if resp.TickerCount != 2 {
		t.Errorf("TickerCount = %d, want 2", resp.TickerCount)
	}
	if resp.Date == "" {
		t.Error("Date should not be empty")
	}
	if len(pub.published) != 2 {
		t.Fatalf("published %d messages, want 2", len(pub.published))
	}
	if pub.published[0].Ticker.Ticker != "AAPL" {
		t.Errorf("first ticker = %q, want %q", pub.published[0].Ticker.Ticker, "AAPL")
	}
	if pub.published[1].Ticker.Ticker != "MSFT" {
		t.Errorf("second ticker = %q, want %q", pub.published[1].Ticker.Ticker, "MSFT")
	}
}

func TestFetchAndPublish_CustomDate(t *testing.T) {
	fetcher := &stubFetcher{tickers: []domain.Ticker{{Ticker: "AAPL"}}}
	pub := &stubPublisher{}

	event := Event{Date: "2026-01-15"}
	resp, err := fetchAndPublish(context.Background(), event, fetcher, pub, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndPublish: %v", err)
	}

	if resp.Date != "2026-01-15" {
		t.Errorf("Date = %q, want %q", resp.Date, "2026-01-15")
	}
	if pub.published[0].Date != "2026-01-15" {
		t.Errorf("message date = %q, want %q", pub.published[0].Date, "2026-01-15")
	}
}

func TestFetchAndPublish_PreservesTickerFields(t *testing.T) {
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{
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
		},
	}
	pub := &stubPublisher{}

	_, err := fetchAndPublish(context.Background(), Event{}, fetcher, pub, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndPublish: %v", err)
	}

	msg := pub.published[0]
	if msg.Ticker.Name != "Apple Inc." {
		t.Errorf("name = %q, want %q", msg.Ticker.Name, "Apple Inc.")
	}
	if msg.Ticker.PrimaryExchange != "XNAS" {
		t.Errorf("primary_exchange = %q, want %q", msg.Ticker.PrimaryExchange, "XNAS")
	}
	if msg.Ticker.CIK != "0000320193" {
		t.Errorf("cik = %q, want %q", msg.Ticker.CIK, "0000320193")
	}
}

func TestFetchAndPublish_FetchError(t *testing.T) {
	fetcher := &stubFetcher{err: fmt.Errorf("api timeout")}
	pub := &stubPublisher{}

	_, err := fetchAndPublish(context.Background(), Event{}, fetcher, pub, discardLogger)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(pub.published) != 0 {
		t.Errorf("should not publish on fetch error, got %d messages", len(pub.published))
	}
}

func TestFetchAndPublish_PublishError(t *testing.T) {
	fetcher := &stubFetcher{tickers: []domain.Ticker{{Ticker: "AAPL"}}}
	pub := &stubPublisher{err: fmt.Errorf("sqs send failed")}

	_, err := fetchAndPublish(context.Background(), Event{}, fetcher, pub, discardLogger)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFetchAndPublish_EmptyTickers(t *testing.T) {
	fetcher := &stubFetcher{tickers: []domain.Ticker{}}
	pub := &stubPublisher{}

	resp, err := fetchAndPublish(context.Background(), Event{}, fetcher, pub, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndPublish: %v", err)
	}
	if resp.TickerCount != 0 {
		t.Errorf("TickerCount = %d, want 0", resp.TickerCount)
	}
	if len(pub.published) != 0 {
		t.Errorf("published %d messages, want 0", len(pub.published))
	}
}

func TestHandleRequest_MissingSQSQueueURL(t *testing.T) {
	t.Setenv("MASSIVE_API_KEY", "test-key")
	t.Setenv("SQS_QUEUE_URL", "")

	_, err := handleRequest(context.Background(), Event{})
	if err == nil {
		t.Fatal("expected error for missing SQS_QUEUE_URL")
	}
}

func TestHandleRequest_MissingAPIKey(t *testing.T) {
	t.Setenv("MASSIVE_API_KEY", "")
	t.Setenv("SQS_QUEUE_URL", "https://sqs.us-east-1.amazonaws.com/123/test")

	_, err := handleRequest(context.Background(), Event{})
	if err == nil {
		t.Fatal("expected error for missing MASSIVE_API_KEY")
	}
}
