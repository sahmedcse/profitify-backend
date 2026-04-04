package domain

// TickerNewsInsight holds per-ticker sentiment analysis for a news article.
type TickerNewsInsight struct {
	ID                 string `json:"id"`
	NewsID             string `json:"news_id"`
	TickerID           string `json:"ticker_id"`
	Sentiment          string `json:"sentiment"`           // positive, negative, neutral
	SentimentReasoning string `json:"sentiment_reasoning"`
}
