package domain

import "time"

// TickerLeadLag represents a lead/lag relationship where one ticker's move
// predicts another ticker's move on a subsequent day.
type TickerLeadLag struct {
	ID              string    `json:"id"`
	LeadTickerID    string    `json:"lead_ticker_id"`
	LagTickerID     string    `json:"lag_ticker_id"`
	LagDays         int       `json:"lag_days"`
	PredictiveScore float64   `json:"predictive_score"`
	Rank            int       `json:"rank"`
	Time            time.Time `json:"time"`
}
