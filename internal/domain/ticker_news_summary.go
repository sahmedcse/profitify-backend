package domain

import "time"

// TickerNewsSummary holds aggregated daily news metrics for a ticker.
type TickerNewsSummary struct {
	ID             string    `json:"id"`
	TickerID       string    `json:"ticker_id"`
	Time           time.Time `json:"time"`
	ArticleCount   int       `json:"article_count"`
	AvgSentiment   float64   `json:"avg_sentiment"`
	PositiveCount  int       `json:"positive_count"`
	NegativeCount  int       `json:"negative_count"`
	NeutralCount   int       `json:"neutral_count"`
}
