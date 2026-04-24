package stats

import "github.com/profitify/profitify-backend/internal/domain"

// FiftyTwoWeekResult holds 52-week high/low extremes and distance metrics.
type FiftyTwoWeekResult struct {
	High52w            float64
	Low52w             float64
	DistFromHigh52wPct float64 // negative when below high
	DistFromLow52wPct  float64 // positive when above low
}

// Compute52Week scans a chronologically ordered slice of daily bars to find
// the 52-week high and low, and computes how far the latest close is from each.
func Compute52Week(prices []domain.DailyPrice, latestClose float64) FiftyTwoWeekResult {
	if len(prices) == 0 {
		return FiftyTwoWeekResult{}
	}

	high := prices[0].High
	low := prices[0].Low
	for _, p := range prices {
		if p.High > high {
			high = p.High
		}
		if p.Low < low {
			low = p.Low
		}
	}

	result := FiftyTwoWeekResult{
		High52w: high,
		Low52w:  low,
	}

	if high > 0 {
		result.DistFromHigh52wPct = ((latestClose - high) / high) * 100
	}
	if low > 0 {
		result.DistFromLow52wPct = ((latestClose - low) / low) * 100
	}

	return result
}
