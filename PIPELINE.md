# Daily Market Data Pipeline

## Overview

Daily pipeline that ingests OHLCV data for all US equities (~5,000-8,000 tickers) from Massive, enriches each ticker with fundamentals, dividends, news/sentiment, related companies, and technical indicators, then computes daily and rolling statistics. Runs after market close via EventBridge.

**Architecture: SQS + Step Functions hybrid.** SQS handles fan-out (no S3 dependency), and each SQS message triggers a child Step Function with 3 sequential stages per ticker for granular retry and observability.

## Architecture Diagram

```
EventBridge (daily 4:30pm ET)
    ↓
Lambda: FetchTickers
  → Massive /v3/reference/tickers (paginated, all active US equities)
  → Sends ~8,000 SQS messages (one per ticker, includes date)
  → Upserts tickers reference table
    ↓
SQS Queue
  → Triggers Lambda: StartTickerPipeline (MaxConcurrency=200)
      → Starts a child Step Function execution per ticker
    ↓
Child Step Function (per ticker, 3 sequential stages):
    ↓
  Stage A — Lambda: IngestOHLCV
    → Massive /v1/open-close/{ticker}/{date}
    → Upsert daily_prices in TimescaleDB (OHLCV + pre-market + after-hours)
    ↓
  Stage B — Lambda: EnrichTicker (sequential sub-steps)
    → Massive /v3/reference/tickers/{ticker}       (fundamentals)
    → Massive /v3/reference/dividends?ticker=       (dividends)
    → Massive /v2/reference/news?ticker={ticker}    (news + per-ticker sentiment)
    → Massive /v1/related-companies/{ticker}         (related tickers)
    → Massive /v1/indicators/sma/{ticker}            (SMA 20/50/200)
    → Massive /v1/indicators/ema/{ticker}            (EMA 12/26)
    → Massive /v1/indicators/rsi/{ticker}            (RSI 14)
    → Massive /v1/indicators/macd/{ticker}           (MACD line/signal/histogram)
    → Self-compute Bollinger Bands, ATR(14), OBV from OHLCV in DB
    → Upsert to 7 tables (see Database Schema)
    ↓
  Stage C — Lambda: ComputeStats
    → Read daily + historical prices + dividends from DB
    → Compute daily metrics (price change, volume change, gap %, day range, relative volume)
    → Compute rolling stats (7d/30d/90d price return, dividend return, total return, volatility, avg volume, max drawdown)
    → Compute 52-week high/low + distance percentages
    → Compute dividend summary (yield, growth, streak) + analytics (sector comparison, income quality)
    → Compute inter-ticker relations (correlations, lead/lag, volume profiles, volatility clusters)
    → Upsert to 7 tables (see Database Schema)

Failed messages (after SFN retries exhaust) → DLQ for inspection
```

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Fan-out mechanism | SQS | No S3 bucket needed, natural backpressure, simple concurrency control via Lambda MaxConcurrency |
| Per-ticker orchestration | Child Step Function | 3 Task states for granular per-stage retry and visual debugging in the console |
| Concurrency | SQS MaxConcurrency=200 | Controls parallel child SFN executions; Massive Business plan has no rate limit |
| SQS visibility timeout | 15 minutes | Long enough for child SFN to complete all 3 stages |
| Failure handling | DLQ after 3 SQS retries | Failed tickers land in DLQ for manual inspection/retry |
| Stage B structure | Sequential within one Lambda | Simpler than parallel sub-steps; sufficient with unlimited Massive plan |
| DB writes | `ON CONFLICT DO UPDATE` everywhere | All stages are idempotent — safe to retry at any point |
| Price precision | NUMERIC(18,6) | Sub-penny precision for all equities |
| Chunk interval | 1 month | 8K rows/day too small for daily chunks; monthly is optimal for compression |
| Primary keys | UUID (`gen_random_uuid()`) | All tables use UUID PK; ticker symbol is a UNIQUE column; child tables reference via `ticker_id` FK |
| Technical indicators | Hybrid (Massive + self-computed) | SMA/EMA/RSI/MACD from Massive (split-adjusted); Bollinger/ATR/OBV computed from OHLCV data |
| Config | Per-Lambda config structs | Each Lambda loads only the env vars it needs; fail-fast on missing required vars |
| Trade data | Skipped | Individual trades = ~400M rows/day at ~$10K+/year storage. Daily aggregates sufficient for current needs. |

