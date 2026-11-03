package database_test

import (
	"context"
	"testing"

	"deltra/webapp/internal/database"
	"deltra/webapp/internal/testdb"
)

// TestMigrationsUpDownUp is the migration quality gate
// (Web-App Development Stack v2.0, Section 13.2): every migration must apply,
// roll back completely, and apply again.
func TestMigrationsUpDownUp(t *testing.T) {
	ctx := context.Background()
	dbURL := testdb.Empty(t)

	sqlDB, err := database.OpenSQL(dbURL, testdb.OwnerRole)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()
	m, err := database.Migrator(sqlDB)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("first up: %v", err)
	}
	if _, err := m.DownTo(ctx, 0); err != nil {
		t.Fatalf("down to zero: %v", err)
	}
	var tables int
	if err := sqlDB.QueryRowContext(ctx,
		`SELECT count(*) FROM pg_tables WHERE schemaname = $1 AND tablename <> 'goose_db_version'`,
		database.Schema).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Fatalf("%d tables remain after rolling back every migration", tables)
	}
	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("second up: %v", err)
	}
}

// TestAppRoleCannotChangeAuditEvents confirms the least-privilege grants:
// the application role may insert audit events but never change or delete them.
func TestAppRoleCannotChangeAuditEvents(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	for _, stmt := range []string{
		`UPDATE audit_events SET action = 'tampered'`,
		`DELETE FROM audit_events`,
		`DELETE FROM items`,
		`CREATE TABLE not_allowed (id int)`,
	} {
		if _, err := pool.Exec(ctx, stmt); err == nil {
			t.Errorf("application role was allowed to run: %s", stmt)
		}
	}
}
