package massive

import (
	"context"
	"fmt"
	"time"

	"github.com/massive-com/client-go/v2/rest/models"

	"github.com/profitify/profitify-backend/internal/domain"
)

// FetchDividends fetches all dividends for a ticker from Massive.
// Uses the "collect then retry" pattern: iterates all pages, and if a
// retryable error occurs mid-pagination, retries the entire operation.
func (c *Client) FetchDividends(ctx context.Context, ticker string) ([]domain.TickerDividend, error) {
	var dividends []domain.TickerDividend

	c.logger.Info("fetching dividends from Massive", "ticker", ticker)

	err := c.retry("FetchDividends", func() error {
		dividends = nil // reset on retry

		params := models.ListDividendsParams{}.
			WithTicker(models.EQ, ticker).
			WithOrder(models.Desc).
			WithLimit(1000)

		iter := c.sdk.ListDividends(ctx, params)
		for iter.Next() {
			dividends = append(dividends, mapDividend(iter.Item()))
		}
		if iter.Err() != nil {
			return fmt.Errorf("iterating dividends: %w", iter.Err())
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("massive.FetchDividends: %w", err)
	}

	c.logger.Info("fetched dividends", "ticker", ticker, "count", len(dividends))
	return dividends, nil
}

func mapDividend(d models.Dividend) domain.TickerDividend {
	return domain.TickerDividend{
		CashAmount:       d.CashAmount,
		ExDividendDate:   d.ExDividendDate,
		DeclarationDate:  formatDate(time.Time(d.DeclarationDate)),
		RecordDate:       formatDate(time.Time(d.RecordDate)),
		PayDate:          formatDate(time.Time(d.PayDate)),
		Frequency:        int(d.Frequency),
		DistributionType: d.DividendType,
	}
}
