package signal_test

import (
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/signal"
)

func ptr(f float64) *float64 { return &f }

func TestClassifyAll_Empty(t *testing.T) {
	got := signal.ClassifyAll(nil, 100)
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestClassifyAll_PartialData(t *testing.T) {
	tech := &domain.TechnicalIndicators{
		RSI14:    ptr(75),
		MACDLine: ptr(1.5),
		// Missing MACDSignal — MACD should be skipped
		SMA20: ptr(100),
	}
	got := signal.ClassifyAll(tech, 105)

	if got[signal.IndicatorRSI] != signal.StatusBullish {
		t.Errorf("RSI = %v, want bullish", got[signal.IndicatorRSI])
	}
	if _, exists := got[signal.IndicatorMACD]; exists {
		t.Error("MACD should not be classified when signal is nil")
	}
	if got[signal.IndicatorSMA20] != signal.StatusBullish {
		t.Errorf("SMA20 = %v, want bullish", got[signal.IndicatorSMA20])
	}
}

func TestClassifyAll_Full(t *testing.T) {
	tech := &domain.TechnicalIndicators{
		RSI14:           ptr(50),
		MACDLine:        ptr(1.0),
		MACDSignal:      ptr(0.5),
		SMA20:           ptr(95),
		SMA50:           ptr(90),
		EMA12:           ptr(98),
		BollingerUpper:  ptr(110),
		BollingerMiddle: ptr(100),
		BollingerLower:  ptr(90),
	}
	got := signal.ClassifyAll(tech, 100)
	if len(got) != 6 {
		t.Errorf("expected 6 statuses, got %d: %v", len(got), got)
	}
}

func TestAggregate_StrongBuy(t *testing.T) {
	tech := &domain.TechnicalIndicators{
		RSI14:      ptr(72),  // bullish
		MACDLine:   ptr(2.0), // bullish
		MACDSignal: ptr(1.0),
		SMA20:      ptr(95), // bullish
		SMA50:      ptr(90), // bullish
		EMA12:      ptr(98), // bullish
	}
	stats := &domain.TickerStats{PriceReturn30d: ptr(8.0)}
	label, strength := signal.Aggregate(tech, stats, 100)
	if strength < 80 {
		t.Errorf("strength = %d, want >=80", strength)
	}
	if label != "Strong Buy" {
		t.Errorf("label = %q, want Strong Buy", label)
	}
}

func TestAggregate_StrongSell(t *testing.T) {
	tech := &domain.TechnicalIndicators{
		RSI14:      ptr(20),
		MACDLine:   ptr(0.5),
		MACDSignal: ptr(1.5),
		SMA20:      ptr(110),
		SMA50:      ptr(115),
		EMA12:      ptr(108),
	}
	stats := &domain.TickerStats{PriceReturn30d: ptr(-10.0)}
	label, strength := signal.Aggregate(tech, stats, 100)
	if strength >= 20 {
		t.Errorf("strength = %d, want <20", strength)
	}
	if label != "Strong Sell" {
		t.Errorf("label = %q, want Strong Sell", label)
	}
}

func TestAggregate_Neutral(t *testing.T) {
	tech := &domain.TechnicalIndicators{
		RSI14:      ptr(50),
		MACDLine:   ptr(1.0),
		MACDSignal: ptr(1.0),
		SMA20:      ptr(100),
	}
	label, strength := signal.Aggregate(tech, nil, 100)
	if strength < 40 || strength >= 60 {
		t.Errorf("strength = %d, want 40-59 for neutral", strength)
	}
	if label != "Neutral" {
		t.Errorf("label = %q, want Neutral", label)
	}
}

func TestAggregate_NoData(t *testing.T) {
	label, strength := signal.Aggregate(nil, nil, 0)
	if strength != 50 {
		t.Errorf("strength = %d, want 50 for empty input", strength)
	}
	if label != "Neutral" {
		t.Errorf("label = %q, want Neutral", label)
	}
}

func TestAggregate_NudgeIsClamped(t *testing.T) {
	tech := &domain.TechnicalIndicators{
		RSI14:      ptr(72),
		MACDLine:   ptr(2.0),
		MACDSignal: ptr(1.0),
		SMA20:      ptr(95),
	}
	stats := &domain.TickerStats{PriceReturn30d: ptr(500.0)}
	_, strength := signal.Aggregate(tech, stats, 100)
	if strength > 100 {
		t.Errorf("strength = %d, want <=100", strength)
	}
}
