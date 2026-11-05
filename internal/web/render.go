package web

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"deltra/webapp/internal/auth"
	"deltra/webapp/internal/requestid"
	assets "deltra/webapp/web"
)

// pageData is passed to every page template.
type pageData struct {
	Title     string
	User      *auth.User
	RequestID string
	Data      any
}

type errorView struct {
	Message   string
	RequestID string
}

type renderer struct {
	pages map[string]*template.Template
}

var funcs = template.FuncMap{
	// date formats a date-only value; nil renders as an empty string.
	"date": func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.Format(time.DateOnly)
	},
	// datetime formats a point in time in the server's local zone.
	"datetime": func(t time.Time) string { return t.Local().Format("2006-01-02 15:04") },
	"add":      func(a, b int) int { return a + b },
}

// newRenderer parses each page in web/templates/pages together with the shared
// layout and partials, so every page is checked when the server starts.
func newRenderer() (*renderer, error) {
	files, err := fs.Glob(assets.Files, "templates/pages/*.html")
	if err != nil {
		return nil, err
	}
	r := &renderer{pages: map[string]*template.Template{}}
	for _, f := range files {
		name := strings.TrimSuffix(path.Base(f), ".html")
		t, err := template.New(name).Funcs(funcs).ParseFS(assets.Files,
			"templates/layout.html", "templates/partials/*.html", f)
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", f, err)
		}
		r.pages[name] = t
	}
	return r, nil
}

// page renders a full page inside the layout.
func (rd *renderer) page(w http.ResponseWriter, r *http.Request, status int, name string, d pageData) {
	rd.execute(w, r, status, name, "layout", d)
}

// block renders one named block of a page, for htmx partial updates.
func (rd *renderer) block(w http.ResponseWriter, r *http.Request, status int, name, block string, d pageData) {
	rd.execute(w, r, status, name, block, d)
}

func (rd *renderer) execute(w http.ResponseWriter, r *http.Request, status int, name, tmpl string, d pageData) {
	if u, ok := auth.UserFrom(r.Context()); ok {
		d.User = &u
	}
	d.RequestID = requestid.From(r.Context())
	t, ok := rd.pages[name]
	if !ok {
		slog.ErrorContext(r.Context(), "unknown template", "name", name)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// Render to a buffer first, so a template error never sends a half-written page.
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, tmpl, d); err != nil {
		slog.ErrorContext(r.Context(), "rendering template", "name", name, "err", err, "request_id", d.RequestID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

// serverError logs err with the request identifier and shows the user a
// generic message with that identifier (Web-App Development Stack v2.1, Section 10.7).
func (s *server) serverError(w http.ResponseWriter, r *http.Request, err error) {
	id := requestid.From(r.Context())
	s.Logger.ErrorContext(r.Context(), "server error", "err", err, "request_id", id, "path", r.URL.Path)
	s.render.page(w, r, http.StatusInternalServerError, "error", pageData{
		Title: "Something went wrong",
		Data:  errorView{Message: "Something went wrong. If it keeps happening, report this reference.", RequestID: id},
	})
}

func (s *server) notFound(w http.ResponseWriter, r *http.Request) {
	s.render.page(w, r, http.StatusNotFound, "error", pageData{
		Title: "Not found",
		Data:  errorView{Message: "The page or record you asked for does not exist."},
	})
}

// isHTMX reports whether r is an htmx request targeting the element with id target.
func isHTMX(r *http.Request, target string) bool {
	return r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") == target
}
