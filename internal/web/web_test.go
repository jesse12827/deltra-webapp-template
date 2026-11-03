package web_test

// Acceptance-style tests for the template's standard behavior. They exercise the
// whole application over HTTPS against a disposable database, signed in through
// development sign-in. Name application tests after SRS acceptance tests, for
// example TestAT01_LoginLogout (Web-App Development Stack v2.0, Section 13.1).

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"deltra/webapp/internal/auth"
	"deltra/webapp/internal/config"
	"deltra/webapp/internal/dbgen"
	"deltra/webapp/internal/items"
	"deltra/webapp/internal/testdb"
	"deltra/webapp/internal/web"
)

type app struct {
	t      *testing.T
	srv    *httptest.Server
	pool   *pgxpool.Pool
	q      *dbgen.Queries
	admin  dbgen.User
	tech   dbgen.User
	former dbgen.User // inactive
}

func newApp(t *testing.T) *app {
	t.Helper()
	pool := testdb.New(t)
	q := dbgen.New(pool)
	ctx := context.Background()
	mk := func(email, role string, active bool) dbgen.User {
		u, err := q.CreateUser(ctx, dbgen.CreateUserParams{Email: email, DisplayName: email, Role: role, Active: active})
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	a := &app{t: t, pool: pool, q: q,
		admin:  mk("admin@synthetic.test", "admin", true),
		tech:   mk("tech@synthetic.test", "tech", true),
		former: mk("former@synthetic.test", "tech", false),
	}
	cfg := config.Config{Env: config.Development, AuthMode: config.AuthDev, SessionIdle: 30 * time.Minute, SessionAbsolute: 8 * time.Hour}
	h, err := web.New(web.Deps{
		Config: cfg, Pool: pool, Sessions: auth.NewSessions(pool, cfg.SessionIdle, cfg.SessionAbsolute),
		Items: items.NewService(pool), Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	a.srv = httptest.NewTLSServer(h) // TLS, because the session cookie is Secure
	t.Cleanup(a.srv.Close)
	return a
}

// client returns a browser-like client with its own cookie jar that does not follow redirects.
func (a *app) client() *http.Client {
	jar, _ := cookiejar.New(nil)
	c := a.srv.Client()
	c.Jar = jar
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return c
}

func (a *app) signIn(u dbgen.User) *http.Client {
	a.t.Helper()
	c := a.client()
	res := a.post(c, "/signin/dev", url.Values{"user_id": {strconv.FormatInt(u.ID, 10)}})
	if res.StatusCode != http.StatusSeeOther {
		a.t.Fatalf("sign-in as %s: status %d", u.Email, res.StatusCode)
	}
	return c
}

func (a *app) get(c *http.Client, path string) (*http.Response, string) {
	a.t.Helper()
	res, err := c.Get(a.srv.URL + path)
	if err != nil {
		a.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res, string(b)
}

func (a *app) post(c *http.Client, path string, form url.Values) *http.Response {
	a.t.Helper()
	res, err := c.PostForm(a.srv.URL+path, form)
	if err != nil {
		a.t.Fatal(err)
	}
	_ = res.Body.Close()
	return res
}

func (a *app) postBody(c *http.Client, path string, form url.Values) (*http.Response, string) {
	a.t.Helper()
	res, err := c.PostForm(a.srv.URL+path, form)
	if err != nil {
		a.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res, string(b)
}

func TestSignedOutUserIsRedirectedToSignIn(t *testing.T) {
	a := newApp(t)
	res, _ := a.get(a.client(), "/items")
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/signin" {
		t.Fatalf("status %d, location %q", res.StatusCode, res.Header.Get("Location"))
	}
}

func TestSignInAndSignOut(t *testing.T) {
	a := newApp(t)
	c := a.signIn(a.tech)
	if res, body := a.get(c, "/"); res.StatusCode != http.StatusOK || !strings.Contains(body, "Dashboard") {
		t.Fatalf("dashboard after sign-in: status %d", res.StatusCode)
	}
	a.post(c, "/signout", nil)
	if res, _ := a.get(c, "/"); res.StatusCode != http.StatusSeeOther {
		t.Fatalf("after sign-out, status %d; want redirect to sign-in", res.StatusCode)
	}
}

func TestInactiveUserIsDeniedAndAudited(t *testing.T) {
	a := newApp(t)
	res := a.post(a.client(), "/signin/dev", url.Values{"user_id": {strconv.FormatInt(a.former.ID, 10)}})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d, want 403", res.StatusCode)
	}
	events, err := a.q.ListAuditEvents(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 || events[0].Action != "signin.denied" {
		t.Fatal("denied sign-in was not audited")
	}
}

func TestTechCannotOpenAdminPage(t *testing.T) {
	a := newApp(t)
	if res, _ := a.get(a.signIn(a.tech), "/admin/audit"); res.StatusCode != http.StatusForbidden {
		t.Fatalf("tech: status %d, want 403", res.StatusCode)
	}
	if res, _ := a.get(a.signIn(a.admin), "/admin/audit"); res.StatusCode != http.StatusOK {
		t.Fatalf("admin: status %d, want 200", res.StatusCode)
	}
}

func TestCreateItemThenFindIt(t *testing.T) {
	a := newApp(t)
	c := a.signIn(a.tech)
	res := a.post(c, "/items", url.Values{"title": {"Synthetic laptop 100%"}, "status": {"open"}, "due_on": {"2026-12-01"}})
	if res.StatusCode != http.StatusSeeOther || !strings.HasPrefix(res.Header.Get("Location"), "/items/") {
		t.Fatalf("create: status %d, location %q", res.StatusCode, res.Header.Get("Location"))
	}
	_, body := a.get(c, "/items?q="+url.QueryEscape("100%"))
	if !strings.Contains(body, "Synthetic laptop 100%") {
		t.Fatal("created item not found by search")
	}
	_, body = a.get(c, "/items?q="+url.QueryEscape("nothing-matches"))
	if !strings.Contains(body, "No items match") {
		t.Fatal("empty search did not show the empty-state message")
	}
}

func TestValidationKeepsInputAndShowsFieldErrors(t *testing.T) {
	a := newApp(t)
	res, body := a.postBody(a.signIn(a.tech), "/items", url.Values{"title": {""}, "description": {"keep this text"}, "due_on": {"12/01/2026"}})
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, want 422", res.StatusCode)
	}
	for _, want := range []string{"keep this text", "Enter a title.", "Enter the date as YYYY-MM-DD."} {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %q", want)
		}
	}
}

func TestConcurrentEditIsRejected(t *testing.T) {
	a := newApp(t)
	it, _, err := items.NewService(a.pool).Create(context.Background(), a.tech.ID, items.Input{Title: "Synthetic shared item"})
	if err != nil {
		t.Fatal(err)
	}
	path := "/items/" + strconv.FormatInt(it.ID, 10)
	form := func(title string) url.Values {
		return url.Values{"title": {title}, "status": {"open"}, "version": {strconv.Itoa(int(it.Version))}}
	}
	first := a.post(a.signIn(a.tech), path, form("First edit"))
	if first.StatusCode != http.StatusSeeOther {
		t.Fatalf("first edit: status %d", first.StatusCode)
	}
	second, body := a.postBody(a.signIn(a.admin), path, form("Second edit from a stale page"))
	if second.StatusCode != http.StatusConflict || !strings.Contains(body, "changed by someone else") {
		t.Fatalf("second edit: status %d, want 409 with explanation", second.StatusCode)
	}
	got, _ := a.q.GetItem(context.Background(), it.ID)
	if got.Title != "First edit" {
		t.Fatalf("title = %q; the first edit was overwritten", got.Title)
	}
}

func TestCrossOriginPostIsRejected(t *testing.T) {
	a := newApp(t)
	c := a.signIn(a.tech)
	req, _ := http.NewRequest(http.MethodPost, a.srv.URL+"/items", strings.NewReader("title=forged"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site") // what a browser sends for a forged cross-site form
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d, want 403", res.StatusCode)
	}
}

func TestSecurityHeadersAndHealth(t *testing.T) {
	a := newApp(t)
	res, body := a.get(a.client(), "/healthz")
	if res.StatusCode != http.StatusOK || strings.TrimSpace(body) != "ok" {
		t.Fatalf("healthz: %d %q", res.StatusCode, body)
	}
	if !strings.Contains(res.Header.Get("Content-Security-Policy"), "script-src 'self'") {
		t.Error("Content-Security-Policy missing")
	}
	if res.Header.Get("X-Request-ID") == "" {
		t.Error("X-Request-ID missing")
	}
}

func TestUnknownRecordShowsNotFound(t *testing.T) {
	a := newApp(t)
	if res, _ := a.get(a.signIn(a.tech), "/items/999999"); res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, want 404", res.StatusCode)
	}
}
