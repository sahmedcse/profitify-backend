package config

import (
	"testing"
)

func TestCsvToSlice(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want []string
	}{
		{name: "empty", env: "", want: nil},
		{name: "single", env: "AAPL", want: []string{"AAPL"}},
		{name: "multiple", env: "AAPL,MSFT,GOOG", want: []string{"AAPL", "MSFT", "GOOG"}},
		{name: "with spaces", env: " AAPL , MSFT , GOOG ", want: []string{"AAPL", "MSFT", "GOOG"}},
		{name: "lowercase to upper", env: "aapl,msft", want: []string{"AAPL", "MSFT"}},
		{name: "mixed case", env: "Aapl,msFT", want: []string{"AAPL", "MSFT"}},
		{name: "trailing comma", env: "AAPL,", want: []string{"AAPL"}},
		{name: "only commas", env: ",,", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_CSV", tt.env)
			got := csvToSlice("TEST_CSV")
			if tt.want == nil {
				if got != nil {
					t.Errorf("csvToSlice(%q) = %v, want nil", tt.env, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("csvToSlice(%q) = %v, want %v", tt.env, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("csvToSlice(%q)[%d] = %q, want %q", tt.env, i, got[i], tt.want[i])
				}
			}
		})
	}
}
