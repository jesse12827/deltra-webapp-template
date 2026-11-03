// Package database opens connections and runs migrations
// (Web-App Development Stack v2.0, Sections 9.2 and 9.4).
package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"deltra/webapp/db"
)

// Schema is the application's schema (Web-App Development Stack v2.0, Section 9.1).
const Schema = "webapp"

// Open returns a connection pool for databaseURL.
//
// In production, the application connects directly as its own role (webapp_app),
// and role is empty. In development and tests, role names the role to assume
// after connecting with SET ROLE, so grants are exercised exactly as in production.
func Open(ctx context.Context, databaseURL, role string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing DATABASE_URL: %w", err)
	}
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error { return prepareConn(ctx, c, role) }
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	return pool, nil
}

// OpenSQL returns a database/sql handle, used only for migrations.
func OpenSQL(databaseURL, role string) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing DATABASE_URL: %w", err)
	}
	return stdlib.OpenDB(*cfg, stdlib.OptionAfterConnect(func(ctx context.Context, c *pgx.Conn) error {
		return prepareConn(ctx, c, role)
	})), nil
}

func prepareConn(ctx context.Context, c *pgx.Conn, role string) error {
	if role != "" {
		if _, err := c.Exec(ctx, "SET ROLE "+pgx.Identifier{role}.Sanitize()); err != nil {
			return fmt.Errorf("SET ROLE %s: %w", role, err)
		}
	}
	_, err := c.Exec(ctx, "SET search_path = "+pgx.Identifier{Schema}.Sanitize())
	return err
}

// Migrator returns a goose provider for the embedded migrations in db/migrations.
func Migrator(sqlDB *sql.DB) (*goose.Provider, error) {
	migrations, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, err
	}
	return goose.NewProvider(goose.DialectPostgres, sqlDB, migrations)
}
