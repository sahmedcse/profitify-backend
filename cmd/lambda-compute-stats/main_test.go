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

type stubPriceReader struct {
	prices []domain.DailyPrice
	err    error
}

func (r *stubPriceReader) GetByTickerAndDateRange(_ context.Context, _ string, _, _ time.Time) ([]domain.DailyPrice, error) {
	return r.prices, r.err
}

type stubTechnicalsReader struct {
	tech *domain.TechnicalIndicators
	err  error
}

func (r *stubTechnicalsReader) GetLatest(_ context.Context, _ string) (*domain.TechnicalIndicators, error) {
	return r.tech, r.err
}

type stubStatsWriter struct {
	upserted *domain.TickerStats
	err      error
}

func (w *stubStatsWriter) Upsert(_ context.Context, s *domain.TickerStats) error {
	w.upserted = s
	return w.err
}

func makePrices(n int) []domain.DailyPrice {
	prices := make([]domain.DailyPrice, n)
	for i := range prices {
		prices[i] = domain.DailyPrice{
			Open:   100 + float64(i),
			High:   105 + float64(i),
			Low:    95 + float64(i),
			Close:  100 + float64(i),
			Volume: 1000000,
		}
	}
	return prices
}

func TestComputeStats_HappyPath(t *testing.T) {
	prices := makePrices(100) // enough for 90-day rolling
	techReader := &stubTechnicalsReader{
		tech: &domain.TechnicalIndicators{
			RSI14:      ptr(55.0),
			SMA20:      ptr(170.0),
			MACDLine:   ptr(1.5),
			MACDSignal: ptr(1.2),
		},
	}
	writer := &stubStatsWriter{}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	resp, err := computeStats(context.Background(), event, &stubPriceReader{prices: prices}, techReader, writer, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.SignalLabel == "" {
		t.Error("expected non-empty SignalLabel")
	}
	if writer.upserted == nil {
		t.Fatal("expected stats to be upserted")
	}
	if writer.upserted.TickerID != "uuid-123" {
		t.Errorf("TickerID = %q, want uuid-123", writer.upserted.TickerID)
	}
	if writer.upserted.PriceReturn30d == nil {
		t.Error("expected PriceReturn30d to be set")
	}
	if writer.upserted.PriceReturn90d == nil {
		t.Error("expected PriceReturn90d to be set")
	}
	if writer.upserted.High52w == nil {
		t.Error("expected High52w to be set")
	}
	if writer.upserted.PivotLevels == nil {
		t.Error("expected PivotLevels to be set")
	}
}

func TestComputeStats_MissingTickerID(t *testing.T) {
	event := pipeline.TickerEvent{Ticker: "AAPL", Date: "2026-04-08"}
	_, err := computeStats(context.Background(), event, nil, nil, nil, discardLogger)
	if err == nil {
		t.Fatal("expected error for missing ticker_id")
	}
}

func TestComputeStats_InsufficientData(t *testing.T) {
	prices := makePrices(1) // only 1 bar
	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	_, err := computeStats(context.Background(), event, &stubPriceReader{prices: prices}, nil, nil, discardLogger)
	if err == nil {
		t.Fatal("expected error for insufficient data")
	}
}

func TestComputeStats_PriceReadError(t *testing.T) {
	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	_, err := computeStats(context.Background(), event, &stubPriceReader{err: fmt.Errorf("db error")}, nil, nil, discardLogger)
	if err == nil {
		t.Fatal("expected error for price read failure")
	}
}

func TestComputeStats_NoTechnicals(t *testing.T) {
	prices := makePrices(10)
	techReader := &stubTechnicalsReader{err: fmt.Errorf("not found")}
	writer := &stubStatsWriter{}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	resp, err := computeStats(context.Background(), event, &stubPriceReader{prices: prices}, techReader, writer, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still compute stats, just without technical signal input
	if resp.SignalLabel == "" {
		t.Error("expected non-empty SignalLabel even without technicals")
	}
}

func TestComputeStats_UpsertError(t *testing.T) {
	prices := makePrices(5)
	techReader := &stubTechnicalsReader{tech: &domain.TechnicalIndicators{}}
	writer := &stubStatsWriter{err: fmt.Errorf("db error")}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	_, err := computeStats(context.Background(), event, &stubPriceReader{prices: prices}, techReader, writer, discardLogger)
	if err == nil {
		t.Fatal("expected error for upsert failure")
	}
}

func TestComputeStats_MinimalData(t *testing.T) {
	prices := makePrices(2) // exactly 2 bars (minimum)
	techReader := &stubTechnicalsReader{tech: &domain.TechnicalIndicators{}}
	writer := &stubStatsWriter{}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	resp, err := computeStats(context.Background(), event, &stubPriceReader{prices: prices}, techReader, writer, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if writer.upserted.PriceReturn7d != nil {
		t.Error("expected nil PriceReturn7d for 2 bars (< 7)")
	}
	if resp.Ticker != "AAPL" {
		t.Errorf("Ticker = %q, want AAPL", resp.Ticker)
	}
}

func TestHandleRequest_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	_, err := handleRequest(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
}
