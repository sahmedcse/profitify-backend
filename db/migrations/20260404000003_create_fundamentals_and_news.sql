-- +goose Up

CREATE TABLE ticker_fundamentals (
    id                          UUID DEFAULT gen_random_uuid(),
    ticker_id                   UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    time                        TIMESTAMPTZ NOT NULL,
    market_cap                  DOUBLE PRECISION,
    shares_outstanding          BIGINT,
    weighted_shares_outstanding BIGINT,
    sic_code                    TEXT NOT NULL DEFAULT '',
    sic_description             TEXT NOT NULL DEFAULT '',
    description                 TEXT NOT NULL DEFAULT '',
    homepage_url                TEXT NOT NULL DEFAULT '',
    phone_number                TEXT NOT NULL DEFAULT '',
    total_employees             INT,
    address                     JSONB NOT NULL DEFAULT '{}',
    branding                    JSONB NOT NULL DEFAULT '{}',
    UNIQUE (ticker_id, time)
);

SELECT create_hypertable('ticker_fundamentals', 'time', chunk_time_interval => INTERVAL '1 month');

CREATE TABLE ticker_news (
    id             TEXT PRIMARY KEY,
    title          TEXT NOT NULL DEFAULT '',
    description    TEXT NOT NULL DEFAULT '',
    author         TEXT NOT NULL DEFAULT '',
    article_url    TEXT NOT NULL DEFAULT '',
    image_url      TEXT NOT NULL DEFAULT '',
    publisher_name TEXT NOT NULL DEFAULT '',
    keywords       TEXT[] NOT NULL DEFAULT '{}',
    tickers        TEXT[] NOT NULL DEFAULT '{}',
    published_utc  TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_ticker_news_published_utc ON ticker_news (published_utc DESC);
CREATE INDEX idx_ticker_news_tickers ON ticker_news USING GIN (tickers);

CREATE TABLE ticker_news_insights (
    id                  UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    news_id             TEXT NOT NULL REFERENCES ticker_news(id) ON DELETE CASCADE,
    ticker_id           UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    sentiment           TEXT NOT NULL DEFAULT '',
    sentiment_reasoning TEXT NOT NULL DEFAULT '',
    UNIQUE (news_id, ticker_id)
);

CREATE INDEX idx_ticker_news_insights_ticker_id ON ticker_news_insights (ticker_id);

CREATE TABLE ticker_news_summaries (
    id             UUID DEFAULT gen_random_uuid(),
    ticker_id      UUID NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    time           TIMESTAMPTZ NOT NULL,
    article_count  INT NOT NULL DEFAULT 0,
    avg_sentiment  DOUBLE PRECISION NOT NULL DEFAULT 0,
    positive_count INT NOT NULL DEFAULT 0,
    negative_count INT NOT NULL DEFAULT 0,
    neutral_count  INT NOT NULL DEFAULT 0,
    UNIQUE (ticker_id, time)
);

SELECT create_hypertable('ticker_news_summaries', 'time', chunk_time_interval => INTERVAL '1 month');

-- +goose Down
DROP TABLE IF EXISTS ticker_news_summaries;
DROP TABLE IF EXISTS ticker_news_insights;
DROP TABLE IF EXISTS ticker_news;
DROP TABLE IF EXISTS ticker_fundamentals;
