package domain

import "time"

// TickerNews represents a news article from Massive.
type TickerNews struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Author        string    `json:"author"`
	ArticleURL    string    `json:"article_url"`
	ImageURL      string    `json:"image_url"`
	PublisherName string    `json:"publisher_name"`
	Keywords      []string  `json:"keywords"`
	Tickers       []string  `json:"tickers"`
	PublishedUTC  time.Time `json:"published_utc"`
}
