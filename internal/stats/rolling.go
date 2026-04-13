package stats

import (
	"math"

	"github.com/profitify/profitify-backend/internal/domain"
)

// RollingResult holds statistics over a window of daily bars.
type RollingResult struct {
	PriceReturn float64
	Volatility  float64 // annualised std dev of daily returns (%)
	AvgVolume   float64
	MaxDrawdown float64 // maximum peak-to-trough decline (%)
}

// ComputeRolling computes rolling statistics from a chronologically ordered
// slice of daily bars. Requires at least 2 bars.
func ComputeRolling(prices []domain.DailyPrice) RollingResult {
	if len(prices) < 2 {
		return RollingResult{}
	}

	first := prices[0].Close
	last := prices[len(prices)-1].Close

	// Price return
	var priceReturn float64
	if first > 0 {
		priceReturn = ((last - first) / first) * 100
	}

	// Average volume
	var totalVol float64
	for _, p := range prices {
		totalVol += p.Volume
	}
	avgVol := totalVol / float64(len(prices))

	// Daily log returns for volatility
	var returns []float64
	for i := 1; i < len(prices); i++ {
		if prices[i-1].Close > 0 {
			r := (prices[i].Close - prices[i-1].Close) / prices[i-1].Close
			returns = append(returns, r)
		}
	}

	var volatility float64
	if len(returns) > 1 {
		mean := 0.0
		for _, r := range returns {
			mean += r
		}
		mean /= float64(len(returns))

		var variance float64
		for _, r := range returns {
			diff := r - mean
			variance += diff * diff
		}
		variance /= float64(len(returns) - 1)
		volatility = math.Sqrt(variance) * math.Sqrt(252) * 100 // annualised
	}

	// Max drawdown
	peak := prices[0].Close
	var maxDD float64
	for _, p := range prices {
		if p.Close > peak {
			peak = p.Close
		}
		if peak > 0 {
			dd := (peak - p.Close) / peak * 100
			if dd > maxDD {
				maxDD = dd
			}
		}
	}

	return RollingResult{
		PriceReturn: priceReturn,
		Volatility:  volatility,
		AvgVolume:   avgVol,
		MaxDrawdown: maxDD,
	}
}
