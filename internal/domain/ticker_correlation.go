package domain

import "time"

// TickerCorrelation holds the Pearson correlation of daily returns between two tickers.
type TickerCorrelation struct {
	ID              string    `json:"id"`
	TickerID        string    `json:"ticker_id"`
	RelatedTickerID string    `json:"related_ticker_id"`
	Correlation30d  float64   `json:"correlation_30d"`
	Correlation90d  float64   `json:"correlation_90d"`
	Rank            int       `json:"rank"`
	Time            time.Time `json:"time"`
}
