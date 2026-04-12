package stats

import (
	"math"
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
)

func TestComputeRolling_BasicReturn(t *testing.T) {
	prices := []domain.DailyPrice{
		{Close: 100, Volume: 1000},
		{Close: 102, Volume: 1200},
		{Close: 105, Volume: 1100},
	}

	result := ComputeRolling(prices)

	// 5% return from 100 → 105
	if math.Abs(result.PriceReturn-5.0) > 0.01 {
		t.Errorf("PriceReturn = %.2f, want 5.0", result.PriceReturn)
	}
	if math.Abs(result.AvgVolume-1100) > 0.01 {
		t.Errorf("AvgVolume = %.2f, want 1100", result.AvgVolume)
	}
	if result.MaxDrawdown != 0 {
		t.Errorf("MaxDrawdown = %.2f, want 0 (monotonically increasing)", result.MaxDrawdown)
	}
}

func TestComputeRolling_Drawdown(t *testing.T) {
	prices := []domain.DailyPrice{
		{Close: 100, Volume: 1000},
		{Close: 110, Volume: 1000}, // peak
		{Close: 99, Volume: 1000},  // trough (10% drawdown from 110)
		{Close: 105, Volume: 1000},
	}

	result := ComputeRolling(prices)

	// Max drawdown: (110 - 99) / 110 * 100 = 10%
	if math.Abs(result.MaxDrawdown-10.0) > 0.01 {
		t.Errorf("MaxDrawdown = %.2f, want 10.0", result.MaxDrawdown)
	}
}

func TestComputeRolling_Volatility(t *testing.T) {
	prices := []domain.DailyPrice{
		{Close: 100, Volume: 1000},
		{Close: 102, Volume: 1000},
		{Close: 98, Volume: 1000},
		{Close: 103, Volume: 1000},
		{Close: 97, Volume: 1000},
	}

	result := ComputeRolling(prices)

	if result.Volatility <= 0 {
		t.Error("expected positive volatility for varying prices")
	}
}

func TestComputeRolling_InsufficientData(t *testing.T) {
	prices := []domain.DailyPrice{{Close: 100}}
	result := ComputeRolling(prices)

	if result.PriceReturn != 0 {
		t.Error("expected zero PriceReturn for single bar")
	}
}
