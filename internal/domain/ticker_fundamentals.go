package domain

import "time"

// Address represents a company's physical address.
type Address struct {
	Address1   string `json:"address1"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
}

// Branding holds company branding assets.
type Branding struct {
	IconURL string `json:"icon_url"`
	LogoURL string `json:"logo_url"`
}

// TickerFundamentals holds company fundamental data from Polygon.io ticker details.
type TickerFundamentals struct {
	ID                        string    `json:"id"`
	TickerID                  string    `json:"ticker_id"`
	Time                      time.Time `json:"time"`
	MarketCap                 float64   `json:"market_cap"`
	SharesOutstanding         int64     `json:"share_class_shares_outstanding"`
	WeightedSharesOutstanding int64     `json:"weighted_shares_outstanding"`
	SICCode                   string    `json:"sic_code"`
	SICDescription            string    `json:"sic_description"`
	Description               string    `json:"description"`
	HomepageURL               string    `json:"homepage_url"`
	PhoneNumber               string    `json:"phone_number"`
	TotalEmployees            int       `json:"total_employees"`
	Address                   Address   `json:"address"`
	Branding                  Branding  `json:"branding"`
}
