// Package pipeline defines shared types used by all per-ticker Lambda
// functions in the enrichment pipeline.
package pipeline

// TickerEvent is the input/output contract for per-ticker pipeline Lambdas.
// Each Lambda receives a TickerEvent from the Step Function and returns
// an augmented response that includes these base fields plus any
// Lambda-specific output fields.
type TickerEvent struct {
	Ticker   string `json:"ticker"`    // e.g. "AAPL"
	TickerID string `json:"ticker_id"` // UUID from tickers table
	Date     string `json:"date"`      // "2006-01-02" format
	RunID    string `json:"run_id"`    // UUID from pipeline_runs table (empty = no tracking)
}
