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

type stubFundamentalsReader struct {
	fund *domain.TickerFundamentals
	err  error
}

func (r *stubFundamentalsReader) GetLatest(_ context.Context, _ string) (*domain.TickerFundamentals, error) {
	return r.fund, r.err
}

type stubTechnicalsReader struct {
	tech *domain.TechnicalIndicators
	err  error
}

func (r *stubTechnicalsReader) GetLatest(_ context.Context, _ string) (*domain.TechnicalIndicators, error) {
	return r.tech, r.err
}

type stubPriceReader struct {
	price *domain.DailyPrice
	err   error
}

func (r *stubPriceReader) GetLatest(_ context.Context, _ string) (*domain.DailyPrice, error) {
	return r.price, r.err
}

type stubSectorUpdater struct {
	tickerID string
	sector   string
	err      error
}

func (u *stubSectorUpdater) UpdateSector(_ context.Context, tickerID string, sector string) error {
	u.tickerID = tickerID
	u.sector = sector
	return u.err
}

type stubStatusUpdater struct {
	tickerID string
	statuses map[string]string
	err      error
}

func (u *stubStatusUpdater) UpdateIndicatorStatuses(_ context.Context, tickerID string, _ time.Time, statuses map[string]string) error {
	u.tickerID = tickerID
	u.statuses = statuses
	return u.err
}

// stubStageTracker is a no-op tracker for tests.
type stubStageTracker struct{}

func (s *stubStageTracker) MarkRunning(_ context.Context, _, _, _ string) (string, error) {
	return "stage-id", nil
}

func (s *stubStageTracker) MarkCompleted(_ context.Context, _, _, _ string) error {
	return nil
}

func (s *stubStageTracker) MarkFailed(_ context.Context, _, _, _, _ string) error {
	return nil
}

// failingStageTracker always returns errors (verifies tracking failures don't abort work).
type failingStageTracker struct{}

func (s *failingStageTracker) MarkRunning(_ context.Context, _, _, _ string) (string, error) {
	return "", fmt.Errorf("tracking unavailable")
}

func (s *failingStageTracker) MarkCompleted(_ context.Context, _, _, _ string) error {
	return fmt.Errorf("tracking unavailable")
}

func (s *failingStageTracker) MarkFailed(_ context.Context, _, _, _, _ string) error {
	return fmt.Errorf("tracking unavailable")
}

