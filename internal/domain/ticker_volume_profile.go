package domain

import "time"

// TickerVolumeProfile represents a volume pattern similarity between two tickers.
type TickerVolumeProfile struct {
	ID              string    `json:"id"`
	TickerID        string    `json:"ticker_id"`
	RelatedTickerID string    `json:"related_ticker_id"`
	SimilarityScore float64   `json:"similarity_score"`
	Rank            int       `json:"rank"`
	Time            time.Time `json:"time"`
}
