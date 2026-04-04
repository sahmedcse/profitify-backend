package domain

import "time"

// TickerDividendAnalytics holds cross-stock dividend analysis for a ticker.
type TickerDividendAnalytics struct {
	ID                  string    `json:"id"`
	TickerID            string    `json:"ticker_id"`
	Time                time.Time `json:"time"`
	SectorAvgYield      *float64  `json:"sector_avg_yield"`
	YieldVsSector       *float64  `json:"yield_vs_sector"`        // positive = above avg, negative = below
	VolatilityBucket    string    `json:"volatility_bucket"`      // low, moderate, high
	IncomeQualityScore  *float64  `json:"income_quality_score"`   // yield / volatility ratio
	ExDividendPriceImpact *float64 `json:"ex_dividend_price_impact"` // actual drop vs expected
}
