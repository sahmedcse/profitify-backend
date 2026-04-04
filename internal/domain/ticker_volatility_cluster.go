package domain

import "time"

// TickerVolatilityCluster groups tickers by similar volatility bands.
type TickerVolatilityCluster struct {
	ID           string    `json:"id"`
	TickerID     string    `json:"ticker_id"`
	ClusterID    int       `json:"cluster_id"`
	Volatility30d float64  `json:"volatility_30d"`
	Rank         int       `json:"rank"`
	Time         time.Time `json:"time"`
}
