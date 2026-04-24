package indicator_test

import (
	"math"
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/indicator"
)

func TestComputeATR_Basic(t *testing.T) {
	// 3 bars → period=2: two true range values
	prices := []domain.DailyPrice{
		{High: 110, Low: 90, Close: 100},  // bar 0 (prev close reference)
		{High: 115, Low: 95, Close: 105},  // TR = max(20, |115-100|=15, |95-100|=5) = 20
		{High: 120, Low: 100, Close: 110}, // TR = max(20, |120-105|=15, |100-105|=5) = 20
	}
	atr, err := indicator.ComputeATR(prices, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(atr-20) > 0.001 {
		t.Errorf("ATR = %v, want 20", atr)
	}
}

func TestComputeATR_GapUp(t *testing.T) {
	// Gap up: prev close=100, next bar high=120, low=115
	// TR = max(5, |120-100|=20, |115-100|=15) = 20
	prices := []domain.DailyPrice{
		{High: 105, Low: 95, Close: 100},
		{High: 120, Low: 115, Close: 118},
	}
	atr, err := indicator.ComputeATR(prices, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(atr-20) > 0.001 {
		t.Errorf("ATR = %v, want 20", atr)
	}
}

func TestComputeATR_InsufficientData(t *testing.T) {
	// period=1 needs 2 bars (period+1); only have 1
	prices := []domain.DailyPrice{
		{High: 110, Low: 90, Close: 100},
	}
	_, err := indicator.ComputeATR(prices, 1)
	if err == nil {
		t.Fatal("expected error for insufficient data (need period+1 bars)")
	}
}

func TestComputeATR_ExactMinimumBars(t *testing.T) {
	// period=1 needs exactly 2 bars
	prices := []domain.DailyPrice{
		{High: 110, Low: 90, Close: 100},
		{High: 115, Low: 95, Close: 105},
	}
	_, err := indicator.ComputeATR(prices, 1)
	if err != nil {
		t.Fatalf("unexpected error with exact minimum bars: %v", err)
	}
}

func TestComputeATR_ZeroPeriod(t *testing.T) {
	_, err := indicator.ComputeATR(nil, 0)
	if err == nil {
		t.Fatal("expected error for zero period")
	}
}

func TestComputeATR_UsesLastBars(t *testing.T) {
	// Extra data at front should be ignored when period+1 < len
	prices := []domain.DailyPrice{
		{High: 200, Low: 150, Close: 175}, // ignored
		{High: 110, Low: 90, Close: 100},  // prev close reference
		{High: 115, Low: 95, Close: 105},  // TR = 20
	}
	atr, err := indicator.ComputeATR(prices, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(atr-20) > 0.001 {
		t.Errorf("ATR = %v, want 20", atr)
	}
}
