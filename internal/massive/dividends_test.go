package massive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchDividends_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "OK",
			"request_id": "test-123",
			"results": []map[string]any{
				{
					"ticker":           "AAPL",
					"cash_amount":      0.24,
					"ex_dividend_date": "2026-03-15",
					"declaration_date": "2026-03-01",
					"record_date":      "2026-03-10",
					"pay_date":         "2026-03-20",
					"frequency":        4,
					"dividend_type":    "CD",
				},
				{
					"ticker":           "AAPL",
					"cash_amount":      0.23,
					"ex_dividend_date": "2025-12-15",
					"declaration_date": "2025-12-01",
					"record_date":      "2025-12-10",
					"pay_date":         "2025-12-20",
					"frequency":        4,
					"dividend_type":    "CD",
				},
			},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	dividends, err := c.FetchDividends(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dividends) != 2 {
		t.Fatalf("expected 2 dividends, got %d", len(dividends))
	}
	if dividends[0].CashAmount != 0.24 {
		t.Errorf("CashAmount = %v, want 0.24", dividends[0].CashAmount)
	}
	if dividends[0].ExDividendDate != "2026-03-15" {
		t.Errorf("ExDividendDate = %q, want 2026-03-15", dividends[0].ExDividendDate)
	}
	if dividends[0].Frequency != 4 {
		t.Errorf("Frequency = %d, want 4", dividends[0].Frequency)
	}
	if dividends[0].DistributionType != "CD" {
		t.Errorf("DistributionType = %q, want CD", dividends[0].DistributionType)
	}
}

func TestFetchDividends_NoDividends(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "OK",
			"request_id": "test-123",
			"results":    []any{},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	dividends, err := c.FetchDividends(context.Background(), "NODIV")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dividends) != 0 {
		t.Errorf("expected 0 dividends, got %d", len(dividends))
	}
}
