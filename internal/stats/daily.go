package stats

import "github.com/profitify/profitify-backend/internal/domain"

// DailyResult holds single-day price and volume metrics.
type DailyResult struct {
	PriceChange     float64
	PriceChangePct  float64
	VolumeChangePct float64
	DayRange        float64
	GapPct          float64
	RelativeVolume  float64
}

// ComputeDaily computes today-vs-yesterday metrics.
// avgVolume30d is used for relative volume; pass 0 to skip.
func ComputeDaily(today, yesterday domain.DailyPrice, avgVolume30d float64) DailyResult {
	result := DailyResult{
		DayRange: today.High - today.Low,
	}

	if yesterday.Close > 0 {
		result.PriceChange = today.Close - yesterday.Close
		result.PriceChangePct = (result.PriceChange / yesterday.Close) * 100
		result.GapPct = ((today.Open - yesterday.Close) / yesterday.Close) * 100
	}

	if yesterday.Volume > 0 {
		result.VolumeChangePct = ((today.Volume - yesterday.Volume) / yesterday.Volume) * 100
	}

	if avgVolume30d > 0 {
		result.RelativeVolume = today.Volume / avgVolume30d
	}

	return result
}
