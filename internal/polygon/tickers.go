package polygon

import (
	"context"
	"fmt"
	"time"

	"github.com/massive-com/client-go/v2/rest/models"

	"github.com/profitify/profitify-backend/internal/domain"
)

// FetchActiveTickers fetches all active US equity tickers from Polygon.
// Uses the "collect then retry" pattern: iterates all pages, and if a
// retryable error occurs mid-pagination, retries the entire operation.
func (c *Client) FetchActiveTickers(ctx context.Context) ([]domain.Ticker, error) {
	var tickers []domain.Ticker

	c.logger.Info("fetching active tickers from Polygon")

	err := c.retry("FetchActiveTickers", func() error {
		tickers = nil // reset on retry

		params := models.ListTickersParams{}.
			WithType("CS").
			WithMarket(models.AssetStocks).
			WithActive(true).
			WithSort(models.TickerSymbol).
			WithOrder(models.Asc).
			WithLimit(1000)

		iter := c.sdk.ListTickers(ctx, params)
		for iter.Next() {
			tickers = append(tickers, mapTicker(iter.Item()))
		}
		if iter.Err() != nil {
			return fmt.Errorf("iterating tickers: %w", iter.Err())
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("polygon.FetchActiveTickers: %w", err)
	}

	c.logger.Info("fetched active tickers", "count", len(tickers))
	return tickers, nil
}

// mapTicker converts the SDK Ticker model to our domain Ticker.
func mapTicker(t models.Ticker) domain.Ticker {
	return domain.Ticker{
		Ticker:          t.Ticker,
		Name:            t.Name,
		Market:          t.Market,
		PrimaryExchange: t.PrimaryExchange,
		Type:            t.Type,
		Active:          t.Active,
		CurrencyName:    t.CurrencyName,
		Locale:          t.Locale,
		CIK:             t.CIK,
		ListDate:        formatDate(time.Time(t.ListDate)),
		DelistedUTC:     formatTime(time.Time(t.DelistedUTC)),
	}
}

// formatDate returns a "2006-01-02" string or empty string for zero-value times.
func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// formatTime returns an RFC3339 string or empty string for zero-value times.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
