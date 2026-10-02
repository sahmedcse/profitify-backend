package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
)

func TestFetchAndPublish_Success(t *testing.T) {
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL", Name: "Apple Inc.", Market: "stocks", Active: true},
			{Ticker: "MSFT", Name: "Microsoft", Market: "stocks", Active: true},
		},
	}
	pub := &stubPublisher{}
	upserter := &stubUpserter{}

	resp, err := fetchAndPublish(context.Background(), Event{}, fetcher, upserter, pub, nil, discardLogger)
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
	upserter := &stubUpserter{}

	event := Event{Date: "2026-01-15"}
	resp, err := fetchAndPublish(context.Background(), event, fetcher, upserter, pub, nil, discardLogger)
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
	upserter := &stubUpserter{}

	_, err := fetchAndPublish(context.Background(), Event{}, fetcher, upserter, pub, nil, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndPublish: %v", err)
	}

	msg := pub.published[0]
	if msg.Name != "Apple Inc." {
		t.Errorf("name = %q, want %q", msg.Name, "Apple Inc.")
	}
	if msg.PrimaryExchange != "XNAS" {
		t.Errorf("primary_exchange = %q, want %q", msg.PrimaryExchange, "XNAS")
	}
	if msg.CIK != "0000320193" {
		t.Errorf("cik = %q, want %q", msg.CIK, "0000320193")
	}
}

func TestFetchAndPublish_FetchError(t *testing.T) {
	fetcher := &stubFetcher{err: fmt.Errorf("api timeout")}
	pub := &stubPublisher{}
	upserter := &stubUpserter{}

	_, err := fetchAndPublish(context.Background(), Event{}, fetcher, upserter, pub, nil, discardLogger)
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
	upserter := &stubUpserter{}

	_, err := fetchAndPublish(context.Background(), Event{}, fetcher, upserter, pub, nil, discardLogger)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFetchAndPublish_EmptyTickers(t *testing.T) {
	fetcher := &stubFetcher{tickers: []domain.Ticker{}}
	pub := &stubPublisher{}
	upserter := &stubUpserter{}

	resp, err := fetchAndPublish(context.Background(), Event{}, fetcher, upserter, pub, nil, discardLogger)
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
