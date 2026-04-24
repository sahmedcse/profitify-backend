-- +goose Up

-- pipeline_runs: one row per pipeline execution.
CREATE TABLE pipeline_runs (
    id                UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    run_params        JSONB NOT NULL DEFAULT '{}',
    sfn_execution_arn TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'pending',
    ticker_count      INT  NOT NULL DEFAULT 0,
    completed_count   INT  NOT NULL DEFAULT 0,
    failed_count      INT  NOT NULL DEFAULT 0,
    error_message     TEXT NOT NULL DEFAULT '',
    started_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pipeline_runs_date ON pipeline_runs ((run_params->>'date'));
CREATE INDEX idx_pipeline_runs_status ON pipeline_runs (status);

-- pipeline_ticker_stages: one row per ticker per stage per run.
CREATE TABLE pipeline_ticker_stages (
    id                UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    run_id            UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    ticker_id         UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    ticker            TEXT NOT NULL,
    stage             TEXT NOT NULL,
    sfn_execution_arn TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'pending',
    error_message     TEXT NOT NULL DEFAULT '',
    started_at        TIMESTAMPTZ,
    completed_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_pts_run_ticker_stage ON pipeline_ticker_stages (run_id, ticker_id, stage);
CREATE INDEX idx_pts_run_status ON pipeline_ticker_stages (run_id, status);
CREATE INDEX idx_pts_ticker_id ON pipeline_ticker_stages (ticker_id);

-- +goose Down
DROP TABLE IF EXISTS pipeline_ticker_stages CASCADE;
DROP TABLE IF EXISTS pipeline_runs CASCADE;
