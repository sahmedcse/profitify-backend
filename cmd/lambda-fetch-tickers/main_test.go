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

// stubUpserter records its input and assigns deterministic IDs so tests can
// verify that TickerMessage.ID is the ID the upserter assigned. It can also
// be made to fail, or to leave one ticker's ID empty (defensive-guard tests).
type stubUpserter struct {
	received  []domain.Ticker
	err       error
	skipIDFor string // symbol whose ID is left empty; "" disables
}

func (u *stubUpserter) UpsertBatch(_ context.Context, tickers []domain.Ticker) error {
	u.received = append(u.received, tickers...)
	if u.err != nil {
		return u.err
	}
	for i := range tickers {
		if tickers[i].Ticker == u.skipIDFor {
			continue
		}
		tickers[i].ID = "uuid-" + tickers[i].Ticker
	}
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

func TestHandleRequest_MissingSQSQueueURL(t *testing.T) {
	t.Setenv("MASSIVE_API_KEY", "test-key")
	t.Setenv("SQS_QUEUE_URL", "")
	t.Setenv("DATABASE_URL", "postgres://localhost/db")

	_, err := handleRequest(context.Background(), Event{})
	if err == nil {
		t.Fatal("expected error for missing SQS_QUEUE_URL")
	}
}

func TestHandleRequest_MissingAPIKey(t *testing.T) {
	t.Setenv("MASSIVE_API_KEY", "")
	t.Setenv("SQS_QUEUE_URL", "https://sqs.us-east-1.amazonaws.com/123/test")
	t.Setenv("DATABASE_URL", "postgres://localhost/db")

	_, err := handleRequest(context.Background(), Event{})
	if err == nil {
		t.Fatal("expected error for missing MASSIVE_API_KEY")
	}
}

func TestHandleRequest_MissingDatabaseURL(t *testing.T) {
	t.Setenv("MASSIVE_API_KEY", "test-key")
	t.Setenv("SQS_QUEUE_URL", "https://sqs.us-east-1.amazonaws.com/123/test")
	t.Setenv("DATABASE_URL", "")

	_, err := handleRequest(context.Background(), Event{})
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
}

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

func TestEffectiveTickerLimit(t *testing.T) {
	tests := []struct {
		name         string
		cfgLimit     int
		eventTickers []string
		want         int
	}{
		{name: "no event tickers uses configured limit", cfgLimit: 110, eventTickers: nil, want: 110},
		{name: "event tickers present disables limit", cfgLimit: 110, eventTickers: []string{"AAPL"}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := effectiveTickerLimit(tt.cfgLimit, tt.eventTickers); got != tt.want {
				t.Errorf("effectiveTickerLimit(%d, %v) = %d, want %d", tt.cfgLimit, tt.eventTickers, got, tt.want)
			}
		})
	}
}
