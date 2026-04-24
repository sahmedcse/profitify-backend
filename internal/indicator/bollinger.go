// Package indicator contains pure numerical functions that compute
// technical indicators from OHLCV data. These are the self-computed
// indicators that Massive does not provide directly.
package indicator

import (
	"fmt"
	"math"
)

// ComputeBollinger calculates Bollinger Bands from a slice of closing prices.
// Returns (upper, middle, lower) bands. The middle band is a simple moving
// average of `period` bars, and the upper/lower bands are `multiplier`
// standard deviations above/below it.
//
// Requires at least `period` data points. The closes slice should be ordered
// chronologically (oldest first); the function uses the last `period` values.
func ComputeBollinger(closes []float64, period int, multiplier float64) (upper, mid, lower float64, err error) {
	if period <= 0 {
		return 0, 0, 0, fmt.Errorf("bollinger: period must be > 0, got %d", period)
	}
	if len(closes) < period {
		return 0, 0, 0, fmt.Errorf("bollinger: need at least %d closes, got %d", period, len(closes))
	}

	window := closes[len(closes)-period:]

	// SMA
	var sum float64
	for _, c := range window {
		sum += c
	}
	mid = sum / float64(period)

	// Standard deviation
	var sqDiff float64
	for _, c := range window {
		d := c - mid
		sqDiff += d * d
	}
	stdDev := math.Sqrt(sqDiff / float64(period))

	upper = mid + multiplier*stdDev
	lower = mid - multiplier*stdDev
	return upper, mid, lower, nil
}