## How SQS + Step Functions Interact

1. `FetchTickers` Lambda sends `{ticker, date}` messages to SQS
2. SQS triggers `StartTickerPipeline` Lambda (batch size=1, MaxConcurrency=200)
3. `StartTickerPipeline` calls `sfn.StartExecution()` with the ticker event as input
4. The child SFN runs Stages A→B→C with per-stage retry
5. `StartTickerPipeline` returns success immediately (fire-and-forget) — SQS deletes the message
6. If `StartExecution` fails, the Lambda returns an error, SQS retries delivery

## Pipeline Stages

### Stage A: Ingest OHLCV

Fetches one day of OHLCV data for a single ticker from Massive and writes it to TimescaleDB.

- **Massive endpoint:** `GET /v1/open-close/{ticker}/{date}`
- **DB table:** `daily_prices` (hypertable)
- **Fields:** open, high, low, close, volume, vwap, pre_market, after_hours, otc

### Stage B: Enrich Ticker

Fetches company fundamentals, dividends, news, related companies, and technical indicators from Massive. Computes Bollinger Bands, ATR, and OBV from historical OHLCV data in the database.

**Massive API calls (sequential, ~11 calls per ticker):**
1. `GET /v3/reference/tickers/{ticker}` → fundamentals (market cap, sector, SIC, employees, address, branding)
2. `GET /v3/reference/dividends?ticker={ticker}` → dividend history (amount, ex-date, frequency, distribution type)
3. `GET /v2/reference/news?ticker={ticker}&limit=10` → news articles with per-ticker sentiment insights
4. `GET /v1/related-companies/{ticker}` → related tickers
5. `GET /v1/indicators/sma/{ticker}?window=20` → SMA 20
6. `GET /v1/indicators/sma/{ticker}?window=50` → SMA 50
7. `GET /v1/indicators/sma/{ticker}?window=200` → SMA 200
8. `GET /v1/indicators/ema/{ticker}?window=12` → EMA 12
9. `GET /v1/indicators/ema/{ticker}?window=26` → EMA 26
10. `GET /v1/indicators/rsi/{ticker}?window=14` → RSI 14
11. `GET /v1/indicators/macd/{ticker}` → MACD (line, signal, histogram)

**Self-computed indicators** (from historical OHLCV in DB):
- Bollinger Bands (20-period, 2 std dev) — upper, middle, lower
- ATR (14-period) — Average True Range
- OBV — On-Balance Volume

**DB tables written:** `ticker_fundamentals`, `ticker_dividends`, `ticker_news` + `ticker_news_insights`, `related_tickers`, `ticker_technicals`

### Stage C: Compute Stats

Reads daily price, historical prices, and dividend data from the database and computes daily metrics, rolling window statistics, dividend analytics, and inter-ticker relationships.

