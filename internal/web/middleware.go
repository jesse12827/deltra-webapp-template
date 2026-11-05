package web

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"deltra/webapp/internal/audit"
	"deltra/webapp/internal/auth"
	"deltra/webapp/internal/requestid"
)

// assignRequestID gives every request an identifier for logs, audit events, and error pages.
func assignRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := requestid.New()
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(requestid.With(r.Context(), id)))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// logRequests writes one structured log entry per request. It never logs
// request bodies, cookies, or query strings, which may contain personal data.
func (s *server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.Logger.InfoContext(r.Context(), "request",
			"method", r.Method, "path", r.URL.Path, "status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", requestid.From(r.Context()))
	})
}

// recoverPanics turns a panic into a logged error and a generic error page.
func (s *server) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler { //nolint:errorlint // sentinel value comparison is intended
					panic(v)
				}
				s.serverError(w, r, fmt.Errorf("panic: %v\n%s", v, debug.Stack()))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// securityHeaders sets the response headers required by
// Web-App Development Stack v2.0, Section 10.6. Nginx adds Strict-Transport-Security.
// Cache-Control: no-store keeps signed-in pages out of shared caches and the
// browser's back-forward cache; cacheStatic replaces it for /static/.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// loadSession attaches the signed-in user, if any, to the request context.
func (s *server) loadSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(auth.CookieName)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		u, err := s.Sessions.Lookup(r.Context(), c.Value)
		switch {
		case errors.Is(err, auth.ErrNoSession):
			auth.ClearCookie(w)
		case err != nil:
			s.serverError(w, r, err)
			return
		default:
			r = r.WithContext(auth.WithUser(r.Context(), u))
		}
		next.ServeHTTP(w, r)
	})
}

// requireUser allows only signed-in users. Others are sent to the sign-in page.
func (s *server) requireUser(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.UserFrom(r.Context()); !ok {
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/signin")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/signin", http.StatusSeeOther)
			return
		}
		h(w, r)
	})
}

// requireRole allows only signed-in users holding role. Denials are audited
// (Web-App Development Stack v2.0, Section 12).
func (s *server) requireRole(role auth.Role, h http.HandlerFunc) http.Handler {
	return s.requireUser(func(w http.ResponseWriter, r *http.Request) {
		u, _ := auth.UserFrom(r.Context())
		if !u.Has(role) {
			if err := audit.Record(r.Context(), s.q, audit.Event{
				UserID: audit.Ptr(u.ID), Action: "access.denied",
				Detail: map[string]string{"method": r.Method, "path": r.URL.Path, "required_role": string(role)},
			}); err != nil {
				// Fail closed: access is still refused, and the request fails.
				s.serverError(w, r, err)
				return
			}
			s.render.page(w, r, http.StatusForbidden, "error", pageData{
				Title: "Access denied",
				Data:  errorView{Message: "You do not have permission to view this page."},
			})
			return
		}
		h(w, r)
	})
}
