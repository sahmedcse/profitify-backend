package config

import (
	"testing"

	"github.com/profitify/profitify-backend/internal/testutil"
)

// TestDBSecretARN_PopulatedOrEmpty covers all eight Lambda config loaders
// (see loaders() in config_helpers_test.go), each of which gained an
// optional DBSecretARN field read from DB_SECRET_ARN.
func TestDBSecretARN_PopulatedOrEmpty(t *testing.T) {
	const arn = "arn:aws:secretsmanager:us-east-1:1:secret:db-admin"

	for _, l := range loaders() {
		t.Run(l.name, func(t *testing.T) {
			t.Run("set", func(t *testing.T) {
				testutil.ClearEnv(t)
				l.setBaseEnv(t)
				t.Setenv("DB_SECRET_ARN", arn)

				res, err := l.load()
				if err != nil {
					t.Fatalf("%s() error = %v", l.name, err)
				}
				if res.dbSecretARN != arn {
					t.Errorf("DBSecretARN = %q, want %q", res.dbSecretARN, arn)
				}
			})

			t.Run("unset", func(t *testing.T) {
				testutil.ClearEnv(t)
				l.setBaseEnv(t)

				res, err := l.load()
				if err != nil {
					t.Fatalf("%s() error = %v", l.name, err)
				}
				if res.dbSecretARN != "" {
					t.Errorf("DBSecretARN = %q, want empty", res.dbSecretARN)
				}
			})
		})
	}
}
