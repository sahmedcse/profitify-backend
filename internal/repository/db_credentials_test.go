package repository_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/profitify/profitify-backend/internal/db"
	"github.com/profitify/profitify-backend/internal/testutil"
)

// credentialFreeURL strips any userinfo from dsn, so it points at the same
// host, port and database with no embedded credentials.
func credentialFreeURL(t *testing.T, dsn string) string {
	t.Helper()
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parsing DATABASE_URL: %v", err)
	}
	u.User = nil
	return u.String()
}

// TestNewWithCredentials_RoundTripsReservedCharacters is the regression test
// for the bug this feature fixes: a password full of URL-reserved characters
// must survive being set directly on pgxpool.Config, rather than being
// percent-decoded (or mis-decoded) as part of a connection string.
func TestNewWithCredentials_RoundTripsReservedCharacters(t *testing.T) {
	adminPool := testPool(t)
	// testPool already skipped if this is unset, so it is safe to read here.
	dsn := os.Getenv("DATABASE_URL")
	ctx := context.Background()

	const password = `p#ss@w:rd/x?y%z&a=b`
	roleName := fmt.Sprintf("profitify_test_role_%d", time.Now().UnixNano())

	var createSQL string
	if err := adminPool.QueryRow(ctx,
		"SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', $1::text, $2::text)",
		roleName, password,
	).Scan(&createSQL); err != nil {
		t.Fatalf("building CREATE ROLE statement: %v", err)
	}
	if _, err := adminPool.Exec(ctx, createSQL); err != nil {
		t.Fatalf("CREATE ROLE: %v", err)
	}
	t.Cleanup(func() {
		var dropSQL string
		if err := adminPool.QueryRow(ctx,
			"SELECT format('DROP ROLE %I', $1::text)", roleName,
		).Scan(&dropSQL); err != nil {
			t.Errorf("building DROP ROLE statement: %v", err)
			return
		}
		if _, err := adminPool.Exec(ctx, dropSQL); err != nil {
			t.Errorf("DROP ROLE %s: %v", roleName, err)
		}
	})

	connStr := credentialFreeURL(t, dsn)
	src := &testutil.FakeCredentialSource{Username: roleName, Password: password}

	pool, err := db.NewWithCredentials(ctx, connStr, src)
	if err != nil {
		t.Fatalf("NewWithCredentials() error = %v", err)
	}
	defer pool.Close()

	var currentUser string
	if err := pool.QueryRow(ctx, "SELECT current_user").Scan(&currentUser); err != nil {
		t.Fatalf("SELECT current_user: %v", err)
	}
	if currentUser != roleName {
		t.Errorf("current_user = %q, want %q", currentUser, roleName)
	}
}
