-- +goose Up

ALTER TABLE daily_prices SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'ticker_id',
    timescaledb.compress_orderby = 'time DESC'
);
SELECT add_compression_policy('daily_prices', INTERVAL '30 days');

ALTER TABLE ticker_technicals SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'ticker_id',
    timescaledb.compress_orderby = 'time DESC'
);
SELECT add_compression_policy('ticker_technicals', INTERVAL '30 days');

ALTER TABLE ticker_stats SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'ticker_id',
    timescaledb.compress_orderby = 'time DESC'
);
SELECT add_compression_policy('ticker_stats', INTERVAL '30 days');

ALTER TABLE ticker_fundamentals SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'ticker_id',
    timescaledb.compress_orderby = 'time DESC'
);
SELECT add_compression_policy('ticker_fundamentals', INTERVAL '30 days');

ALTER TABLE ticker_news_summaries SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'ticker_id',
    timescaledb.compress_orderby = 'time DESC'
);
SELECT add_compression_policy('ticker_news_summaries', INTERVAL '30 days');

ALTER TABLE ticker_dividend_summaries SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'ticker_id',
    timescaledb.compress_orderby = 'time DESC'
);
SELECT add_compression_policy('ticker_dividend_summaries', INTERVAL '30 days');

ALTER TABLE ticker_dividend_analytics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'ticker_id',
    timescaledb.compress_orderby = 'time DESC'
);
SELECT add_compression_policy('ticker_dividend_analytics', INTERVAL '30 days');

ALTER TABLE ticker_correlations SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'ticker_id',
    timescaledb.compress_orderby = 'time DESC'
);
SELECT add_compression_policy('ticker_correlations', INTERVAL '30 days');

ALTER TABLE ticker_lead_lag SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'lead_ticker_id',
    timescaledb.compress_orderby = 'time DESC'
);
SELECT add_compression_policy('ticker_lead_lag', INTERVAL '30 days');

ALTER TABLE ticker_volume_profiles SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'ticker_id',
    timescaledb.compress_orderby = 'time DESC'
);
SELECT add_compression_policy('ticker_volume_profiles', INTERVAL '30 days');

ALTER TABLE ticker_volatility_clusters SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'ticker_id',
    timescaledb.compress_orderby = 'time DESC'
);
SELECT add_compression_policy('ticker_volatility_clusters', INTERVAL '30 days');

-- +goose Down
SELECT remove_compression_policy('ticker_volatility_clusters');
SELECT remove_compression_policy('ticker_volume_profiles');
SELECT remove_compression_policy('ticker_lead_lag');
SELECT remove_compression_policy('ticker_correlations');
SELECT remove_compression_policy('ticker_dividend_analytics');
SELECT remove_compression_policy('ticker_dividend_summaries');
SELECT remove_compression_policy('ticker_news_summaries');
SELECT remove_compression_policy('ticker_fundamentals');
SELECT remove_compression_policy('ticker_stats');
SELECT remove_compression_policy('ticker_technicals');
SELECT remove_compression_policy('daily_prices');

ALTER TABLE daily_prices SET (timescaledb.compress = false);
ALTER TABLE ticker_technicals SET (timescaledb.compress = false);
ALTER TABLE ticker_stats SET (timescaledb.compress = false);
ALTER TABLE ticker_fundamentals SET (timescaledb.compress = false);
ALTER TABLE ticker_news_summaries SET (timescaledb.compress = false);
ALTER TABLE ticker_dividend_summaries SET (timescaledb.compress = false);
ALTER TABLE ticker_dividend_analytics SET (timescaledb.compress = false);
ALTER TABLE ticker_correlations SET (timescaledb.compress = false);
ALTER TABLE ticker_lead_lag SET (timescaledb.compress = false);
ALTER TABLE ticker_volume_profiles SET (timescaledb.compress = false);
ALTER TABLE ticker_volatility_clusters SET (timescaledb.compress = false);
