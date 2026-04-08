package stats

import "github.com/profitify/profitify-backend/internal/domain"

// NormalizeSeries converts a series of daily prices into normalized
// percentage returns relative to the first close. The first element is
// always 0. Later elements are (close[i] / close[0] - 1) * 100.
//
// Returns nil for empty input and also nil if the first close is zero
// (which would make normalization undefined).
//
// This is used by GET /v1/compare to put two tickers on the same Y axis
// for the "Stock vs Stock" chart.
func NormalizeSeries(prices []domain.DailyPrice) []float64 {
	if len(prices) == 0 {
		return nil
	}
	base := prices[0].Close
	if base == 0 {
		return nil
	}
	out := make([]float64, len(prices))
	for i, p := range prices {
		out[i] = (p.Close/base - 1) * 100
	}
	return out
}