func TestEnrichTicker_HappyPath(t *testing.T) {
	fundReader := &stubFundamentalsReader{
		fund: &domain.TickerFundamentals{SICCode: "3571"}, // Electronic Computers → Technology
	}
	techReader := &stubTechnicalsReader{
		tech: &domain.TechnicalIndicators{
			RSI14:      ptr(55.0),
			SMA20:      ptr(170.0),
			MACDLine:   ptr(1.5),
			MACDSignal: ptr(1.2),
		},
	}
	priceReader := &stubPriceReader{price: &domain.DailyPrice{Close: 175.0}}
	sectorUpd := &stubSectorUpdater{}
	statusUpd := &stubStatusUpdater{}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	resp, err := enrichTicker(context.Background(), event, fundReader, techReader, priceReader, sectorUpd, statusUpd, &stubStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Sector != "Technology" {
		t.Errorf("Sector = %q, want Technology", resp.Sector)
	}
	if resp.IndicatorsClassified == 0 {
		t.Error("expected indicators to be classified")
	}
	if sectorUpd.tickerID != "uuid-123" {
		t.Errorf("sector update tickerID = %q, want uuid-123", sectorUpd.tickerID)
	}
	if sectorUpd.sector != "Technology" {
		t.Errorf("sector = %q, want Technology", sectorUpd.sector)
	}
	if statusUpd.tickerID != "uuid-123" {
		t.Errorf("status update tickerID = %q, want uuid-123", statusUpd.tickerID)
	}
	if _, ok := statusUpd.statuses["rsi_14"]; !ok {
		t.Error("expected rsi_14 in indicator statuses")
	}
}

func TestEnrichTicker_MissingTickerID(t *testing.T) {
	event := pipeline.TickerEvent{Ticker: "AAPL", Date: "2026-04-08"}
	_, err := enrichTicker(context.Background(), event, nil, nil, nil, nil, nil, &stubStageTracker{}, discardLogger)
	if err == nil {
		t.Fatal("expected error for missing ticker_id")
	}
}

func TestEnrichTicker_NoFundamentals(t *testing.T) {
	fundReader := &stubFundamentalsReader{err: fmt.Errorf("not found")}
	techReader := &stubTechnicalsReader{
		tech: &domain.TechnicalIndicators{RSI14: ptr(55.0)},
	}
	priceReader := &stubPriceReader{price: &domain.DailyPrice{Close: 175.0}}
	statusUpd := &stubStatusUpdater{}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	resp, err := enrichTicker(context.Background(), event, fundReader, techReader, priceReader, nil, statusUpd, &stubStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Sector != "" {
		t.Errorf("Sector = %q, want empty", resp.Sector)
	}
	if resp.IndicatorsClassified == 0 {
		t.Error("expected indicators to be classified even without fundamentals")
	}
}

func TestEnrichTicker_NoTechnicals(t *testing.T) {
	fundReader := &stubFundamentalsReader{
		fund: &domain.TickerFundamentals{SICCode: "3571"},
	}
	techReader := &stubTechnicalsReader{err: fmt.Errorf("not found")}
	sectorUpd := &stubSectorUpdater{}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	resp, err := enrichTicker(context.Background(), event, fundReader, techReader, nil, sectorUpd, nil, &stubStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Sector != "Technology" {
		t.Errorf("Sector = %q, want Technology", resp.Sector)
	}
	if resp.IndicatorsClassified != 0 {
		t.Errorf("IndicatorsClassified = %d, want 0", resp.IndicatorsClassified)
	}
}

func TestEnrichTicker_NoDailyPrice(t *testing.T) {
	fundReader := &stubFundamentalsReader{fund: &domain.TickerFundamentals{}}
	techReader := &stubTechnicalsReader{
		tech: &domain.TechnicalIndicators{RSI14: ptr(55.0)},
	}
	priceReader := &stubPriceReader{err: fmt.Errorf("not found")}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	resp, err := enrichTicker(context.Background(), event, fundReader, techReader, priceReader, nil, nil, &stubStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.IndicatorsClassified != 0 {
		t.Errorf("IndicatorsClassified = %d, want 0", resp.IndicatorsClassified)
	}
}

func TestEnrichTicker_SectorUpdateError(t *testing.T) {
	fundReader := &stubFundamentalsReader{
		fund: &domain.TickerFundamentals{SICCode: "3571"},
	}
	sectorUpd := &stubSectorUpdater{err: fmt.Errorf("db error")}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	_, err := enrichTicker(context.Background(), event, fundReader, nil, nil, sectorUpd, nil, &stubStageTracker{}, discardLogger)
	if err == nil {
		t.Fatal("expected error for sector update failure")
	}
}

func TestEnrichTicker_StatusUpdateError(t *testing.T) {
	fundReader := &stubFundamentalsReader{fund: &domain.TickerFundamentals{}}
	techReader := &stubTechnicalsReader{
		tech: &domain.TechnicalIndicators{RSI14: ptr(55.0)},
	}
	priceReader := &stubPriceReader{price: &domain.DailyPrice{Close: 175.0}}
	statusUpd := &stubStatusUpdater{err: fmt.Errorf("db error")}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08"}
	_, err := enrichTicker(context.Background(), event, fundReader, techReader, priceReader, nil, statusUpd, &stubStageTracker{}, discardLogger)
	if err == nil {
		t.Fatal("expected error for status update failure")
	}
}

func TestEnrichTicker_TrackingFailure_DoesNotAbort(t *testing.T) {
	fundReader := &stubFundamentalsReader{fund: &domain.TickerFundamentals{}}
	techReader := &stubTechnicalsReader{err: fmt.Errorf("not found")}

	event := pipeline.TickerEvent{Ticker: "AAPL", TickerID: "uuid-123", Date: "2026-04-08", RunID: "run-123"}
	resp, err := enrichTicker(context.Background(), event, fundReader, techReader, nil, nil, nil, &failingStageTracker{}, discardLogger)
	if err != nil {
		t.Fatalf("tracking failure should not abort work: %v", err)
	}
	if resp.Ticker != "AAPL" {
		t.Errorf("Ticker = %q, want AAPL", resp.Ticker)
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
