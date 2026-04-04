# Daily Market Data Pipeline

## Overview

Daily pipeline that ingests OHLCV data for all US equities (~5,000-8,000 tickers) from Polygon.io, enriches each ticker with fundamentals, news, sentiment, related companies, and technical indicators, then computes daily and rolling statistics. Runs after market close via EventBridge.

**Architecture: SQS + Step Functions hybrid.** SQS handles fan-out (no S3 dependency), and each SQS message triggers a child Step Function with 3 sequential stages per ticker for granular retry and observability.

## Architecture Diagram

```
EventBridge (daily 4:30pm ET)
    ↓
Lambda: FetchTickers
  → Polygon /v3/reference/tickers (paginated, all active US equities)
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
    → Polygon /v2/aggs/ticker/{ticker}/range/1/day/{date}/{date}
    → Upsert daily_prices in TimescaleDB
    ↓
  Stage B — Lambda: EnrichTicker (sequential sub-steps)
    → Polygon /v3/reference/tickers/{ticker}     (fundamentals)
    → Polygon /v2/reference/news?ticker={ticker}  (news/sentiment)
    → Polygon /v1/related-companies/{ticker}       (related tickers)
    → Compute SMA, EMA, RSI, MACD, Bollinger from historical OHLCV in DB
    → Upsert to ticker_fundamentals, ticker_news, related_tickers, ticker_technicals
    ↓
  Stage C — Lambda: ComputeStats
    → Read daily + historical prices from DB
    → Compute daily metrics (% change, volume change, range, VWAP)
    → Compute rolling stats (7d/30d/90d returns, volatility, avg volume, drawdown)
    → Upsert ticker_stats

Failed messages (after SFN retries exhaust) → DLQ for inspection
```

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Fan-out mechanism | SQS | No S3 bucket needed, natural backpressure, simple concurrency control via Lambda MaxConcurrency |
| Per-ticker orchestration | Child Step Function | 3 Task states for granular per-stage retry and visual debugging in the console |
| Concurrency | SQS MaxConcurrency=200 | Controls parallel child SFN executions; Polygon Business plan has no rate limit |
| SQS visibility timeout | 15 minutes | Long enough for child SFN to complete all 3 stages |
| Failure handling | DLQ after 3 SQS retries | Failed tickers land in DLQ for manual inspection/retry |
| Stage B structure | Sequential within one Lambda | Simpler than parallel sub-steps; sufficient with unlimited Polygon plan |
| DB writes | `ON CONFLICT DO UPDATE` everywhere | All stages are idempotent — safe to retry at any point |
| Price precision | NUMERIC(18,6) | Sub-penny precision for all equities |
| Chunk interval | 1 month | 8K rows/day too small for daily chunks; monthly is optimal for compression |

## How SQS + Step Functions Interact

1. `FetchTickers` Lambda sends `{ticker, date}` messages to SQS
2. SQS triggers `StartTickerPipeline` Lambda (batch size=1, MaxConcurrency=200)
3. `StartTickerPipeline` calls `sfn.StartExecution()` with the ticker event as input
4. The child SFN runs Stages A→B→C with per-stage retry
5. `StartTickerPipeline` returns success immediately (fire-and-forget) — SQS deletes the message
6. If `StartExecution` fails, the Lambda returns an error, SQS retries delivery

## Pipeline Stages

### Stage A: Ingest OHLCV

Fetches one day of OHLCV data for a single ticker from Polygon and writes it to TimescaleDB.

- **Polygon endpoint:** `GET /v2/aggs/ticker/{ticker}/range/1/day/{date}/{date}`
- **DB table:** `daily_prices` (hypertable)
- **Fields:** ticker, time, open, high, low, close, volume, vwap, num_trades

### Stage B: Enrich Ticker

Fetches company fundamentals, news, and related companies from Polygon, then computes technical indicators from historical OHLCV data already in the database.

**Polygon API calls (sequential):**
1. `GET /v3/reference/tickers/{ticker}` → company fundamentals (market cap, sector, industry, SIC, employees)
2. `GET /v2/reference/news?ticker={ticker}&limit=10` → recent news with sentiment
3. `GET /v1/related-companies/{ticker}` → related tickers

**Computed technical indicators** (from last 200 days of historical OHLCV in DB):
- SMA (20, 50, 200-day)
- EMA (12, 26-day)
- RSI (14-period)
- MACD (line, signal, histogram)
- Bollinger Bands (upper, middle, lower, bandwidth)

**DB tables written:** `ticker_fundamentals`, `ticker_news`, `related_tickers`, `ticker_technicals`

### Stage C: Compute Stats

Reads the daily price just written (plus historical) from the database and computes daily and rolling window statistics.

**Daily metrics:**
- Percentage change
- Volume change
- Intraday range (high - low)
- VWAP

**Rolling window stats (7d, 30d, 90d):**
- Returns
- Volatility (standard deviation of returns)
- Average volume
- Maximum drawdown

**DB table written:** `ticker_stats`

## Go Package Layout

