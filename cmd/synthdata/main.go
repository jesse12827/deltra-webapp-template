// Command synthdata loads synthetic data into a development or acceptance
// database (AI Policy Section 26A.7). It refuses to run in production and is
// never deployed.
//
//	synthdata [-items 600] [-seed 1]
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"

	"deltra/webapp/internal/config"
	"deltra/webapp/internal/database"
	"deltra/webapp/internal/dbgen"
	"deltra/webapp/internal/synth"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	n := flag.Int("items", 600, "number of synthetic items to create")
	seed := flag.Uint64("seed", 1, "random seed; the same seed always produces the same data")
	flag.Parse()

	env := config.Env(os.Getenv("APP_ENV"))
	if env != config.Development && env != config.Acceptance {
		return errors.New("synthdata runs only with APP_ENV=development or APP_ENV=acceptance")
	}

	ctx := context.Background()
	pool, err := database.Open(ctx, os.Getenv("DATABASE_URL"), os.Getenv("DATABASE_ROLE"))
	if err != nil {
		return err
	}
	defer pool.Close()
	q := dbgen.New(pool)

	var adminID int64
	for _, u := range synth.Users() {
		existing, err := q.GetUserByEmail(ctx, u.Email)
		switch {
		case err == nil:
			if u.Role == "admin" {
				adminID = existing.ID
			}
			continue
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}
		created, err := q.CreateUser(ctx, dbgen.CreateUserParams{
			Email: u.Email, DisplayName: u.DisplayName, Role: u.Role, Active: u.Active,
		})
		if err != nil {
			return fmt.Errorf("creating user %s: %w", u.Email, err)
		}
		if u.Role == "admin" {
			adminID = created.ID
		}
	}

	for _, it := range synth.Items(*seed, *n) {
		if _, err := q.CreateItem(ctx, dbgen.CreateItemParams{
			Title: it.Title, Description: it.Description, Status: it.Status, DueOn: it.DueOn, UserID: adminID,
		}); err != nil {
			return fmt.Errorf("creating item: %w", err)
		}
	}
	fmt.Printf("loaded %d synthetic users and %d items (seed %d)\n", len(synth.Users()), *n, *seed)
	return nil
}
