-- +goose Up

CREATE TABLE related_tickers (
    id                UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    ticker_id         UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    related_ticker_id UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    time              TIMESTAMPTZ NOT NULL,
    UNIQUE (ticker_id, related_ticker_id, time)
);

CREATE INDEX idx_related_tickers_ticker_id ON related_tickers (ticker_id);
CREATE INDEX idx_related_tickers_related_ticker_id ON related_tickers (related_ticker_id);

CREATE TABLE ticker_correlations (
    id                UUID DEFAULT gen_random_uuid(),
    ticker_id         UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    related_ticker_id UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    correlation_30d   DOUBLE PRECISION NOT NULL DEFAULT 0,
    correlation_90d   DOUBLE PRECISION NOT NULL DEFAULT 0,
    rank              INT NOT NULL DEFAULT 0,
    time              TIMESTAMPTZ NOT NULL,
    UNIQUE (ticker_id, related_ticker_id, time)
);

SELECT create_hypertable('ticker_correlations', 'time', chunk_time_interval => INTERVAL '1 month');

CREATE TABLE ticker_lead_lag (
    id               UUID DEFAULT gen_random_uuid(),
    lead_ticker_id   UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    lag_ticker_id    UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    lag_days         INT NOT NULL DEFAULT 0,
    predictive_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    rank             INT NOT NULL DEFAULT 0,
    time             TIMESTAMPTZ NOT NULL,
    UNIQUE (lead_ticker_id, lag_ticker_id, time)
);

SELECT create_hypertable('ticker_lead_lag', 'time', chunk_time_interval => INTERVAL '1 month');

CREATE TABLE ticker_volume_profiles (
    id                UUID DEFAULT gen_random_uuid(),
    ticker_id         UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    related_ticker_id UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    similarity_score  DOUBLE PRECISION NOT NULL DEFAULT 0,
    rank              INT NOT NULL DEFAULT 0,
    time              TIMESTAMPTZ NOT NULL,
    UNIQUE (ticker_id, related_ticker_id, time)
);

SELECT create_hypertable('ticker_volume_profiles', 'time', chunk_time_interval => INTERVAL '1 month');

CREATE TABLE ticker_volatility_clusters (
    id             UUID DEFAULT gen_random_uuid(),
    ticker_id      UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    cluster_id     INT NOT NULL DEFAULT 0,
    volatility_30d DOUBLE PRECISION NOT NULL DEFAULT 0,
    rank           INT NOT NULL DEFAULT 0,
    time           TIMESTAMPTZ NOT NULL,
    UNIQUE (ticker_id, time)
);

SELECT create_hypertable('ticker_volatility_clusters', 'time', chunk_time_interval => INTERVAL '1 month');

-- +goose Down
DROP TABLE IF EXISTS ticker_volatility_clusters;
DROP TABLE IF EXISTS ticker_volume_profiles;
DROP TABLE IF EXISTS ticker_lead_lag;
DROP TABLE IF EXISTS ticker_correlations;
DROP TABLE IF EXISTS related_tickers;
