package domain

import "time"

// TickerStats holds daily and rolling window statistics for a ticker.
// All values are computed from OHLCV data in the database.
type TickerStats struct {
	ID       string    `json:"id"`
	TickerID string    `json:"ticker_id"`
	Time     time.Time `json:"time"`

	// Daily metrics (today vs yesterday)
	PriceChange     float64 `json:"price_change"`
	PriceChangePct  float64 `json:"price_change_pct"`
	VolumeChangePct float64 `json:"volume_change_pct"`
	DayRange        float64 `json:"day_range"`
	GapPct          float64 `json:"gap_pct"`
	RelativeVolume  float64 `json:"relative_volume"`

	// 7-day rolling
	Return7d      *float64 `json:"return_7d"`
	Volatility7d  *float64 `json:"volatility_7d"`
	AvgVolume7d   *float64 `json:"avg_volume_7d"`
	MaxDrawdown7d *float64 `json:"max_drawdown_7d"`

	// 30-day rolling
	Return30d      *float64 `json:"return_30d"`
	Volatility30d  *float64 `json:"volatility_30d"`
	AvgVolume30d   *float64 `json:"avg_volume_30d"`
	MaxDrawdown30d *float64 `json:"max_drawdown_30d"`

	// 90-day rolling
	Return90d      *float64 `json:"return_90d"`
	Volatility90d  *float64 `json:"volatility_90d"`
	AvgVolume90d   *float64 `json:"avg_volume_90d"`
	MaxDrawdown90d *float64 `json:"max_drawdown_90d"`

	// 52-week extremes
	High52w        *float64 `json:"high_52w"`
	Low52w         *float64 `json:"low_52w"`
	DistFromHigh52wPct *float64 `json:"dist_from_high_52w_pct"`
	DistFromLow52wPct  *float64 `json:"dist_from_low_52w_pct"`
}
