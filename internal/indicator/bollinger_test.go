package indicator_test

import (
	"math"
	"testing"

	"github.com/profitify/profitify-backend/internal/indicator"
)

func TestComputeBollinger_Basic(t *testing.T) {
	// 5-period Bollinger with multiplier 2
	// All same price → std dev = 0 → bands collapse to the price
	closes := []float64{100, 100, 100, 100, 100}
	upper, mid, lower, err := indicator.ComputeBollinger(closes, 5, 2.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mid != 100 {
		t.Errorf("mid = %v, want 100", mid)
	}
	if upper != 100 {
		t.Errorf("upper = %v, want 100 (no deviation)", upper)
	}
	if lower != 100 {
		t.Errorf("lower = %v, want 100 (no deviation)", lower)
	}
}

func TestComputeBollinger_WithVariance(t *testing.T) {
	// 3-period: [98, 100, 102]
	// Mean = 100, StdDev = sqrt((4+0+4)/3) = sqrt(8/3) ≈ 1.6330
	// With multiplier 2: upper ≈ 103.27, lower ≈ 96.73
	closes := []float64{98, 100, 102}
	upper, mid, lower, err := indicator.ComputeBollinger(closes, 3, 2.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(mid-100) > 0.001 {
		t.Errorf("mid = %v, want 100", mid)
	}
	expectedStdDev := math.Sqrt(8.0 / 3.0)
	expectedUpper := 100 + 2*expectedStdDev
	expectedLower := 100 - 2*expectedStdDev
	if math.Abs(upper-expectedUpper) > 0.001 {
		t.Errorf("upper = %v, want %v", upper, expectedUpper)
	}
	if math.Abs(lower-expectedLower) > 0.001 {
		t.Errorf("lower = %v, want %v", lower, expectedLower)
	}
}

func TestComputeBollinger_UsesLastNPeriod(t *testing.T) {
	// Extra data at front should be ignored
	closes := []float64{50, 60, 70, 100, 100, 100}
	_, mid, _, err := indicator.ComputeBollinger(closes, 3, 2.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mid != 100 {
		t.Errorf("mid = %v, want 100 (should use last 3)", mid)
	}
}

func TestComputeBollinger_InsufficientData(t *testing.T) {
	closes := []float64{100, 101}
	_, _, _, err := indicator.ComputeBollinger(closes, 5, 2.0)
	if err == nil {
		t.Fatal("expected error for insufficient data")
	}
}

func TestComputeBollinger_ZeroPeriod(t *testing.T) {
	_, _, _, err := indicator.ComputeBollinger([]float64{100}, 0, 2.0)
	if err == nil {
		t.Fatal("expected error for zero period")
	}
}
