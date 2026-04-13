package stats

import (
	"math"
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
)

func TestCompute52Week_Basic(t *testing.T) {
	prices := []domain.DailyPrice{
		{High: 110, Low: 95},
		{High: 120, Low: 100}, // high here
		{High: 115, Low: 90},  // low here
		{High: 112, Low: 105},
	}

	result := Compute52Week(prices, 110)

	if result.High52w != 120 {
		t.Errorf("High52w = %.2f, want 120", result.High52w)
	}
	if result.Low52w != 90 {
		t.Errorf("Low52w = %.2f, want 90", result.Low52w)
	}
	// Dist from high: (110 - 120) / 120 * 100 = -8.33%
	if math.Abs(result.DistFromHigh52wPct-(-8.333)) > 0.01 {
		t.Errorf("DistFromHigh52wPct = %.2f, want -8.33", result.DistFromHigh52wPct)
	}
	// Dist from low: (110 - 90) / 90 * 100 = 22.22%
	if math.Abs(result.DistFromLow52wPct-22.222) > 0.01 {
		t.Errorf("DistFromLow52wPct = %.2f, want 22.22", result.DistFromLow52wPct)
	}
}

func TestCompute52Week_Empty(t *testing.T) {
	result := Compute52Week(nil, 100)
	if result.High52w != 0 {
		t.Error("expected zero High52w for empty prices")
	}
}
