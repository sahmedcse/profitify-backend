package massive

import (
	"context"
	"fmt"

	"github.com/massive-com/client-go/v2/rest/models"

	"github.com/profitify/profitify-backend/internal/domain"
)

// FetchTickerDetails fetches company details from Massive for a given ticker.
func (c *Client) FetchTickerDetails(ctx context.Context, ticker string) (*domain.TickerFundamentals, error) {
	var result *domain.TickerFundamentals

	c.logger.Info("fetching ticker details from Massive", "ticker", ticker)

	err := c.retry("FetchTickerDetails", func() error {
		resp, err := c.sdk.GetTickerDetails(ctx, &models.GetTickerDetailsParams{
			Ticker: ticker,
		})
		if err != nil {
			return fmt.Errorf("getting ticker details: %w", err)
		}
		result = mapTickerDetails(resp.Results)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("massive.FetchTickerDetails: %w", err)
	}

	return result, nil
}

func mapTickerDetails(t models.Ticker) *domain.TickerFundamentals {
	return &domain.TickerFundamentals{
		MarketCap:                 t.MarketCap,
		SharesOutstanding:         t.ShareClassSharesOutstanding,
		WeightedSharesOutstanding: t.WeightedSharesOutstanding,
		SICCode:                   t.SICCode,
		SICDescription:            t.SICDescription,
		Description:               t.Description,
		HomepageURL:               t.HomepageURL,
		PhoneNumber:               t.PhoneNumber,
		TotalEmployees:            int(t.TotalEmployees),
		Address: domain.Address{
			Address1:   t.Address.Address1,
			City:       t.Address.City,
			State:      t.Address.State,
			PostalCode: t.Address.PostalCode,
		},
		Branding: domain.Branding{
			IconURL: t.Branding.IconURL,
			LogoURL: t.Branding.LogoURL,
		},
	}
}
