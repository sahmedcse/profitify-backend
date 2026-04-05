package domain

import "time"

// Ticker represents a tradeable security's identity.
type Ticker struct {
	ID              string    `json:"id"`
	Ticker          string    `json:"ticker"`
	Name            string    `json:"name"`
	Market          string    `json:"market"`
	PrimaryExchange string    `json:"primary_exchange"`
	Type            string    `json:"type"`
	Active          bool      `json:"active"`
	CurrencyName    string    `json:"currency_name"`
	Locale          string    `json:"locale"`
	CIK             string    `json:"cik"`
	ListDate        string    `json:"list_date"`
	DelistedUTC     string    `json:"delisted_utc"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
