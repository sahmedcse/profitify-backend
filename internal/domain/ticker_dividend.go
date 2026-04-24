package domain

// TickerDividend represents a single dividend event from Massive.
type TickerDividend struct {
	ID                       string  `json:"id"`
	TickerID                 string  `json:"ticker_id"`
	CashAmount               float64 `json:"cash_amount"`
	SplitAdjustedCashAmount  float64 `json:"split_adjusted_cash_amount"`
	Currency                 string  `json:"currency"`
	ExDividendDate           string  `json:"ex_dividend_date"`
	DeclarationDate          string  `json:"declaration_date"`
	RecordDate               string  `json:"record_date"`
	PayDate                  string  `json:"pay_date"`
	Frequency                int     `json:"frequency"`
	DistributionType         string  `json:"distribution_type"`
	HistoricalAdjustmentFactor float64 `json:"historical_adjustment_factor"`
}