**Daily metrics:**
- Price change ($ and %)
- Volume change (%)
- Gap % (today's open vs yesterday's close)
- Day range (high - low)
- Relative volume (today's volume / 30d avg)

**Rolling window stats (7d, 30d, 90d):**
- Price return (price-only % change)
- Dividend return (income-only % change)
- Total return (price + dividends combined)
- Volatility (annualized std dev of daily returns)
- Average volume
- Maximum drawdown

**52-week extremes:**
- 52-week high and low
- Distance from 52w high/low as %

**Dividend analytics:**
- Dividend summary: current/forward/trailing yield, growth rates (1y/3y/5y), consecutive increase streak, next ex-date
- Dividend analytics: yield vs sector average, income quality score, ex-dividend price impact

**Inter-ticker relations (computed daily):**
- Correlations: Pearson correlation of daily returns (30d and 90d windows) with rank
- Lead/lag: predictive relationships where one ticker's move predicts another's
- Volume profiles: volume pattern similarity scores with rank
- Volatility clusters: grouping tickers by similar 30d volatility bands

**DB tables written:** `ticker_stats`, `ticker_dividend_summaries`, `ticker_dividend_analytics`, `ticker_news_summaries`, `ticker_correlations`, `ticker_lead_lag`, `ticker_volume_profiles`, `ticker_volatility_clusters`

## Go Package Layout

```
cmd/
  lambda-fetch-tickers/       # Step 1: Fetch all tickers → SQS
    main.go
  lambda-start-pipeline/      # SQS consumer: start child SFN per ticker
    main.go
  lambda-ingest-ohlcv/        # Stage A: OHLCV → DB
    main.go
  lambda-enrich-ticker/       # Stage B: Fundamentals + dividends + news + related + technicals → DB
    main.go
  lambda-compute-stats/       # Stage C: Stats + dividend analytics + relations → DB
    main.go

internal/
  config/
    config.go                 #   Config (API), FetchTickersConfig, StartPipelineConfig,
                              #   IngestOHLCVConfig, EnrichTickerConfig, ComputeStatsConfig

  domain/                     # Pure domain model types (no DB, no API deps)
    ticker.go                 #   Ticker (identity: symbol, name, market, exchange, CIK)
    ticker_fundamentals.go    #   TickerFundamentals, Address, Branding
    daily_price.go            #   DailyPrice (OHLCV + pre-market + after-hours)
    ticker_snapshot.go        #   TickerSnapshot, Bar, MinuteBar, LastTrade, LastQuote
    ticker_technicals.go      #   TechnicalIndicators (Massive: SMA/EMA/RSI/MACD; self: Bollinger/ATR/OBV)
    ticker_stats.go           #   TickerStats (daily metrics + rolling price/dividend/total return + 52w)
    ticker_news.go            #   TickerNews (article with keywords, publisher, mentioned tickers)
    ticker_news_insight.go    #   TickerNewsInsight (per-ticker sentiment + reasoning)
    ticker_news_summary.go    #   TickerNewsSummary (daily aggregate: article count, avg sentiment)
    ticker_dividend.go        #   TickerDividend (raw Massive: amount, ex-date, frequency, type)
    ticker_dividend_summary.go    #   TickerDividendSummary (yield, growth rate, streak)
    ticker_dividend_analytics.go  #   TickerDividendAnalytics (sector comparison, income quality)
    related_ticker.go         #   RelatedTicker (Massive-sourced)
    ticker_correlation.go     #   TickerCorrelation (30d/90d return correlation with rank)
    ticker_lead_lag.go        #   TickerLeadLag (predictive score with rank)
    ticker_volume_profile.go  #   TickerVolumeProfile (similarity score with rank)
    ticker_volatility_cluster.go  #   TickerVolatilityCluster (cluster ID, volatility, rank)

  massive/                    # Massive REST API client
    client.go                 #   HTTP client with auth, rate limiting, retry on 429
    tickers.go                #   FetchActiveTickers (paginated cursor)
    aggregates.go             #   FetchDailyOHLCV
    fundamentals.go           #   FetchTickerDetails
    dividends.go              #   FetchDividends
    news.go                   #   FetchTickerNews (articles + insights)
    related.go                #   FetchRelatedCompanies
    indicators.go             #   FetchSMA, FetchEMA, FetchRSI, FetchMACD

  repository/                 # Database access layer (pgx raw SQL, ON CONFLICT DO UPDATE)
    repository.go             #   Interfaces: TickerRepository, DailyPriceRepository, etc.
    ticker.go                 #   UpsertBatch, GetActive, GetBySymbol
    daily_price.go            #   Upsert, GetHistorical, GetLatest
    ticker_fundamentals.go    #   Upsert
    ticker_technicals.go      #   Upsert
    ticker_stats.go           #   Upsert
    ticker_news.go            #   UpsertBatch
    ticker_news_insight.go    #   UpsertBatch
    ticker_news_summary.go    #   Upsert
    ticker_dividend.go        #   UpsertBatch
    ticker_dividend_summary.go    #   Upsert
    ticker_dividend_analytics.go  #   Upsert
    related_ticker.go         #   UpsertBatch
    ticker_correlation.go     #   UpsertBatch
    ticker_lead_lag.go        #   UpsertBatch
    ticker_volume_profile.go  #   UpsertBatch
    ticker_volatility_cluster.go  #   UpsertBatch

  indicator/                  # Self-computed technical indicators (pure math)
    bollinger.go              #   BollingerBands (20-period, 2 std dev)
    atr.go                    #   ATR (14-period Average True Range)
    obv.go                    #   OBV (On-Balance Volume)

  stats/                      # Stats computation logic (pure functions)
    daily.go                  #   ComputeDaily (price change, volume change, gap, range, relative vol)
    rolling.go                #   ComputeRolling (price/dividend/total return, volatility, drawdown)
    dividend.go               #   ComputeDividendSummary, ComputeDividendAnalytics
    relations.go              #   ComputeCorrelations, ComputeLeadLag, ComputeVolumeProfiles, ComputeVolatilityClusters

  pipeline/                   # Shared pipeline types and utilities
    event.go                  #   TickerEvent, FetchTickersOutput
    errors.go                 #   RateLimitError, MassiveAPIError, DatabaseError
    sqs.go                    #   SendTickerMessages (batch send)
```

## Database Schema

All tables use UUID primary keys with `gen_random_uuid()`. The `tickers` table has a UNIQUE constraint on the `ticker` column. All child tables reference `tickers(id)` via a `ticker_id UUID` foreign key.

### Tables

| Table | Primary Key | Type | Description |
|-------|-------------|------|-------------|
| `tickers` | `id UUID` | Regular | Reference data: symbol (UNIQUE), name, market, exchange, type, active, CIK, list/delist dates |
| `daily_prices` | `(id, time)` | Hypertable | OHLCV + VWAP + pre-market + after-hours, NUMERIC(18,6) precision |
| `ticker_fundamentals` | `(id, time)` | Hypertable | Market cap, sector, SIC, employees, JSONB for address/branding |
| `ticker_technicals` | `(id, time)` | Hypertable | SMA(20/50/200), EMA(12/26), RSI(14), MACD, Bollinger, ATR(14), OBV |
| `ticker_stats` | `(id, time)` | Hypertable | Daily metrics + 7d/30d/90d price/dividend/total return + 52w high/low |
| `ticker_news` | `(id)` | Regular | News articles with keywords, publisher, mentioned tickers |
| `ticker_news_insights` | `(id)` | Regular | Per-ticker sentiment + reasoning, FK to ticker_news |
| `ticker_news_summaries` | `(id, time)` | Hypertable | Daily aggregate: article count, avg sentiment, positive/negative/neutral counts |
| `ticker_dividends` | `(id)` | Regular | Raw dividend events: amount, ex-date, pay-date, frequency, distribution type |
| `ticker_dividend_summaries` | `(id, time)` | Hypertable | Yield (current/forward/trailing), growth rates, consecutive increase streak |
| `ticker_dividend_analytics` | `(id, time)` | Hypertable | Yield vs sector avg, income quality score, ex-dividend price impact |
| `related_tickers` | `(id)` | Regular | Massive-sourced related company relationships |
| `ticker_correlations` | `(id, time)` | Hypertable | 30d/90d Pearson return correlations with rank |
| `ticker_lead_lag` | `(id, time)` | Hypertable | Lead/lag predictive relationships with rank |
| `ticker_volume_profiles` | `(id, time)` | Hypertable | Volume pattern similarity scores with rank |
| `ticker_volatility_clusters` | `(id, time)` | Hypertable | Volatility band groupings with cluster ID and rank |

**All hypertables:** 1-month chunk interval, compression enabled after 30 days (segmented by ticker_id, ordered by time DESC).

## Error Handling

### Per-Stage Retry (Step Functions)

Each Task state in the child Step Function has retry configuration:

| Error Type | Max Attempts | Initial Interval | Backoff Rate |
|-----------|-------------|-------------------|-------------|
| `RateLimitError` | 5 | 15s | 2.5x |
| `MassiveAPIError` | 3 | 5s | 2.0x |
| `DatabaseError` | 3 | 3s | 2.0x |

After all retries exhaust, the stage routes to a `RecordFailure` catch state that logs `{ticker, stage, error, timestamp}`.

### SQS Retry

If `sfn.StartExecution()` fails in the `StartTickerPipeline` Lambda, the Lambda returns an error. SQS retries delivery up to 3 times (`maxReceiveCount=3`). After 3 failures, the message moves to the DLQ.

### Idempotency

All database writes use `INSERT ... ON CONFLICT DO UPDATE`. Any stage can be safely retried without creating duplicate data.

### Partial Enrichment

Stage B commits each sub-step independently (fundamentals, dividends, news, related, technicals are separate upserts). On retry, already-written data is overwritten idempotently — only the failed sub-step needs to succeed.

## Configuration

Per-Lambda config structs — each Lambda loads only the env vars it needs.

| Variable | Used By | Description |
|----------|---------|-------------|
| `DATABASE_URL` | fetch-tickers, ingest-ohlcv, enrich-ticker, compute-stats | PostgreSQL/TimescaleDB connection string |
| `MASSIVE_API_KEY` | fetch-tickers, ingest-ohlcv, enrich-ticker | Massive API key (Business plan) |
| `SQS_QUEUE_URL` | fetch-tickers | SQS queue URL for ticker fan-out |
| `SFN_ARN` | start-pipeline | Child Step Function ARN for per-ticker pipeline |
| `API_PORT` | api server | HTTP server port (default: 8080) |
| `APP_ENV` | api server | Environment name (default: development) |

## Dependencies

```
github.com/aws/aws-sdk-go-v2
github.com/aws/aws-sdk-go-v2/config
github.com/aws/aws-sdk-go-v2/service/sqs
github.com/aws/aws-sdk-go-v2/service/sfn
golang.org/x/time                        # rate.Limiter for Massive client
```

## Build

```bash
make build-lambdas    # Builds all Lambda binaries (linux/arm64 for Graviton2)
```

Each Lambda produces a `bootstrap` binary in `bin/lambda-<name>/bootstrap`.

## Implementation Plan

Implementation is sliced vertically by Lambda function. Each branch delivers a complete, testable Lambda from repository through to `cmd/` entrypoint.

### Branch 1: `feature/lambda-fetch-tickers`

**Lambdas:** FetchTickers + StartPipeline (fan-out layer)

| Package | Files | Description |
|---------|-------|-------------|
| `massive` | `client.go`, `tickers.go` | Base HTTP client (auth, rate limiter, retry), paginated ticker fetch |
| `repository` | `repository.go`, `ticker.go` | Interface definitions + ticker CRUD (UpsertBatch, GetActive, GetBySymbol) |
| `pipeline` | `event.go`, `errors.go`, `sqs.go` | TickerEvent type, custom error types, SQS batch send |
| `cmd/` | `lambda-fetch-tickers/`, `lambda-start-pipeline/` | Lambda entrypoints |

**New dependencies:** `aws-sdk-go-v2` (sqs, sfn, config), `golang.org/x/time`

### Branch 2: `feature/lambda-ingest-ohlcv`

**Lambda:** IngestOHLCV (Stage A)

| Package | Files | Description |
|---------|-------|-------------|
| `massive` | `aggregates.go` | Fetch daily OHLCV from `/v1/open-close/{ticker}/{date}` |
| `repository` | `daily_price.go` | Upsert, GetHistorical, GetLatest |
| `cmd/` | `lambda-ingest-ohlcv/` | Lambda entrypoint |

### Branch 3: `feature/lambda-enrich-ticker`

**Lambda:** EnrichTicker (Stage B)

| Package | Files | Description |
|---------|-------|-------------|
| `massive` | `fundamentals.go`, `dividends.go`, `news.go`, `related.go`, `indicators.go` | 5 Massive API endpoint clients |
| `repository` | 6 files | Upsert for fundamentals, dividends, news, news_insights, related, technicals |
| `indicator` | `bollinger.go`, `atr.go`, `obv.go` | Pure math functions for self-computed indicators |
| `cmd/` | `lambda-enrich-ticker/` | Lambda entrypoint: 11 Massive calls → self-compute → upsert |

### Branch 4: `feature/lambda-compute-stats`

**Lambda:** ComputeStats (Stage C)

| Package | Files | Description |
|---------|-------|-------------|
| `stats` | `daily.go`, `rolling.go`, `dividend.go`, `relations.go` | Pure computation: daily metrics, rolling returns, dividend analytics, cross-ticker relations |
| `repository` | 8 files | Upsert for stats, news_summaries, dividend_summaries, dividend_analytics, correlations, lead_lag, volume_profiles, volatility_clusters |
| `cmd/` | `lambda-compute-stats/` | Lambda entrypoint: read → compute → upsert |

### Design Patterns

- **Repository interfaces**: Defined in `internal/repository/repository.go`. Each repo has a concrete struct taking `*pgxpool.Pool` via constructor. Interfaces enable unit testing Lambda handlers with mocks.
- **Massive client**: Methods on `*Client` receiver, return domain types directly. API response structs are unexported (private to massive package). Retry on 429 (with Retry-After) and 5xx (exponential backoff).
- **Testing**: Unit tests use `httptest.NewServer` for Massive and mock interfaces for repos. Integration tests run against Docker DB. Table-driven tests everywhere.