```
cmd/
  lambda-fetch-tickers/       # Step 1: Fetch all tickers → SQS
    main.go
  lambda-start-pipeline/      # SQS consumer: start child SFN per ticker
    main.go
  lambda-ingest-ohlcv/        # Stage A: OHLCV → DB
    main.go
  lambda-enrich-ticker/       # Stage B: Fundamentals + news + related + technicals → DB
    main.go
  lambda-compute-stats/       # Stage C: Daily + rolling stats → DB
    main.go

internal/
  domain/                     # Pure domain model types (no DB, no API deps)
    ticker.go                 #   Ticker, TickerDetail
    ohlcv.go                  #   DailyPrice
    fundamentals.go           #   TickerFundamentals
    news.go                   #   TickerNews
    technicals.go             #   TechnicalIndicators
    stats.go                  #   TickerStats
    related.go                #   RelatedTicker

  polygon/                    # Polygon.io REST API client
    client.go                 #   HTTP client with auth, rate limiting, retry
    tickers.go                #   FetchActiveTickers (paginated)
    aggregates.go             #   FetchDailyOHLCV
    fundamentals.go           #   FetchTickerDetails
    news.go                   #   FetchTickerNews
    related.go                #   FetchRelatedCompanies

  repository/                 # Database access layer (pgx raw SQL)
    daily_price.go            #   Upsert, GetHistorical
    ticker.go                 #   UpsertBatch, GetActive
    fundamentals.go           #   Upsert
    news.go                   #   UpsertBatch
    technicals.go             #   Upsert
    stats.go                  #   Upsert
    related.go                #   UpsertBatch

  indicator/                  # Technical indicator computation (pure math)
    sma.go                    #   Simple Moving Average
    ema.go                    #   Exponential Moving Average
    rsi.go                    #   Relative Strength Index
    macd.go                   #   MACD
    bollinger.go              #   Bollinger Bands

  stats/                      # Stats computation logic
    daily.go                  #   ComputeDaily
    rolling.go                #   ComputeRolling

  pipeline/                   # Shared pipeline types and utilities
    event.go                  #   TickerEvent, FetchTickersOutput
    errors.go                 #   RateLimitError, PolygonAPIError, DatabaseError
    sqs.go                    #   SendTickerMessages (batch send)
```

## Database Schema

### Tables

| Table | Primary Key | Type | Description |
|-------|-------------|------|-------------|
| `tickers` | `(ticker)` | Regular | Reference data: symbol, name, market, exchange, type, active |
| `daily_prices` | `(ticker, time)` | Hypertable | OHLCV candles, NUMERIC(18,6) precision |
| `ticker_fundamentals` | `(ticker, time)` | Hypertable | Market cap, sector, industry, SIC, employees, JSONB for address/branding |
| `ticker_news` | `(id, time)` | Hypertable | News articles with sentiment score, indexed on ticker |
| `ticker_technicals` | `(ticker, time)` | Hypertable | SMA, EMA, RSI, MACD, Bollinger values |
| `ticker_stats` | `(ticker, time)` | Hypertable | Daily metrics + 7d/30d/90d rolling window stats |
| `related_tickers` | `(ticker, related_ticker, time)` | Hypertable | Related company relationships |

**All hypertables:** 1-month chunk interval, compression enabled after 30 days (segmented by ticker, ordered by time DESC).

## Error Handling

### Per-Stage Retry (Step Functions)

Each Task state in the child Step Function has retry configuration:

| Error Type | Max Attempts | Initial Interval | Backoff Rate |
|-----------|-------------|-------------------|-------------|
| `RateLimitError` | 5 | 15s | 2.5x |
| `PolygonAPIError` | 3 | 5s | 2.0x |
| `DatabaseError` | 3 | 3s | 2.0x |

After all retries exhaust, the stage routes to a `RecordFailure` catch state that logs `{ticker, stage, error, timestamp}`.

### SQS Retry

If `sfn.StartExecution()` fails in the `StartTickerPipeline` Lambda, the Lambda returns an error. SQS retries delivery up to 3 times (`maxReceiveCount=3`). After 3 failures, the message moves to the DLQ.

### Idempotency

All database writes use `INSERT ... ON CONFLICT DO UPDATE`. Any stage can be safely retried without creating duplicate data.

### Partial Enrichment

Stage B commits each sub-step independently (fundamentals, news, related, technicals are separate upserts). On retry, already-written data is overwritten idempotently — only the failed sub-step needs to succeed.

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL/TimescaleDB connection string |
| `POLYGON_API_KEY` | Yes | Polygon.io API key (Business plan) |
| `SQS_QUEUE_URL` | Yes | SQS queue URL for ticker fan-out |
| `SFN_ARN` | Yes | Child Step Function ARN for per-ticker pipeline |

## Dependencies

```
github.com/aws/aws-sdk-go-v2
github.com/aws/aws-sdk-go-v2/config
github.com/aws/aws-sdk-go-v2/service/sqs
github.com/aws/aws-sdk-go-v2/service/sfn
golang.org/x/time                        # rate.Limiter for Polygon client
```

## Build

```bash
make build-lambdas    # Builds all Lambda binaries (linux/arm64 for Graviton2)
```

Each Lambda produces a `bootstrap` binary in `bin/lambda-<name>/bootstrap`.
