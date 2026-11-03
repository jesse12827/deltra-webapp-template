// Command webapp is the application binary. It contains its templates, static
// files, and migrations (Web-App Development Stack v2.0, Section 14).
//
// Subcommands:
//
//	serve             run the web server (default)
//	migrate up        apply all pending migrations
//	migrate down      roll back the most recent migration
//	migrate status    list migrations and whether each is applied
//	version           print the version and commit
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"deltra/webapp/internal/auth"
	"deltra/webapp/internal/config"
	"deltra/webapp/internal/database"
	"deltra/webapp/internal/items"
	"deltra/webapp/internal/web"
)

// Set at build time with -ldflags (see Makefile).
var (
	version = "dev"
	commit  = "none"
)

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, getenv func(string) string, stdout io.Writer) error {
	cmd := "serve"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "version":
		_, err := fmt.Fprintf(stdout, "%s (commit %s)\n", version, commit)
		return err
	case "serve":
		return serve(getenv)
	case "migrate":
		return migrate(args, getenv, stdout)
	default:
		return fmt.Errorf("unknown command %q (want serve, migrate, or version)", cmd)
	}
}

func serve(getenv func(string) string) error {
	cfg, err := config.Load(getenv)
	if err != nil {
		return fmt.Errorf("configuration:\n%w", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.DatabaseURL, cfg.DatabaseRole)
	if err != nil {
		return err
	}
	defer pool.Close()

	sessions := auth.NewSessions(pool, cfg.SessionIdle, cfg.SessionAbsolute)
	var entra *auth.Entra
	if cfg.AuthMode == config.AuthEntra {
		if entra, err = auth.NewEntra(ctx, cfg.Entra); err != nil {
			return err
		}
	}

	handler, err := web.New(web.Deps{
		Config: cfg, Pool: pool, Sessions: sessions, Entra: entra,
		Items: items.NewService(pool), Logger: logger,
	})
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go purgeExpiredSessions(ctx, sessions, logger)

	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	logger.Info("server started", "addr", cfg.Addr, "env", cfg.Env, "auth_mode", cfg.AuthMode,
		"version", version, "commit", commit)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func purgeExpiredSessions(ctx context.Context, s *auth.Sessions, logger *slog.Logger) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := s.DeleteExpired(ctx); err != nil {
				logger.Error("purging expired sessions", "err", err)
			}
		}
	}
}

// migrate runs schema migrations. In production the Approver runs it with the
// owner role, following docs/operations/runbook.md (Web-App Development Stack v2.0, Section 9.4).
func migrate(args []string, getenv func(string) string, stdout io.Writer) error {
	if len(args) != 1 {
		return errors.New("usage: migrate up|down|status")
	}
	dbURL := getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	sqlDB, err := database.OpenSQL(dbURL, getenv("DATABASE_ROLE"))
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()
	m, err := database.Migrator(sqlDB)
	if err != nil {
		return err
	}
	ctx := context.Background()

	switch args[0] {
	case "up":
		results, err := m.Up(ctx)
		for _, r := range results {
			_, _ = fmt.Fprintf(stdout, "applied %s (%s)\n", r.Source.Path, r.Duration.Round(time.Millisecond))
		}
		if err != nil {
			return err
		}
		if len(results) == 0 {
			_, _ = fmt.Fprintln(stdout, "no pending migrations")
		}
	case "down":
		r, err := m.Down(ctx)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "rolled back %s\n", r.Source.Path)
	case "status":
		statuses, err := m.Status(ctx)
		if err != nil {
			return err
		}
		for _, st := range statuses {
			_, _ = fmt.Fprintf(stdout, "%-8s %s\n", st.State, st.Source.Path)
		}
	default:
		return fmt.Errorf("unknown migrate command %q (want up, down, or status)", args[0])
	}
	return nil
}
