package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/pipeline"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func ptr(f float64) *float64 { return &f }

// stubs

type stubIndicatorFetcher struct {
	tech *domain.TechnicalIndicators
	err  error
}

func (f *stubIndicatorFetcher) FetchAllIndicators(_ context.Context, _ string, _ time.Time) (*domain.TechnicalIndicators, error) {
	return f.tech, f.err
}

type stubPriceReader struct {
	prices []domain.DailyPrice
	err    error
}

func (r *stubPriceReader) GetByTickerAndDateRange(_ context.Context, _ string, _, _ time.Time) ([]domain.DailyPrice, error) {
	return r.prices, r.err
}

type stubTechnicalsWriter struct {
	upserted *domain.TechnicalIndicators
	err      error
}

func (w *stubTechnicalsWriter) Upsert(_ context.Context, tech *domain.TechnicalIndicators) error {
	w.upserted = tech
	return w.err
}

func TestFetchTechnicals_HappyPath(t *testing.T) {
	fetcher := &stubIndicatorFetcher{
		tech: &domain.TechnicalIndicators{
			SMA20: ptr(175.0), SMA50: ptr(170.0), RSI14: ptr(55.0),
			MACDLine: ptr(1.5), MACDSignal: ptr(1.2), MACDHistogram: ptr(0.3),
		},
	}
	// Provide enough OHLCV data for all self-computed indicators
	prices := make([]domain.DailyPrice, 25)
	for i := range prices {
		prices[i] = domain.DailyPrice{
			Close:  100 + float64(i),
			High:   105 + float64(i),
			Low:    95 + float64(i),
			Volume: 1000000,
		}
	}
	reader := &stubPriceReader{prices: prices}
	writer := &stubTechnicalsWriter{}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	resp, err := fetchTechnicals(context.Background(), event, fetcher, reader, writer, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.IndicatorsFetched != 4 { // SMA20, SMA50, RSI14, MACDLine
		t.Errorf("IndicatorsFetched = %d, want 4", resp.IndicatorsFetched)
	}
	if resp.SelfComputed != 3 { // Bollinger, ATR, OBV
		t.Errorf("SelfComputed = %d, want 3", resp.SelfComputed)
	}
	if writer.upserted == nil {
		t.Fatal("expected technicals to be upserted")
	}
	if writer.upserted.TickerID != "uuid-123" {
		t.Errorf("TickerID = %q, want uuid-123", writer.upserted.TickerID)
	}
	if writer.upserted.BollingerUpper == nil {
		t.Error("expected Bollinger to be computed")
	}
	if writer.upserted.ATR14 == nil {
		t.Error("expected ATR to be computed")
	}
	if writer.upserted.OBV == nil {
		t.Error("expected OBV to be computed")
	}
}

func TestFetchTechnicals_NoOHLCVHistory(t *testing.T) {
	fetcher := &stubIndicatorFetcher{
		tech: &domain.TechnicalIndicators{SMA20: ptr(175.0)},
	}
	reader := &stubPriceReader{prices: nil}
	writer := &stubTechnicalsWriter{}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	resp, err := fetchTechnicals(context.Background(), event, fetcher, reader, writer, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.SelfComputed != 0 {
		t.Errorf("SelfComputed = %d, want 0 (no OHLCV)", resp.SelfComputed)
	}
}

func TestFetchTechnicals_MissingTickerID(t *testing.T) {
	event := pipeline.TickerEvent{Ticker: "AAPL", Date: "2026-04-08"}
	_, err := fetchTechnicals(context.Background(), event, nil, nil, nil, discardLogger)
	if err == nil {
		t.Fatal("expected error for missing ticker_id")
	}
}

func TestFetchTechnicals_FetchError(t *testing.T) {
	fetcher := &stubIndicatorFetcher{err: fmt.Errorf("api error")}
	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}

	_, err := fetchTechnicals(context.Background(), event, fetcher, nil, nil, discardLogger)
	if err == nil {
		t.Fatal("expected error for fetch failure")
	}
}

func TestFetchTechnicals_UpsertError(t *testing.T) {
	fetcher := &stubIndicatorFetcher{tech: &domain.TechnicalIndicators{}}
	reader := &stubPriceReader{}
	writer := &stubTechnicalsWriter{err: fmt.Errorf("db error")}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	_, err := fetchTechnicals(context.Background(), event, fetcher, reader, writer, discardLogger)
	if err == nil {
		t.Fatal("expected error for upsert failure")
	}
}

func TestHandleRequest_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("MASSIVE_API_KEY", "test-key")

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	_, err := handleRequest(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
}

func TestHandleRequest_MissingAPIKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("MASSIVE_API_KEY", "")

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	_, err := handleRequest(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for missing MASSIVE_API_KEY")
	}
}
