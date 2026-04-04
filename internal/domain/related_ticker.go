package domain

import "time"

// RelatedTicker represents a Polygon.io-provided relationship between two tickers.
type RelatedTicker struct {
	ID              string    `json:"id"`
	TickerID        string    `json:"ticker_id"`
	RelatedTickerID string    `json:"related_ticker_id"`
	Time            time.Time `json:"time"`
}
