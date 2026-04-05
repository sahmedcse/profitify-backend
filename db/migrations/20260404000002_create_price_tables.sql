-- +goose Up

CREATE TABLE daily_prices (
    id          UUID DEFAULT gen_random_uuid(),
    ticker_id   UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    time        TIMESTAMPTZ NOT NULL,
    open        NUMERIC(18,6),
    high        NUMERIC(18,6),
    low         NUMERIC(18,6),
    close       NUMERIC(18,6),
    volume      NUMERIC(18,6),
    vwap        NUMERIC(18,6),
    pre_market  NUMERIC(18,6),
    after_hours NUMERIC(18,6),
    otc         BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (ticker_id, time)
);

SELECT create_hypertable('daily_prices', 'time', chunk_time_interval => INTERVAL '1 month');

CREATE TABLE ticker_technicals (
    id               UUID DEFAULT gen_random_uuid(),
    ticker_id        UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    time             TIMESTAMPTZ NOT NULL,
    sma_20           DOUBLE PRECISION,
    sma_50           DOUBLE PRECISION,
    sma_200          DOUBLE PRECISION,
    ema_12           DOUBLE PRECISION,
    ema_26           DOUBLE PRECISION,
    rsi_14           DOUBLE PRECISION,
    macd_line        DOUBLE PRECISION,
    macd_signal      DOUBLE PRECISION,
    macd_histogram   DOUBLE PRECISION,
    bollinger_upper  DOUBLE PRECISION,
    bollinger_middle DOUBLE PRECISION,
    bollinger_lower  DOUBLE PRECISION,
    atr_14           DOUBLE PRECISION,
    obv              DOUBLE PRECISION,
    UNIQUE (ticker_id, time)
);

SELECT create_hypertable('ticker_technicals', 'time', chunk_time_interval => INTERVAL '1 month');

CREATE TABLE ticker_stats (
    id                     UUID DEFAULT gen_random_uuid(),
    ticker_id              UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    time                   TIMESTAMPTZ NOT NULL,
    price_change           DOUBLE PRECISION NOT NULL DEFAULT 0,
    price_change_pct       DOUBLE PRECISION NOT NULL DEFAULT 0,
    volume_change_pct      DOUBLE PRECISION NOT NULL DEFAULT 0,
    day_range              DOUBLE PRECISION NOT NULL DEFAULT 0,
    gap_pct                DOUBLE PRECISION NOT NULL DEFAULT 0,
    relative_volume        DOUBLE PRECISION NOT NULL DEFAULT 0,
    price_return_7d        DOUBLE PRECISION,
    dividend_return_7d     DOUBLE PRECISION,
    total_return_7d        DOUBLE PRECISION,
    volatility_7d          DOUBLE PRECISION,
    avg_volume_7d          DOUBLE PRECISION,
    max_drawdown_7d        DOUBLE PRECISION,
    price_return_30d       DOUBLE PRECISION,
    dividend_return_30d    DOUBLE PRECISION,
    total_return_30d       DOUBLE PRECISION,
    volatility_30d         DOUBLE PRECISION,
    avg_volume_30d         DOUBLE PRECISION,
    max_drawdown_30d       DOUBLE PRECISION,
    price_return_90d       DOUBLE PRECISION,
    dividend_return_90d    DOUBLE PRECISION,
    total_return_90d       DOUBLE PRECISION,
    volatility_90d         DOUBLE PRECISION,
    avg_volume_90d         DOUBLE PRECISION,
    max_drawdown_90d       DOUBLE PRECISION,
    high_52w               DOUBLE PRECISION,
    low_52w                DOUBLE PRECISION,
    dist_from_high_52w_pct DOUBLE PRECISION,
    dist_from_low_52w_pct  DOUBLE PRECISION,
    UNIQUE (ticker_id, time)
);

SELECT create_hypertable('ticker_stats', 'time', chunk_time_interval => INTERVAL '1 month');

-- +goose Down
DROP TABLE IF EXISTS ticker_stats;
DROP TABLE IF EXISTS ticker_technicals;
DROP TABLE IF EXISTS daily_prices;
