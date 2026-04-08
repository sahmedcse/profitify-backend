-- seed_dashboard_data.sql
--
-- Populates a small handful of tickers with enough materialized data to
-- exercise every dashboard endpoint locally without running the full
-- Massive ingestion + compute pipeline. Idempotent: rerun freely.
--
-- Usage:
--   psql "$DATABASE_URL" -f scripts/seed_dashboard_data.sql

BEGIN;

-- ---------- 1. tickers ----------
INSERT INTO tickers (ticker, name, market, primary_exchange, type, active, currency_name, locale, sector)
VALUES
    ('AAPL', 'Apple Inc.',         'stocks', 'XNAS', 'CS', TRUE, 'usd', 'us', 'Technology'),
    ('MSFT', 'Microsoft Corp.',    'stocks', 'XNAS', 'CS', TRUE, 'usd', 'us', 'Technology'),
    ('JPM',  'JPMorgan Chase & Co.', 'stocks', 'XNYS', 'CS', TRUE, 'usd', 'us', 'Financial Services'),
    ('XOM',  'Exxon Mobil Corp.',  'stocks', 'XNYS', 'CS', TRUE, 'usd', 'us', 'Energy'),
    ('JNJ',  'Johnson & Johnson',  'stocks', 'XNYS', 'CS', TRUE, 'usd', 'us', 'Healthcare')
ON CONFLICT (ticker) DO UPDATE SET
    name = EXCLUDED.name,
    sector = EXCLUDED.sector,
    active = TRUE;

-- ---------- 2. daily_prices: 30 days per ticker ----------
-- Generates a synthetic random walk anchored to a base price so charts
-- look believable. Uses a deterministic seed per ticker via hashtext().
INSERT INTO daily_prices (ticker_id, time, open, high, low, close, volume)
SELECT
    t.id,
    d.day,
    base + (random() * 2 - 1),
    base + 1.5 + random(),
    base - 1.5 - random(),
    base + (random() * 2 - 1),
    1000000 + (random() * 500000)::INT
FROM (
    SELECT id, ticker,
        CASE ticker
            WHEN 'AAPL' THEN 175.0
            WHEN 'MSFT' THEN 340.0
            WHEN 'JPM'  THEN 155.0
            WHEN 'XOM'  THEN 110.0
            WHEN 'JNJ'  THEN 158.0
        END AS base
    FROM tickers
    WHERE ticker IN ('AAPL','MSFT','JPM','XOM','JNJ')
) t
CROSS JOIN LATERAL (
    SELECT generate_series(
        date_trunc('day', NOW() - INTERVAL '30 days'),
        date_trunc('day', NOW()),
        INTERVAL '1 day'
    ) AS day
) d
ON CONFLICT (ticker_id, time) DO NOTHING;

-- ---------- 3. ticker_technicals: latest snapshot per ticker ----------
INSERT INTO ticker_technicals (
    ticker_id, time,
    sma_20, sma_50, ema_12,
    rsi_14, macd_line, macd_signal, macd_histogram,
    bollinger_upper, bollinger_middle, bollinger_lower,
    indicator_statuses
)
SELECT
    t.id,
    NOW(),
    base * 0.98, base * 0.96, base * 0.99,
    rsi, macd, macd_sig, macd - macd_sig,
    base * 1.02, base, base * 0.98,
    statuses::jsonb
FROM (
    VALUES
        ('AAPL', 175.0, 62.0, 1.20,  0.80, '{"rsi_14":"neutral","macd":"bullish","sma_20":"bullish","sma_50":"bullish","ema_12":"bullish","bollinger":"neutral"}'),
        ('MSFT', 340.0, 71.5, 2.10,  1.30, '{"rsi_14":"bullish","macd":"bullish","sma_20":"bullish","sma_50":"bullish","ema_12":"bullish","bollinger":"neutral"}'),
        ('JPM',  155.0, 48.0, -0.30, 0.10, '{"rsi_14":"neutral","macd":"bearish","sma_20":"neutral","sma_50":"neutral","ema_12":"neutral","bollinger":"neutral"}'),
        ('XOM',  110.0, 28.0, -0.90, -0.40,'{"rsi_14":"bearish","macd":"bearish","sma_20":"bearish","sma_50":"bearish","ema_12":"bearish","bollinger":"neutral"}'),
        ('JNJ',  158.0, 55.0, 0.40,  0.20, '{"rsi_14":"neutral","macd":"bullish","sma_20":"neutral","sma_50":"neutral","ema_12":"neutral","bollinger":"neutral"}')
) AS s(symbol, base, rsi, macd, macd_sig, statuses)
JOIN tickers t ON t.ticker = s.symbol
ON CONFLICT (ticker_id, time) DO NOTHING;

-- ---------- 4. ticker_stats: latest snapshot per ticker ----------
INSERT INTO ticker_stats (
    ticker_id, time,
    price_change, price_change_pct, day_range, gap_pct, relative_volume,
    high_52w, low_52w, avg_volume_30d,
    signal_label, signal_strength,
    pivot_levels
)
SELECT
    t.id,
    NOW(),
    chg, chg_pct, range_, gap, rvol,
    hi52, lo52, avgvol,
    label, strength,
    pivots::jsonb
