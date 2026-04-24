package massive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchTickerDetails_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "OK",
			"request_id": "test-123",
			"results": map[string]any{
				"ticker":                        "AAPL",
				"name":                          "Apple Inc.",
				"market_cap":                    2500000000000.0,
				"sic_code":                      "3571",
				"sic_description":               "Electronic Computers",
				"description":                   "Apple designs consumer electronics.",
				"homepage_url":                  "https://www.apple.com",
				"phone_number":                  "408-996-1010",
				"total_employees":               164000,
				"share_class_shares_outstanding": 15700000000,
				"weighted_shares_outstanding":    15600000000,
				"address": map[string]any{
					"address1":    "One Apple Park Way",
					"city":        "Cupertino",
					"state":       "CA",
					"postal_code": "95014",
				},
				"branding": map[string]any{
					"logo_url": "https://example.com/logo.png",
					"icon_url": "https://example.com/icon.png",
				},
			},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	fund, err := c.FetchTickerDetails(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fund.MarketCap != 2500000000000 {
		t.Errorf("MarketCap = %v, want 2500000000000", fund.MarketCap)
	}
	if fund.SICCode != "3571" {
		t.Errorf("SICCode = %q, want 3571", fund.SICCode)
	}
	if fund.TotalEmployees != 164000 {
		t.Errorf("TotalEmployees = %d, want 164000", fund.TotalEmployees)
	}
	if fund.Address.City != "Cupertino" {
		t.Errorf("Address.City = %q, want Cupertino", fund.Address.City)
	}
	if fund.Branding.LogoURL != "https://example.com/logo.png" {
		t.Errorf("Branding.LogoURL = %q, want https://example.com/logo.png", fund.Branding.LogoURL)
	}
}

func TestFetchTickerDetails_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "NOT_FOUND",
			"error":  "Ticker not found",
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.FetchTickerDetails(context.Background(), "INVALID")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}
