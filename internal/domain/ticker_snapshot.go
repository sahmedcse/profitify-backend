package domain

// Bar represents an OHLCV + VWAP bar. Reused for day, prevDay, and minute data.
type Bar struct {
	Open   float64 `json:"o"`
	High   float64 `json:"h"`
	Low    float64 `json:"l"`
	Close  float64 `json:"c"`
	Volume float64 `json:"v"`
	VWAP   float64 `json:"vw"`
}

// MinuteBar extends Bar with accumulated volume, trade count, and timestamp.
type MinuteBar struct {
	Bar
	AccumulatedVolume float64 `json:"av"`
	NumTrades         int64   `json:"n"`
	Timestamp         int64   `json:"t"`
}

// LastTrade represents the most recent trade for a ticker.
type LastTrade struct {
	Conditions []int   `json:"c"`
	ID         string  `json:"i"`
	Price      float64 `json:"p"`
	Size       float64 `json:"s"`
	Timestamp  int64   `json:"t"`
	Exchange   int     `json:"x"`
}

// LastQuote represents the most recent NBBO quote for a ticker.
type LastQuote struct {
	BidPrice float64 `json:"p"`
	BidSize  float64 `json:"s"`
	AskPrice float64 `json:"P"`
	AskSize  float64 `json:"S"`
	Timestamp int64  `json:"t"`
}

// TickerSnapshot represents the real-time state of a ticker from the Snapshot endpoint.
type TickerSnapshot struct {
	TickerID        string    `json:"ticker_id"`
	Day             Bar       `json:"day"`
	PrevDay         Bar       `json:"prevDay"`
	MinuteBar       MinuteBar `json:"min"`
	LastTrade       LastTrade `json:"lastTrade"`
	LastQuote       LastQuote `json:"lastQuote"`
	TodaysChange    float64   `json:"todaysChange"`
	TodaysChangePct float64   `json:"todaysChangePerc"`
	Updated         int64     `json:"updated"`
}
