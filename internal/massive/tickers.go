package massive

import (
	"context"
	"fmt"
	"net/http"
	"time"

	v3rest "github.com/massive-com/client-go/v3/rest"
	v3gen "github.com/massive-com/client-go/v3/rest/gen"

	"github.com/profitify/profitify-backend/internal/domain"
)

// FetchActiveTickers fetches active US equity tickers from Massive using the v3 SDK.
// With v3 and no auto-pagination, a single API call returns up to maxTickers results.
func (c *Client) FetchActiveTickers(ctx context.Context) ([]domain.Ticker, error) {
	var tickers []domain.Ticker

	c.logger.Info("fetching active tickers from Massive")

	err := c.retry("FetchActiveTickers", func() error {
		tickers = nil // reset on retry

		params := &v3gen.ListTickersParams{
			Type:   v3rest.Ptr(v3gen.ListTickersParamsTypeCS),
			Market: v3rest.Ptr(v3gen.ListTickersParamsMarketStocks),
			Active: v3rest.Bool(true),
			Sort:   v3rest.Ptr(v3gen.ListTickersParamsSortTicker),
			Order:  v3rest.Ptr(v3gen.ListTickersParamsOrderAsc),
			Limit:  v3rest.Int(c.maxTickers),
		}

		resp, err := c.tickerSDK.ListTickersWithResponse(ctx, params)
		if err != nil {
			return fmt.Errorf("calling ListTickers: %w", err)
		}

		if sc := resp.HTTPResponse.StatusCode; sc != http.StatusOK {
			return &httpError{statusCode: sc, body: string(resp.Body)}
		}

		if resp.JSON200 == nil || resp.JSON200.Results == nil {
			return nil
		}

		for _, t := range *resp.JSON200.Results {
			tickers = append(tickers, domain.Ticker{
				Ticker:          t.Ticker,
				Name:            t.Name,
				Market:          string(t.Market),
				PrimaryExchange: derefStr(t.PrimaryExchange),
				Type:            derefStr(t.Type),
				Active:          derefBool(t.Active),
				CurrencyName:    derefStr(t.CurrencyName),
				Locale:          string(t.Locale),
				CIK:             derefStr(t.Cik),
				DelistedUTC:     formatTime(derefTime(t.DelistedUtc)),
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("massive.FetchActiveTickers: %w", err)
	}

	c.logger.Info("fetched active tickers", "count", len(tickers))
	return tickers, nil
}

// derefStr safely dereferences a *string, returning "" for nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// derefBool safely dereferences a *bool, returning false for nil.
func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// derefTime safely dereferences a *time.Time, returning zero time for nil.
func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
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
