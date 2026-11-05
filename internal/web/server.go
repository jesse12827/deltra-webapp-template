// Package web is the HTTP layer: routes, middleware, handlers, and template
// rendering (Web-App Development Stack v2.1, Section 7). Handlers parse and
// validate input, call domain packages, and render. Handlers contain no SQL.
package web

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"deltra/webapp/internal/auth"
	"deltra/webapp/internal/config"
	"deltra/webapp/internal/dbgen"
	"deltra/webapp/internal/items"
	assets "deltra/webapp/web"
)

// Deps are the server's dependencies, constructed in cmd/webapp.
type Deps struct {
	Config   config.Config
	Pool     *pgxpool.Pool
	Sessions *auth.Sessions
	Entra    *auth.Entra // nil unless AUTH_MODE=entra
	Items    *items.Service
	Logger   *slog.Logger
}

type server struct {
	Deps
	q      *dbgen.Queries
	render *renderer
}

// New returns the application's HTTP handler.
func New(d Deps) (http.Handler, error) {
	r, err := newRenderer()
	if err != nil {
		return nil, err
	}
	s := &server{Deps: d, q: dbgen.New(d.Pool), render: r}

	static, err := fs.Sub(assets.Files, "static")
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()

	// Public routes.
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.Handle("GET /static/", cacheStatic(http.StripPrefix("/static/", http.FileServerFS(static))))
	mux.HandleFunc("GET /signin", s.signinPage)
	if d.Config.AuthMode == config.AuthDev {
		// Configuration already refuses dev mode outside development; this is a second check.
		if d.Config.Env != config.Development {
			return nil, errors.New("development sign-in is permitted only when APP_ENV=development")
		}
		mux.HandleFunc("POST /signin/dev", s.devSignin)
	} else {
		mux.HandleFunc("GET /signin/entra", s.entraBegin)
		mux.HandleFunc("GET /auth/callback", s.entraCallback)
	}

	// Signed-in routes. Access is denied by default: every route below declares its requirement.
	mux.Handle("POST /signout", s.requireUser(s.signout))
	mux.Handle("GET /{$}", s.requireUser(s.dashboard))
	mux.Handle("GET /items", s.requireUser(s.itemsList))
	mux.Handle("GET /items/new", s.requireRole(auth.RoleTech, s.itemNew))
	mux.Handle("POST /items", s.requireRole(auth.RoleTech, s.itemCreate))
	mux.Handle("GET /items/{id}", s.requireUser(s.itemDetail))
	mux.Handle("GET /items/{id}/edit", s.requireRole(auth.RoleTech, s.itemEdit))
	mux.Handle("POST /items/{id}", s.requireRole(auth.RoleTech, s.itemUpdate))

	// Administrator routes.
	mux.Handle("GET /admin/audit", s.requireRole(auth.RoleAdmin, s.auditList))

	// Unmatched paths render the standard not-found page.
	mux.HandleFunc("/", s.notFound)

	// Middleware, outermost first.
	var h http.Handler = mux
	h = s.loadSession(h)
	h = http.NewCrossOriginProtection().Handler(h) // CSRF defense (Section 10.4)
	h = securityHeaders(h)
	h = s.recoverPanics(h)
	h = s.logRequests(h)
	h = assignRequestID(h)
	return h, nil
}

func (s *server) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if err := s.Pool.Ping(ctx); err != nil {
		s.Logger.ErrorContext(ctx, "health check: database unavailable", "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("database unavailable\n"))
		return
	}
	_, _ = w.Write([]byte("ok\n"))
}

func cacheStatic(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		h.ServeHTTP(w, r)
	})
}
