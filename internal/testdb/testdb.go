// Package testdb creates a disposable PostgreSQL database for one test,
// built exactly as production is: bootstrap scripts, then migrations as the
// owner role, then a pool that runs as the application role
// (Web-App Development Stack v2.0, Section 13.1).
//
// Set TEST_DATABASE_URL to an administrator connection URL on a local
// development server, for example: postgres:///postgres?host=/var/run/postgresql
// `make test` and `make check` set it, and also set REQUIRE_TEST_DATABASE=1,
// which turns a missing database into a failure instead of a skip.
package testdb

import (
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"deltra/webapp/db"
	"deltra/webapp/internal/database"
)

const (
	ownerRole = database.Schema + "_owner"
	appRole   = database.Schema + "_app"
)

// adminURL returns TEST_DATABASE_URL, skipping (or failing under make) when unset.
func adminURL(t testing.TB) string {
	t.Helper()
	u := os.Getenv("TEST_DATABASE_URL")
	if u == "" {
		if os.Getenv("REQUIRE_TEST_DATABASE") != "" {
			t.Fatal("TEST_DATABASE_URL is required")
		}
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	return u
}

// withDatabase returns adminURL with its database name replaced by name.
func withDatabase(t testing.TB, adminURL, name string) string {
	t.Helper()
	u, err := url.Parse(adminURL)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatalf("TEST_DATABASE_URL must be a postgres:// URL, got %q", adminURL)
	}
	u.Path = "/" + name
	return u.String()
}

// Empty creates a bootstrapped database with no migrations applied and returns
// an administrator connection URL for it. The database is dropped when the test ends.
func Empty(t testing.TB) string {
	t.Helper()
	ctx := context.Background()
	admin := adminURL(t)

	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatalf("connecting to TEST_DATABASE_URL: %v", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	runFile(ctx, t, conn, "bootstrap/001_roles.sql")
	name := fmt.Sprintf("%s_test_%d", database.Schema, time.Now().UnixNano())
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()+" OWNER "+ownerRole); err != nil {
		t.Fatalf("creating test database: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		c, err := pgx.Connect(ctx, admin)
		if err != nil {
			t.Logf("cleanup: %v", err)
			return
		}
		defer func() { _ = c.Close(ctx) }()
		if _, err := c.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)"); err != nil {
			t.Logf("cleanup: dropping %s: %v", name, err)
		}
	})

	dbURL := withDatabase(t, admin, name)
	dbConn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	defer func() { _ = dbConn.Close(ctx) }()
	runFile(ctx, t, dbConn, "bootstrap/002_schema.sql")

	return dbURL
}

// New returns a pool, running as the application role, on a fully migrated
// disposable database. The database is dropped when the test ends.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	dbURL := Empty(t)

	sqlDB, err := database.OpenSQL(dbURL, ownerRole)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()
	m, err := database.Migrator(sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("applying migrations: %v", err)
	}

	pool, err := database.Open(ctx, dbURL, appRole)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// OwnerRole and AppRole are exported for tests that need to name the roles.
const (
	OwnerRole = ownerRole
	AppRole   = appRole
)

// runFile executes an embedded bootstrap script, which may contain several statements.
func runFile(ctx context.Context, t testing.TB, c *pgx.Conn, path string) {
	t.Helper()
	b, err := fs.ReadFile(db.Bootstrap, path)
	if err != nil {
		t.Fatal(err)
	}
	// PgConn().Exec uses the simple query protocol, which allows multiple statements.
	if _, err := c.PgConn().Exec(ctx, string(b)).ReadAll(); err != nil {
		t.Fatalf("running %s: %v", path, err)
	}
}
