-- +goose Up
CREATE TABLE tickers (
    id               UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    ticker           TEXT UNIQUE NOT NULL,
    name             TEXT NOT NULL DEFAULT '',
    market           TEXT NOT NULL DEFAULT '',
    primary_exchange TEXT NOT NULL DEFAULT '',
    type             TEXT NOT NULL DEFAULT '',
    active           BOOLEAN NOT NULL DEFAULT TRUE,
    currency_name    TEXT NOT NULL DEFAULT '',
    locale           TEXT NOT NULL DEFAULT '',
    cik              TEXT NOT NULL DEFAULT '',
    list_date        TEXT NOT NULL DEFAULT '',
    delisted_utc     TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tickers_active ON tickers (active) WHERE active = TRUE;

-- +goose Down
DROP TABLE IF EXISTS tickers CASCADE;