FROM (
    VALUES
        ('AAPL', 1.85, 1.06, 3.20, 0.30, 1.15, 199.6, 124.2, 55_000_000, 'Bullish',     78,
         '{"R3":{"price":182.0,"strength":"weak"},"R2":{"price":179.5,"strength":"moderate"},"R1":{"price":177.0,"strength":"strong"},"S1":{"price":172.5,"strength":"strong"},"S2":{"price":170.0,"strength":"moderate"},"S3":{"price":167.5,"strength":"weak"}}'),
        ('MSFT', 4.20, 1.24, 5.10, 0.40, 1.20, 384.3, 245.6, 24_000_000, 'Strong Buy',  88,
         '{"R3":{"price":352.0,"strength":"weak"},"R2":{"price":348.0,"strength":"moderate"},"R1":{"price":344.0,"strength":"strong"},"S1":{"price":336.0,"strength":"strong"},"S2":{"price":332.0,"strength":"moderate"},"S3":{"price":328.0,"strength":"weak"}}'),
        ('JPM',  -0.40,-0.26, 1.80, -0.10, 0.95, 172.9, 135.1, 9_500_000, 'Neutral',     50,
         '{"R3":{"price":159.0,"strength":"weak"},"R2":{"price":157.5,"strength":"moderate"},"R1":{"price":156.0,"strength":"weak"},"S1":{"price":153.5,"strength":"weak"},"S2":{"price":152.0,"strength":"moderate"},"S3":{"price":150.5,"strength":"weak"}}'),
        ('XOM',  -1.30,-1.18, 2.40, -0.50, 1.05, 120.7,  95.0, 17_000_000, 'Bearish',     22,
         '{"R3":{"price":113.0,"strength":"weak"},"R2":{"price":111.5,"strength":"weak"},"R1":{"price":110.0,"strength":"moderate"},"S1":{"price":107.5,"strength":"strong"},"S2":{"price":105.5,"strength":"moderate"},"S3":{"price":103.0,"strength":"weak"}}'),
        ('JNJ',  0.10, 0.06, 1.20, 0.05, 0.90, 175.9, 142.0, 7_200_000, 'Neutral',     55,
         '{"R3":{"price":162.0,"strength":"weak"},"R2":{"price":160.5,"strength":"moderate"},"R1":{"price":159.0,"strength":"weak"},"S1":{"price":156.5,"strength":"weak"},"S2":{"price":155.0,"strength":"moderate"},"S3":{"price":153.5,"strength":"weak"}}')
) AS s(symbol, chg, chg_pct, range_, gap, rvol, hi52, lo52, avgvol, label, strength, pivots)
JOIN tickers t ON t.ticker = s.symbol
ON CONFLICT (ticker_id, time) DO NOTHING;

-- ---------- 5. ticker_fundamentals: latest snapshot per ticker ----------
INSERT INTO ticker_fundamentals (
    ticker_id, time, market_cap, shares_outstanding, sic_code, sic_description, description, homepage_url, total_employees
)
SELECT t.id, NOW(), market_cap, shares, sic, sic_desc, descr, url, employees
FROM (
    VALUES
        ('AAPL', 2_900_000_000_000.0::float8, 15_500_000_000::bigint, '3674', 'Semiconductors',                 'Apple designs consumer electronics and software.', 'https://apple.com',  164000),
        ('MSFT', 2_700_000_000_000.0,         7_400_000_000::bigint, '7372', 'Prepackaged Software',           'Microsoft makes software, cloud and devices.',     'https://microsoft.com', 221000),
        ('JPM',    460_000_000_000.0,         2_900_000_000::bigint, '6020', 'State Commercial Banks',         'JPMorgan Chase is a global financial services firm.', 'https://jpmorganchase.com', 309000),
        ('XOM',    430_000_000_000.0,         4_000_000_000::bigint, '2911', 'Petroleum Refining',             'Exxon Mobil is an international oil and gas company.', 'https://exxonmobil.com', 62000),
        ('JNJ',    400_000_000_000.0,         2_400_000_000::bigint, '2834', 'Pharmaceutical Preparations',    'Johnson & Johnson is a global healthcare company.', 'https://jnj.com', 130000)
) AS s(symbol, market_cap, shares, sic, sic_desc, descr, url, employees)
JOIN tickers t ON t.ticker = s.symbol
ON CONFLICT (ticker_id, time) DO NOTHING;

-- ---------- 6. ticker_dividend_summaries: latest snapshot per ticker ----------
INSERT INTO ticker_dividend_summaries (
    ticker_id, time, current_yield, forward_yield, trailing_yield_12m, payout_frequency
)
SELECT t.id, NOW(), cur, fwd, ttm, freq
FROM (
    VALUES
        ('AAPL', 0.55, 0.58, 0.55, 4),
        ('MSFT', 0.78, 0.82, 0.78, 4),
        ('JPM',  2.45, 2.55, 2.45, 4),
        ('XOM',  3.30, 3.45, 3.30, 4),
        ('JNJ',  3.05, 3.15, 3.05, 4)
) AS s(symbol, cur, fwd, ttm, freq)
JOIN tickers t ON t.ticker = s.symbol
ON CONFLICT (ticker_id, time) DO NOTHING;

COMMIT;
