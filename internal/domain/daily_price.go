package domain

import "time"

// DailyPrice represents a single day's OHLCV bar for a ticker.
type DailyPrice struct {
	ID         string    `json:"id"`
	TickerID   string    `json:"ticker_id"`
	Time       time.Time `json:"time"`
	Open       float64   `json:"open"`
	High       float64   `json:"high"`
	Low        float64   `json:"low"`
	Close      float64   `json:"close"`
	Volume     float64   `json:"volume"`
	VWAP       float64   `json:"vwap"`
	PreMarket  float64   `json:"pre_market"`
	AfterHours float64   `json:"after_hours"`
	OTC        bool      `json:"otc"`
}
