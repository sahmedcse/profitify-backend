package stats_test

import (
	"math"
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/stats"
)

func TestComputePivots_Empty(t *testing.T) {
	got := stats.ComputePivots(nil)
	if got != (stats.PivotLevels{}) {
		t.Errorf("expected zero value, got %+v", got)
	}
}

func TestComputePivots_ClassicFormula(t *testing.T) {
	// Single bar: H=110, L=90, C=100
	// P = (110 + 90 + 100) / 3 = 100
	// R1 = 2*100 - 90 = 110
	// S1 = 2*100 - 110 = 90
	// R2 = 100 + 20 = 120
	// S2 = 100 - 20 = 80
	// R3 = 110 + 2*(100 - 90) = 130
	// S3 = 90 - 2*(110 - 100) = 70
	bars := []domain.DailyPrice{
		{High: 110, Low: 90, Close: 100},
	}
	got := stats.ComputePivots(bars)

	tests := []struct {
		name string
		got  float64
		want float64
	}{
		{"R3", got.R3.Price, 130},
		{"R2", got.R2.Price, 120},
		{"R1", got.R1.Price, 110},
		{"S1", got.S1.Price, 90},
		{"S2", got.S2.Price, 80},
		{"S3", got.S3.Price, 70},
	}
	for _, tt := range tests {
		if math.Abs(tt.got-tt.want) > 1e-9 {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestComputePivots_UsesLastBarForPivots(t *testing.T) {
	// Last bar H=100, L=80, C=90 → P=90, R1=2*90-80=100
	// Earlier bars should not affect the pivot *values*.
	bars := []domain.DailyPrice{
		{High: 200, Low: 150, Close: 175},
		{High: 100, Low: 80, Close: 90},
	}
	got := stats.ComputePivots(bars)
	if math.Abs(got.R1.Price-100) > 1e-9 {
		t.Errorf("R1 = %v, want 100", got.R1.Price)
	}
}

func TestComputePivots_StrengthFromTouches(t *testing.T) {
	// Last bar H=110, L=90, C=100 → R1 = 110.
	// Add three historical bars that close within 0.5% of 110.
	bars := []domain.DailyPrice{
		{High: 111, Low: 108, Close: 110.1}, // touch R1
		{High: 112, Low: 109, Close: 109.9}, // touch R1
		{High: 113, Low: 108, Close: 110.2}, // touch R1
		{High: 110, Low: 90, Close: 100},    // last bar
	}
	got := stats.ComputePivots(bars)
	if got.R1.Strength != stats.StrengthStrong {
		t.Errorf("R1 strength = %q, want strong (3 touches)", got.R1.Strength)
	}
}

func TestComputePivots_StrengthWeak(t *testing.T) {
	// Only last bar, far from pivots → weak
	bars := []domain.DailyPrice{
		{High: 110, Low: 90, Close: 100},
	}
	got := stats.ComputePivots(bars)
	// Last bar's close=100 is exactly at the pivot (P=100) but not at R1 (110)
	if got.R1.Strength != stats.StrengthWeak {
		t.Errorf("R1 strength with no touches = %q, want weak", got.R1.Strength)
	}
}

func TestComputePivots_StrengthModerate(t *testing.T) {
	// Exactly one touch → moderate
	bars := []domain.DailyPrice{
		{High: 112, Low: 108, Close: 110.1}, // touch R1=110
		{High: 110, Low: 90, Close: 100},
	}
	got := stats.ComputePivots(bars)
	if got.R1.Strength != stats.StrengthModerate {
		t.Errorf("R1 strength with 1 touch = %q, want moderate", got.R1.Strength)
	}
}
