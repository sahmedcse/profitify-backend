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

// Pipeline stage names — one per Lambda in the pipeline.
const (
	StageStartPipeline     = "start_pipeline"
	StageIngestOHLCV       = "ingest_ohlcv"
	StageFetchTechnicals   = "fetch_technicals"
	StageFetchFundamentals = "fetch_fundamentals"
	StageEnrichTicker      = "enrich_ticker"
	StageComputeStats      = "compute_stats"
)

// AllStages is the ordered list of all pipeline stages.
var AllStages = []string{
	StageStartPipeline,
	StageIngestOHLCV,
	StageFetchTechnicals,
	StageFetchFundamentals,
	StageEnrichTicker,
	StageComputeStats,
}

// PipelineRun represents a single per-ticker execution of the enrichment pipeline.
type PipelineRun struct {
	ID              string     `json:"id"`
	TickerID        string     `json:"ticker_id"`
	Ticker          string     `json:"ticker"`
	Date            time.Time  `json:"date"`
	SFNExecutionArn string     `json:"sfn_execution_arn"`
	Status          string     `json:"status"`
	ErrorMessage    string     `json:"error_message"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
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
