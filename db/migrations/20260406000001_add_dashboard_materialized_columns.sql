-- +goose Up

-- Dashboard-facing materialized columns. These values are derived from
-- existing data and written by the enrich-ticker and compute-stats pipeline
-- stages. Storing them lets the dashboard API do thin SELECTs instead of
-- recomputing classifiers on every request.

ALTER TABLE tickers
    ADD COLUMN sector TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_tickers_sector ON tickers (sector) WHERE sector <> '';

ALTER TABLE ticker_technicals
    ADD COLUMN indicator_statuses JSONB NOT NULL DEFAULT '{}';

ALTER TABLE ticker_stats
    ADD COLUMN signal_label    TEXT NOT NULL DEFAULT '',
    ADD COLUMN signal_strength INT  NOT NULL DEFAULT 0,
    ADD COLUMN pivot_levels    JSONB NOT NULL DEFAULT '{}';

-- +goose Down

ALTER TABLE ticker_stats
    DROP COLUMN IF EXISTS pivot_levels,
    DROP COLUMN IF EXISTS signal_strength,
    DROP COLUMN IF EXISTS signal_label;

ALTER TABLE ticker_technicals
    DROP COLUMN IF EXISTS indicator_statuses;

DROP INDEX IF EXISTS idx_tickers_sector;

ALTER TABLE tickers
    DROP COLUMN IF EXISTS sector;
