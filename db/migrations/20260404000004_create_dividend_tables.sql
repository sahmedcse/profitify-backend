-- +goose Up

CREATE TABLE ticker_dividends (
    id                           UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    ticker_id                    UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    cash_amount                  DOUBLE PRECISION NOT NULL DEFAULT 0,
    split_adjusted_cash_amount   DOUBLE PRECISION NOT NULL DEFAULT 0,
    currency                     TEXT NOT NULL DEFAULT '',
    ex_dividend_date             DATE NOT NULL,
    declaration_date             DATE,
    record_date                  DATE,
    pay_date                     DATE,
    frequency                    INT NOT NULL DEFAULT 0,
    distribution_type            TEXT NOT NULL DEFAULT '',
    historical_adjustment_factor DOUBLE PRECISION NOT NULL DEFAULT 1,
    UNIQUE (ticker_id, ex_dividend_date)
);

CREATE INDEX idx_ticker_dividends_ticker_id ON ticker_dividends (ticker_id);

CREATE TABLE ticker_dividend_summaries (
    id                       UUID DEFAULT gen_random_uuid(),
    ticker_id                UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    time                     TIMESTAMPTZ NOT NULL,
    current_yield            DOUBLE PRECISION,
    forward_yield            DOUBLE PRECISION,
    trailing_yield_12m       DOUBLE PRECISION,
    dividend_growth_rate_1y  DOUBLE PRECISION,
    dividend_growth_rate_3y  DOUBLE PRECISION,
    dividend_growth_rate_5y  DOUBLE PRECISION,
    consecutive_increases    INT NOT NULL DEFAULT 0,
    next_ex_dividend_date    DATE,
    days_until_ex_dividend   INT,
    payout_frequency         INT NOT NULL DEFAULT 0,
    latest_distribution_type TEXT NOT NULL DEFAULT '',
    UNIQUE (ticker_id, time)
);

SELECT create_hypertable('ticker_dividend_summaries', 'time', chunk_time_interval => INTERVAL '1 month');

CREATE TABLE ticker_dividend_analytics (
    id                       UUID DEFAULT gen_random_uuid(),
    ticker_id                UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    time                     TIMESTAMPTZ NOT NULL,
    sector_avg_yield         DOUBLE PRECISION,
    yield_vs_sector          DOUBLE PRECISION,
    volatility_bucket        TEXT NOT NULL DEFAULT '',
    income_quality_score     DOUBLE PRECISION,
    ex_dividend_price_impact DOUBLE PRECISION,
    UNIQUE (ticker_id, time)
);

SELECT create_hypertable('ticker_dividend_analytics', 'time', chunk_time_interval => INTERVAL '1 month');

-- +goose Down
DROP TABLE IF EXISTS ticker_dividend_analytics;
DROP TABLE IF EXISTS ticker_dividend_summaries;
DROP TABLE IF EXISTS ticker_dividends;
