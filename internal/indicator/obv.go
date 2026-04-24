package indicator

import (
	"fmt"

	"github.com/profitify/profitify-backend/internal/domain"
)

// ComputeOBV calculates On-Balance Volume from a chronological slice of
// daily prices. OBV accumulates volume: if today's close > yesterday's
// close, add today's volume; if lower, subtract it; if equal, no change.
//
// Returns the final OBV value. Requires at least 2 bars.
func ComputeOBV(prices []domain.DailyPrice) (float64, error) {
	if len(prices) < 2 {
		return 0, fmt.Errorf("obv: need at least 2 prices, got %d", len(prices))
	}

	var obv float64
	for i := 1; i < len(prices); i++ {
		switch {
		case prices[i].Close > prices[i-1].Close:
			obv += prices[i].Volume
		case prices[i].Close < prices[i-1].Close:
			obv -= prices[i].Volume
		}
	}
	return obv, nil
}
