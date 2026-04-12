package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/pipeline"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// stubs

type stubDetailsFetcher struct {
	fund *domain.TickerFundamentals
	err  error
}

func (f *stubDetailsFetcher) FetchTickerDetails(_ context.Context, _ string) (*domain.TickerFundamentals, error) {
	return f.fund, f.err
}

type stubDividendFetcher struct {
	dividends []domain.TickerDividend
	err       error
}

func (f *stubDividendFetcher) FetchDividends(_ context.Context, _ string) ([]domain.TickerDividend, error) {
	return f.dividends, f.err
}

type stubFundamentalsWriter struct {
	upserted *domain.TickerFundamentals
	err      error
}

func (w *stubFundamentalsWriter) Upsert(_ context.Context, f *domain.TickerFundamentals) error {
	w.upserted = f
	return w.err
}

type stubDividendWriter struct {
	upserted []domain.TickerDividend
	err      error
}

func (w *stubDividendWriter) UpsertBatch(_ context.Context, divs []domain.TickerDividend) error {
	w.upserted = divs
	return w.err
}

type stubPriceReader struct {
	price *domain.DailyPrice
	err   error
}

func (r *stubPriceReader) GetLatest(_ context.Context, _ string) (*domain.DailyPrice, error) {
	return r.price, r.err
}

type stubSummaryWriter struct {
	upserted *domain.TickerDividendSummary
	err      error
}

func (w *stubSummaryWriter) Upsert(_ context.Context, s *domain.TickerDividendSummary) error {
	w.upserted = s
	return w.err
}

func TestFetchFundamentals_HappyPath(t *testing.T) {
	details := &stubDetailsFetcher{
		fund: &domain.TickerFundamentals{
			MarketCap: 2500000000000,
			SICCode:   "3571",
		},
	}
	divFetcher := &stubDividendFetcher{
		dividends: []domain.TickerDividend{
			{CashAmount: 0.24, ExDividendDate: "2026-03-15", Frequency: 4, DistributionType: "CD"},
			{CashAmount: 0.24, ExDividendDate: "2025-12-15", Frequency: 4, DistributionType: "CD"},
		},
	}
	fundWriter := &stubFundamentalsWriter{}
	divWriter := &stubDividendWriter{}
	prices := &stubPriceReader{price: &domain.DailyPrice{Close: 175.0}}
	sumWriter := &stubSummaryWriter{}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	resp, err := fetchFundamentals(context.Background(), event, details, divFetcher, fundWriter, divWriter, prices, sumWriter, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.HasFundamentals {
		t.Error("expected HasFundamentals = true")
	}
	if resp.DividendCount != 2 {
		t.Errorf("DividendCount = %d, want 2", resp.DividendCount)
	}
	if fundWriter.upserted == nil {
		t.Fatal("expected fundamentals to be upserted")
	}
	if fundWriter.upserted.TickerID != "uuid-123" {
		t.Errorf("TickerID = %q, want uuid-123", fundWriter.upserted.TickerID)
	}
	if divWriter.upserted == nil || len(divWriter.upserted) != 2 {
		t.Fatal("expected 2 dividends to be upserted")
	}
	if sumWriter.upserted == nil {
		t.Fatal("expected dividend summary to be upserted")
	}
}

func TestFetchFundamentals_MissingTickerID(t *testing.T) {
	event := pipeline.TickerEvent{Ticker: "AAPL", Date: "2026-04-08"}
	_, err := fetchFundamentals(context.Background(), event, nil, nil, nil, nil, nil, nil, discardLogger)
	if err == nil {
		t.Fatal("expected error for missing ticker_id")
	}
}

func TestFetchFundamentals_DetailsError_ContinuesWithDividends(t *testing.T) {
	details := &stubDetailsFetcher{err: fmt.Errorf("api error")}
	divFetcher := &stubDividendFetcher{
		dividends: []domain.TickerDividend{
			{CashAmount: 0.24, ExDividendDate: "2026-03-15", Frequency: 4},
		},
	}
	fundWriter := &stubFundamentalsWriter{}
	divWriter := &stubDividendWriter{}
	prices := &stubPriceReader{price: &domain.DailyPrice{Close: 175.0}}
	sumWriter := &stubSummaryWriter{}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	resp, err := fetchFundamentals(context.Background(), event, details, divFetcher, fundWriter, divWriter, prices, sumWriter, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.HasFundamentals {
		t.Error("expected HasFundamentals = false when details fail")
	}
	if resp.DividendCount != 1 {
		t.Errorf("DividendCount = %d, want 1", resp.DividendCount)
	}
}

func TestFetchFundamentals_DividendFetchError(t *testing.T) {
	details := &stubDetailsFetcher{
		fund: &domain.TickerFundamentals{MarketCap: 100},
	}
	divFetcher := &stubDividendFetcher{err: fmt.Errorf("api error")}
	fundWriter := &stubFundamentalsWriter{}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	resp, err := fetchFundamentals(context.Background(), event, details, divFetcher, fundWriter, nil, nil, nil, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.HasFundamentals {
		t.Error("expected HasFundamentals = true")
	}
	if resp.DividendCount != 0 {
		t.Errorf("DividendCount = %d, want 0", resp.DividendCount)
	}
}

func TestFetchFundamentals_NoDividends(t *testing.T) {
	details := &stubDetailsFetcher{fund: &domain.TickerFundamentals{}}
	divFetcher := &stubDividendFetcher{dividends: nil}
	fundWriter := &stubFundamentalsWriter{}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	resp, err := fetchFundamentals(context.Background(), event, details, divFetcher, fundWriter, nil, nil, nil, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.DividendCount != 0 {
		t.Errorf("DividendCount = %d, want 0", resp.DividendCount)
	}
}

func TestFetchFundamentals_UpsertFundamentalsError(t *testing.T) {
	details := &stubDetailsFetcher{fund: &domain.TickerFundamentals{}}
	fundWriter := &stubFundamentalsWriter{err: fmt.Errorf("db error")}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	_, err := fetchFundamentals(context.Background(), event, details, nil, fundWriter, nil, nil, nil, discardLogger)
	if err == nil {
		t.Fatal("expected error for upsert failure")
	}
}

func TestFetchFundamentals_UpsertDividendsError(t *testing.T) {
	details := &stubDetailsFetcher{fund: &domain.TickerFundamentals{}}
	divFetcher := &stubDividendFetcher{
		dividends: []domain.TickerDividend{{CashAmount: 0.24, ExDividendDate: "2026-03-15"}},
	}
	fundWriter := &stubFundamentalsWriter{}
	divWriter := &stubDividendWriter{err: fmt.Errorf("db error")}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	_, err := fetchFundamentals(context.Background(), event, details, divFetcher, fundWriter, divWriter, nil, nil, discardLogger)
	if err == nil {
		t.Fatal("expected error for dividend upsert failure")
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
