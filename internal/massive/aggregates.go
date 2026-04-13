package massive

import (
	"context"
	"fmt"
	"time"

	"github.com/massive-com/client-go/v2/rest/models"

	"github.com/profitify/profitify-backend/internal/domain"
)

// FetchDailyBars fetches daily OHLCV bars for a ticker over a date range
// using the aggregates endpoint.
func (c *Client) FetchDailyBars(ctx context.Context, ticker string, from, to time.Time) ([]domain.DailyPrice, error) {
	params := models.ListAggsParams{
		Ticker:     ticker,
		Multiplier: 1,
		Timespan:   "day",
		From:       models.Millis(from),
		To:         models.Millis(to),
	}.WithOrder(models.Asc).WithLimit(50000).WithAdjusted(true)

	iter := c.sdk.ListAggs(ctx, params)

	var prices []domain.DailyPrice
	for iter.Next() {
		agg := iter.Item()
		prices = append(prices, domain.DailyPrice{
			Time:   time.Time(agg.Timestamp),
			Open:   agg.Open,
			High:   agg.High,
			Low:    agg.Low,
			Close:  agg.Close,
			Volume: agg.Volume,
		})
	}
	if iter.Err() != nil {
		return nil, fmt.Errorf("massive.FetchDailyBars(%s): %w", ticker, iter.Err())
	}

	return prices, nil
}

// FetchDailyOHLCV fetches the open, close, afterhours, and pre-market prices
// for a single ticker on a given date.
func (c *Client) FetchDailyOHLCV(ctx context.Context, ticker string, date time.Time) (*domain.DailyPrice, error) {
	var resp *models.GetDailyOpenCloseAggResponse

	err := c.retry("FetchDailyOHLCV", func() error {
		var callErr error
		resp, callErr = c.sdk.GetDailyOpenCloseAgg(ctx, &models.GetDailyOpenCloseAggParams{
			Ticker: ticker,
			Date:   models.Date(date),
		})
		return callErr
	})
	if err != nil {
		return nil, fmt.Errorf("massive.FetchDailyOHLCV(%s, %s): %w", ticker, date.Format("2006-01-02"), err)
	}

	return &domain.DailyPrice{
		Time:       date,
		Open:       resp.Open,
		High:       resp.High,
		Low:        resp.Low,
		Close:      resp.Close,
		Volume:     resp.Volume,
		PreMarket:  resp.PreMarket,
		AfterHours: resp.AfterHours,
		OTC:        resp.OTC,
	}, nil
}
