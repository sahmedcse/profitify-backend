package config

import (
	"strings"
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

func TestRequired(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "set", value: "something", wantErr: false},
		{name: "empty", value: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_REQUIRED", tt.value)
			got, err := required("TEST_REQUIRED")
			if tt.wantErr {
				if err == nil {
					t.Fatal("required() error = nil, want error")
				}
				if !strings.Contains(err.Error(), "TEST_REQUIRED is required") {
					t.Errorf("error = %q, want it to name the missing key", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("required() error = %v", err)
			}
			if got != tt.value {
				t.Errorf("required() = %q, want %q", got, tt.value)
			}
		})
	}
}

func TestEnvOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		want     string
	}{
		{name: "set overrides fallback", value: "custom", fallback: "fb", want: "custom"},
		{name: "empty uses fallback", value: "", fallback: "fb", want: "fb"},
		{name: "whitespace is kept", value: " ", fallback: "fb", want: " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ENV_DEFAULT", tt.value)
			if got := envOrDefault("TEST_ENV_DEFAULT", tt.fallback); got != tt.want {
				t.Errorf("envOrDefault() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIntOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback int
		want     int
	}{
		{name: "valid int", value: "12", fallback: 4, want: 12},
		{name: "empty uses fallback", value: "", fallback: 4, want: 4},
		{name: "non-numeric uses fallback", value: "abc", fallback: 4, want: 4},
		{name: "negative", value: "-3", fallback: 4, want: -3},
		{name: "zero", value: "0", fallback: 4, want: 0},
		{name: "float is invalid", value: "1.5", fallback: 4, want: 4},
		{name: "whitespace is invalid", value: " 7 ", fallback: 4, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_INT_DEFAULT", tt.value)
			if got := intOrDefault("TEST_INT_DEFAULT", tt.fallback); got != tt.want {
				t.Errorf("intOrDefault() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNormalizeSymbols(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "nil input", in: nil, want: nil},
		{name: "empty input", in: []string{}, want: nil},
		{name: "trims and uppercases", in: []string{" aapl ", "msft"}, want: []string{"AAPL", "MSFT"}},
		{name: "drops blanks", in: []string{"", " ", "AAPL"}, want: []string{"AAPL"}},
		{name: "all blank returns nil", in: []string{"", "  "}, want: nil},
		{name: "dedupes preserving first-seen order", in: []string{"AAPL", "msft", "aapl", "MSFT", "GOOG"}, want: []string{"AAPL", "MSFT", "GOOG"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeSymbols(tt.in)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("NormalizeSymbols(%v) = %v, want nil", tt.in, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("NormalizeSymbols(%v) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("NormalizeSymbols(%v)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
				}
			}
		})
	}
}
