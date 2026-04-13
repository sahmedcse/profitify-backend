package domain

import "time"

// Pipeline run statuses.
const (
	PipelineStatusPending   = "pending"
	PipelineStatusRunning   = "running"
	PipelineStatusCompleted = "completed"
	PipelineStatusFailed    = "failed"
	PipelineStatusSkipped   = "skipped"
)

// Pipeline stage names — one per per-ticker Lambda.
const (
	StageIngestOHLCV       = "ingest_ohlcv"
	StageFetchTechnicals   = "fetch_technicals"
	StageFetchFundamentals = "fetch_fundamentals"
	StageEnrichTicker      = "enrich_ticker"
	StageComputeStats      = "compute_stats"
)

// AllStages is the ordered list of pipeline stages. Used by FetchTickers
// to bulk-insert pending stage rows for every active ticker.
var AllStages = []string{
	StageIngestOHLCV,
	StageFetchTechnicals,
	StageFetchFundamentals,
	StageEnrichTicker,
	StageComputeStats,
}

// PipelineRunParams holds the input parameters for a pipeline run.
// Currently only Date; extensible via JSONB storage.
type PipelineRunParams struct {
	Date string `json:"date"` // "2006-01-02" format
}

// PipelineRun represents a single execution of the enrichment pipeline.
type PipelineRun struct {
	ID              string            `json:"id"`
	RunParams       PipelineRunParams `json:"run_params"`
	SFNExecutionArn string            `json:"sfn_execution_arn"`
	Status          string            `json:"status"`
	TickerCount     int               `json:"ticker_count"`
	CompletedCount  int               `json:"completed_count"`
	FailedCount     int               `json:"failed_count"`
	ErrorMessage    string            `json:"error_message"`
	StartedAt       time.Time         `json:"started_at"`
	CompletedAt     *time.Time        `json:"completed_at"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// PipelineTickerStage tracks a single ticker's progress through one
// pipeline stage within a run.
type PipelineTickerStage struct {
	ID              string     `json:"id"`
	RunID           string     `json:"run_id"`
	TickerID        string     `json:"ticker_id"`
	Ticker          string     `json:"ticker"`
	Stage           string     `json:"stage"`
	SFNExecutionArn string     `json:"sfn_execution_arn"`
	Status          string     `json:"status"`
	ErrorMessage    string     `json:"error_message"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
