package main

import (
	"context"
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
)

func TestFetchAndPublish_Allowlist(t *testing.T) {
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL", Name: "Apple Inc."},
			{Ticker: "MSFT", Name: "Microsoft"},
			{Ticker: "GOOG", Name: "Alphabet"},
			{Ticker: "TSLA", Name: "Tesla"},
		},
	}
	pub := &stubPublisher{}
	upserter := &stubUpserter{}

	allowlist := []string{"AAPL", "TSLA"}
	resp, err := fetchAndPublish(context.Background(), Event{}, fetcher, upserter, pub, allowlist, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndPublish: %v", err)
	}

	if resp.TickerCount != 2 {
		t.Errorf("TickerCount = %d, want 2", resp.TickerCount)
	}
	if len(pub.published) != 2 {
		t.Fatalf("published %d messages, want 2", len(pub.published))
	}
	if pub.published[0].Ticker.Ticker != "AAPL" {
		t.Errorf("first ticker = %q, want %q", pub.published[0].Ticker.Ticker, "AAPL")
	}
	if pub.published[1].Ticker.Ticker != "TSLA" {
		t.Errorf("second ticker = %q, want %q", pub.published[1].Ticker.Ticker, "TSLA")
	}
}

func TestFetchAndPublish_AllowlistEmpty(t *testing.T) {
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL"},
			{Ticker: "MSFT"},
			{Ticker: "GOOG"},
		},
	}
	pub := &stubPublisher{}
	upserter := &stubUpserter{}

	resp, err := fetchAndPublish(context.Background(), Event{}, fetcher, upserter, pub, nil, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndPublish: %v", err)
	}

	if resp.TickerCount != 3 {
		t.Errorf("TickerCount = %d, want 3", resp.TickerCount)
	}
	if len(pub.published) != 3 {
		t.Fatalf("published %d messages, want 3", len(pub.published))
	}
}

func TestFetchAndPublish_AllowlistCaseInsensitive(t *testing.T) {
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL", Name: "Apple Inc."},
			{Ticker: "MSFT", Name: "Microsoft"},
		},
	}
	pub := &stubPublisher{}
	upserter := &stubUpserter{}

	// Config csvToSlice uppercases values, so lowercase input becomes uppercase
	allowlist := []string{"AAPL"} // simulates "aapl" after csvToSlice processing
	resp, err := fetchAndPublish(context.Background(), Event{}, fetcher, upserter, pub, allowlist, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndPublish: %v", err)
	}

	if resp.TickerCount != 1 {
		t.Errorf("TickerCount = %d, want 1", resp.TickerCount)
	}
	if pub.published[0].Ticker.Ticker != "AAPL" {
		t.Errorf("ticker = %q, want %q", pub.published[0].Ticker.Ticker, "AAPL")
	}
}

func TestFetchAndPublish_UpsertsOnlyFilteredTickers(t *testing.T) {
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL"},
			{Ticker: "MSFT"},
			{Ticker: "GOOG"},
		},
	}
	pub := &stubPublisher{}
	upserter := &stubUpserter{}

	_, err := fetchAndPublish(context.Background(), Event{}, fetcher, upserter, pub, []string{"AAPL"}, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndPublish: %v", err)
	}

	if len(upserter.received) != 1 {
		t.Fatalf("upserter received %d tickers, want 1", len(upserter.received))
	}
	if upserter.received[0].Ticker != "AAPL" {
		t.Errorf("upserter received %q, want AAPL", upserter.received[0].Ticker)
	}
}

func TestFetchAndPublish_EventTickersOverrideAllowlist(t *testing.T) {
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL"},
			{Ticker: "MSFT"},
			{Ticker: "TSLA"},
		},
	}
	pub := &stubPublisher{}
	upserter := &stubUpserter{}

	event := Event{Tickers: []string{"aapl", " msft "}}
	resp, err := fetchAndPublish(context.Background(), event, fetcher, upserter, pub, []string{"TSLA"}, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndPublish: %v", err)
	}

	if resp.TickerCount != 2 {
		t.Fatalf("TickerCount = %d, want 2", resp.TickerCount)
	}
	symbols := map[string]bool{}
	for _, msg := range pub.published {
		symbols[msg.Ticker.Ticker] = true
	}
	if !symbols["AAPL"] || !symbols["MSFT"] {
		t.Errorf("published symbols = %v, want AAPL and MSFT", symbols)
	}
	if symbols["TSLA"] {
		t.Error("TSLA should not be published; event tickers override the allowlist")
	}
}

func TestFetchAndPublish_EventTickersUnmatched(t *testing.T) {
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL"},
		},
	}
	pub := &stubPublisher{}
	upserter := &stubUpserter{}

	event := Event{Tickers: []string{"AAPL", "ZZZZ"}}
	resp, err := fetchAndPublish(context.Background(), event, fetcher, upserter, pub, nil, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndPublish: %v", err)
	}

	if resp.TickerCount != 1 {
		t.Errorf("TickerCount = %d, want 1", resp.TickerCount)
	}
	if len(resp.UnmatchedTickers) != 1 || resp.UnmatchedTickers[0] != "ZZZZ" {
		t.Errorf("UnmatchedTickers = %v, want [ZZZZ]", resp.UnmatchedTickers)
	}
}

func TestFetchAndPublish_EventTickersBlank(t *testing.T) {
	fetcher := &stubFetcher{
		tickers: []domain.Ticker{
			{Ticker: "AAPL"},
			{Ticker: "MSFT"},
		},
	}
	pub := &stubPublisher{}
	upserter := &stubUpserter{}

	event := Event{Tickers: []string{"", " "}}
	resp, err := fetchAndPublish(context.Background(), event, fetcher, upserter, pub, []string{"AAPL"}, discardLogger)
	if err != nil {
		t.Fatalf("fetchAndPublish: %v", err)
	}

	if resp.TickerCount != 1 {
		t.Errorf("TickerCount = %d, want 1 (env allowlist should apply)", resp.TickerCount)
	}
	if len(resp.UnmatchedTickers) != 0 {
		t.Errorf("UnmatchedTickers = %v, want none (blank event tickers behave as absent)", resp.UnmatchedTickers)
	}
}
