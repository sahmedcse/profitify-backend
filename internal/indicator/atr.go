package indicator

import (
	"fmt"
	"math"

	"github.com/profitify/profitify-backend/internal/domain"
)

// ComputeATR calculates the Average True Range over the given period.
// Prices must be ordered chronologically (oldest first) and must have
// at least period+1 entries (the first bar provides the "previous close"
// for the second bar's true range calculation).
//
// True Range = max(High-Low, |High-PrevClose|, |Low-PrevClose|)
// ATR = SMA of True Range over `period` bars.
func ComputeATR(prices []domain.DailyPrice, period int) (float64, error) {
	if period <= 0 {
		return 0, fmt.Errorf("atr: period must be > 0, got %d", period)
	}
	if len(prices) < period+1 {
		return 0, fmt.Errorf("atr: need at least %d prices, got %d", period+1, len(prices))
	}

	// Use the last (period+1) bars so we get exactly `period` true range values.
	start := len(prices) - period - 1

	var sum float64
	for i := start + 1; i < len(prices); i++ {
		prevClose := prices[i-1].Close
		h := prices[i].High
		l := prices[i].Low
		tr := math.Max(h-l, math.Max(math.Abs(h-prevClose), math.Abs(l-prevClose)))
		sum += tr
	}

	return sum / float64(period), nil
}
