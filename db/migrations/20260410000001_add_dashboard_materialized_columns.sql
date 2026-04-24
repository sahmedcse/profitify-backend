-- +goose Up
-- Add materialized columns used by the dashboard API and enrichment pipeline.

ALTER TABLE tickers ADD COLUMN IF NOT EXISTS sector TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_tickers_sector ON tickers (sector) WHERE sector <> '';

ALTER TABLE ticker_technicals ADD COLUMN IF NOT EXISTS indicator_statuses JSONB NOT NULL DEFAULT '{}';

ALTER TABLE ticker_stats
    ADD COLUMN IF NOT EXISTS signal_label TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS signal_strength INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS pivot_levels JSONB NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE ticker_stats
    DROP COLUMN IF EXISTS pivot_levels,
    DROP COLUMN IF EXISTS signal_strength,
    DROP COLUMN IF EXISTS signal_label;

ALTER TABLE ticker_technicals DROP COLUMN IF EXISTS indicator_statuses;

DROP INDEX IF EXISTS idx_tickers_sector;
ALTER TABLE tickers DROP COLUMN IF EXISTS sector;
