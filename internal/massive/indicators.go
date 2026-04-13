package massive

import (
	"context"
	"fmt"
	"time"

	"github.com/massive-com/client-go/v2/rest/models"

	"github.com/profitify/profitify-backend/internal/domain"
)

// FetchSMA fetches a simple moving average for a ticker at the given window
// and date. Returns nil if no data is available for that date.
func (c *Client) FetchSMA(ctx context.Context, ticker string, window int, date time.Time) (*float64, error) {
	var resp *models.GetSMAResponse
	ts := models.Millis(date)

	err := c.retry("FetchSMA", func() error {
		var callErr error
		resp, callErr = c.sdk.GetSMA(ctx, models.GetSMAParams{Ticker: ticker}.
			WithWindow(window).
			WithTimestamp(models.LTE, ts).
			WithOrder(models.Desc).
			WithLimit(1))
		return callErr
	})
	if err != nil {
		return nil, fmt.Errorf("massive.FetchSMA(%s, %d): %w", ticker, window, err)
	}

	if len(resp.Results.Values) == 0 {
		return nil, nil
	}
	return &resp.Results.Values[0].Value, nil
}

// FetchEMA fetches an exponential moving average for a ticker at the given
// window and date. Returns nil if no data is available.
func (c *Client) FetchEMA(ctx context.Context, ticker string, window int, date time.Time) (*float64, error) {
	var resp *models.GetEMAResponse
	ts := models.Millis(date)

	err := c.retry("FetchEMA", func() error {
		var callErr error
		resp, callErr = c.sdk.GetEMA(ctx, models.GetEMAParams{Ticker: ticker}.
			WithWindow(window).
			WithTimestamp(models.LTE, ts).
			WithOrder(models.Desc).
			WithLimit(1))
		return callErr
	})
	if err != nil {
		return nil, fmt.Errorf("massive.FetchEMA(%s, %d): %w", ticker, window, err)
	}

	if len(resp.Results.Values) == 0 {
		return nil, nil
	}
	return &resp.Results.Values[0].Value, nil
}

// FetchRSI fetches the relative strength index for a ticker at the given
// window and date. Returns nil if no data is available.
func (c *Client) FetchRSI(ctx context.Context, ticker string, window int, date time.Time) (*float64, error) {
	var resp *models.GetRSIResponse
	ts := models.Millis(date)

	err := c.retry("FetchRSI", func() error {
		var callErr error
		resp, callErr = c.sdk.GetRSI(ctx, models.GetRSIParams{Ticker: ticker}.
			WithWindow(window).
			WithTimestamp(models.LTE, ts).
			WithOrder(models.Desc).
			WithLimit(1))
		return callErr
	})
	if err != nil {
		return nil, fmt.Errorf("massive.FetchRSI(%s, %d): %w", ticker, window, err)
	}

	if len(resp.Results.Values) == 0 {
		return nil, nil
	}
	return &resp.Results.Values[0].Value, nil
}

// FetchMACD fetches MACD line, signal, and histogram for a ticker on the
// given date. Returns nil values for any component with no data.
func (c *Client) FetchMACD(ctx context.Context, ticker string, date time.Time) (line, signal, histogram *float64, err error) {
	var resp *models.GetMACDResponse
	ts := models.Millis(date)

	err = c.retry("FetchMACD", func() error {
		var callErr error
		resp, callErr = c.sdk.GetMACD(ctx, models.GetMACDParams{Ticker: ticker}.
			WithTimestamp(models.LTE, ts).
			WithOrder(models.Desc).
			WithLimit(1))
		return callErr
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("massive.FetchMACD(%s): %w", ticker, err)
	}

	if len(resp.Results.Values) == 0 {
		return nil, nil, nil, nil
	}

	v := resp.Results.Values[0]
	return &v.Value, &v.Signal, &v.Histogram, nil
}

// FetchAllIndicators fetches all Massive-sourced indicators for a ticker
// on the given date. Individual failures are logged but do not abort —
// partial results are returned with nil fields for failed calls.
func (c *Client) FetchAllIndicators(ctx context.Context, ticker string, date time.Time) (*domain.TechnicalIndicators, error) {
	tech := &domain.TechnicalIndicators{
		Time: date,
	}

	// SMA 20, 50, 200
	for _, w := range []int{20, 50, 200} {
		val, err := c.FetchSMA(ctx, ticker, w, date)
		if err != nil {
			c.logger.Warn("failed to fetch SMA", "ticker", ticker, "window", w, "error", err)
			continue
		}
		switch w {
		case 20:
			tech.SMA20 = val
		case 50:
			tech.SMA50 = val
		case 200:
			tech.SMA200 = val
		}
	}

	// EMA 12, 26
	for _, w := range []int{12, 26} {
		val, err := c.FetchEMA(ctx, ticker, w, date)
		if err != nil {
			c.logger.Warn("failed to fetch EMA", "ticker", ticker, "window", w, "error", err)
			continue
		}
		switch w {
		case 12:
			tech.EMA12 = val
		case 26:
			tech.EMA26 = val
		}
	}

	// RSI 14
	rsi, err := c.FetchRSI(ctx, ticker, 14, date)
	if err != nil {
		c.logger.Warn("failed to fetch RSI", "ticker", ticker, "error", err)
	} else {
		tech.RSI14 = rsi
	}

	// MACD
	line, signal, histogram, err := c.FetchMACD(ctx, ticker, date)
	if err != nil {
		c.logger.Warn("failed to fetch MACD", "ticker", ticker, "error", err)
	} else {
		tech.MACDLine = line
		tech.MACDSignal = signal
		tech.MACDHistogram = histogram
	}

	return tech, nil
}
