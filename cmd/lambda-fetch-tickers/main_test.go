package main

import (
	"context"
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
