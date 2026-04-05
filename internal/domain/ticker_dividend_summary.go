package domain

import "time"

// TickerDividendSummary holds computed dividend metrics for a ticker.
type TickerDividendSummary struct {
	ID                    string    `json:"id"`
	TickerID              string    `json:"ticker_id"`
	Time                  time.Time `json:"time"`
	CurrentYield          *float64  `json:"current_yield"`
	ForwardYield          *float64  `json:"forward_yield"`
	TrailingYield12m      *float64  `json:"trailing_yield_12m"`
	DividendGrowthRate1y  *float64  `json:"dividend_growth_rate_1y"`
	DividendGrowthRate3y  *float64  `json:"dividend_growth_rate_3y"`
	DividendGrowthRate5y  *float64  `json:"dividend_growth_rate_5y"`
	ConsecutiveIncreases  int       `json:"consecutive_increases"`
	NextExDividendDate    *string   `json:"next_ex_dividend_date"`
	DaysUntilExDividend   *int      `json:"days_until_ex_dividend"`
	PayoutFrequency       int       `json:"payout_frequency"`
	LatestDistributionType string   `json:"latest_distribution_type"`
}
