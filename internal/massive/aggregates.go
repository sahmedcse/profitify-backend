package massive

import (
	"context"
	"fmt"
	"time"

	"github.com/massive-com/client-go/v2/rest/models"

	"github.com/profitify/profitify-backend/internal/domain"
)

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
