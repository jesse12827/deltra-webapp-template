package web_test

// Acceptance-style tests for the template's standard behavior. They exercise the
// whole application over HTTPS against a disposable database, signed in through
// development sign-in. Name application tests after SRS acceptance tests, for
// example TestAT01_LoginLogout (Web-App Development Stack, Section 13.1).

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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"deltra/webapp/internal/auth"
	"deltra/webapp/internal/config"
	"deltra/webapp/internal/database"
	"deltra/webapp/internal/dbgen"
	"deltra/webapp/internal/items"
	"deltra/webapp/internal/testdb"
	"deltra/webapp/internal/web"
)

type app struct {
	t      *testing.T
	srv    *httptest.Server
	pool   *pgxpool.Pool
	dbURL  string // administrator connection to the same database
	q      *dbgen.Queries
	admin  dbgen.User
	tech   dbgen.User
	former dbgen.User // inactive
}

func newApp(t *testing.T) *app {
	t.Helper()
	pool, dbURL := testdb.NewWithURL(t)
	q := dbgen.New(pool)
	ctx := context.Background()
	mk := func(email, role string, active bool) dbgen.User {
		u, err := q.CreateUser(ctx, dbgen.CreateUserParams{Email: email, DisplayName: email, Role: role, Active: active})
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	a := &app{t: t, pool: pool, dbURL: dbURL, q: q,
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

// response is what the tests inspect. The helpers read and close the body,
// so no test holds an open *http.Response.
type response struct {
	StatusCode int
	Header     http.Header
}

// do sends one request with the test's context and returns the response and body.
func (a *app) do(c *http.Client, method, path string, form url.Values) (response, string) {
	a.t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(a.t.Context(), method, a.srv.URL+path, body)
	if err != nil {
		a.t.Fatal(err)
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	res, err := c.Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	b, err := io.ReadAll(res.Body)
	if cerr := res.Body.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		a.t.Fatal(err)
	}
	return response{StatusCode: res.StatusCode, Header: res.Header}, string(b)
}

func (a *app) get(c *http.Client, path string) (response, string) {
	a.t.Helper()
	return a.do(c, http.MethodGet, path, nil)
}

func (a *app) post(c *http.Client, path string, form url.Values) response {
	a.t.Helper()
	if form == nil {
		form = url.Values{}
	}
	res, _ := a.do(c, http.MethodPost, path, form)
	return res
}

func (a *app) postBody(c *http.Client, path string, form url.Values) (response, string) {
	a.t.Helper()
	return a.do(c, http.MethodPost, path, form)
}

// failAudit makes every later insert of an audit event with this action fail,
// using a trigger installed by the test as the database administrator. The
// trigger exists only in this test's disposable database; it is never part of
// a migration.
func (a *app) failAudit(action string) {
	a.t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, a.dbURL)
	if err != nil {
		a.t.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()
	schema := pgx.Identifier{database.Schema}.Sanitize()
	if _, err := conn.Exec(ctx, "CREATE OR REPLACE FUNCTION "+schema+`.test_fail_audit() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.action = ANY (TG_ARGV) THEN
				RAISE EXCEPTION 'test: audit write refused for %', NEW.action;
			END IF;
			RETURN NEW;
		END $$`); err != nil {
		a.t.Fatal(err)
	}
	trigger := pgx.Identifier{"test_fail_audit_" + strings.ReplaceAll(action, ".", "_")}.Sanitize()
	if _, err := conn.Exec(ctx, "CREATE TRIGGER "+trigger+" BEFORE INSERT ON "+schema+".audit_events"+
		" FOR EACH ROW EXECUTE FUNCTION "+schema+".test_fail_audit("+pgx.Identifier{action}.Sanitize()+")"); err != nil {
		a.t.Fatal(err)
	}
}

// sessionCount returns how many sessions u has.
func (a *app) sessionCount(u dbgen.User) int {
	a.t.Helper()
	var n int
	if err := a.pool.QueryRow(context.Background(), "SELECT count(*) FROM sessions WHERE user_id = $1", u.ID).Scan(&n); err != nil {
		a.t.Fatal(err)
	}
	return n
}

// sessionCookie returns the session token held in c's cookie jar.
func (a *app) sessionCookie(c *http.Client) string {
	a.t.Helper()
	u, _ := url.Parse(a.srv.URL)
	for _, ck := range c.Jar.Cookies(u) {
		if ck.Name == auth.CookieName {
			return ck.Value
		}
	}
	return ""
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
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, a.srv.URL+"/items", strings.NewReader("title=forged"))
	if err != nil {
		t.Fatal(err)
	}
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

// Signing in ends any session the browser already had
// (Stack Section 10.2).
func TestSignInReplacesExistingSession(t *testing.T) {
	a := newApp(t)
	c := a.signIn(a.tech)
	old := a.sessionCookie(c)
	if old == "" {
		t.Fatal("no session cookie after sign-in")
	}
	if res := a.post(c, "/signin/dev", url.Values{"user_id": {strconv.FormatInt(a.admin.ID, 10)}}); res.StatusCode != http.StatusSeeOther {
		t.Fatalf("second sign-in: status %d", res.StatusCode)
	}
	if now := a.sessionCookie(c); now == "" || now == old {
		t.Fatal("second sign-in did not issue a new session token")
	}
	if n := a.sessionCount(a.tech); n != 0 {
		t.Fatalf("earlier session still stored: %d sessions for the first user", n)
	}
	if n := a.sessionCount(a.admin); n != 1 {
		t.Fatalf("%d sessions for the new user, want 1", n)
	}
	replay := a.client()
	u, _ := url.Parse(a.srv.URL)
	replay.Jar.SetCookies(u, []*http.Cookie{{Name: auth.CookieName, Value: old, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode}})
	if res, _ := a.get(replay, "/"); res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/signin" {
		t.Fatalf("old token: status %d, location %q; want redirect to sign-in", res.StatusCode, res.Header.Get("Location"))
	}
}

// The session and its signin.success event commit together
// (Stack Section 10.2).
func TestSigninAuditFailureLeavesNoSession(t *testing.T) {
	a := newApp(t)
	a.failAudit("signin.success")
	c := a.client()
	res := a.post(c, "/signin/dev", url.Values{"user_id": {strconv.FormatInt(a.tech.ID, 10)}})
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", res.StatusCode)
	}
	if a.sessionCookie(c) != "" {
		t.Error("session cookie set although the sign-in was not audited")
	}
	if n := a.sessionCount(a.tech); n != 0 {
		t.Errorf("%d session rows remain, want 0", n)
	}
}

// A sign-in denial that cannot be audited fails the request; sign-in is still refused.
func TestSigninDeniedAuditFailureFailsRequest(t *testing.T) {
	a := newApp(t)
	a.failAudit("signin.denied")
	c := a.client()
	res := a.post(c, "/signin/dev", url.Values{"user_id": {strconv.FormatInt(a.former.ID, 10)}})
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", res.StatusCode)
	}
	if a.sessionCookie(c) != "" || a.sessionCount(a.former) != 0 {
		t.Error("inactive user was given a session")
	}
}

// An authorization denial that cannot be audited fails the request; access is
// still refused (Stack Section 12).
func TestAccessDeniedAuditFailureFailsRequest(t *testing.T) {
	a := newApp(t)
	c := a.signIn(a.tech)
	a.failAudit("access.denied")
	res, body := a.get(c, "/admin/audit")
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", res.StatusCode)
	}
	if strings.Contains(body, "Audit log") {
		t.Error("admin page content shown to a tech user")
	}
}

// Sign-out and its audit event commit together: if the event cannot be
// recorded, the request fails (Stack Section 12).
func TestSignoutAuditFailureFailsRequest(t *testing.T) {
	a := newApp(t)
	c := a.signIn(a.tech)
	a.failAudit("signout")
	if res := a.post(c, "/signout", nil); res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", res.StatusCode)
	}
	if n := a.sessionCount(a.tech); n != 1 {
		t.Fatalf("%d sessions after failed sign-out, want the session unchanged", n)
	}
}

// Pages are not stored by caches; static files are (Stack Section 10.6).
func TestSignedInPagesAreNotCached(t *testing.T) {
	a := newApp(t)
	c := a.signIn(a.tech)
	for _, path := range []string{"/", "/items", "/signin"} {
		if res, _ := a.get(c, path); res.Header.Get("Cache-Control") != "no-store" {
			t.Errorf("%s: Cache-Control %q, want no-store", path, res.Header.Get("Cache-Control"))
		}
	}
	if res, _ := a.get(c, "/static/css/style.css"); res.Header.Get("Cache-Control") != "public, max-age=3600" {
		t.Errorf("static: Cache-Control %q", res.Header.Get("Cache-Control"))
	}
}

// The server refuses to register development sign-in outside development, even
// if configuration validation were bypassed (Stack Section 10.1).
func TestServerRefusesDevSigninOutsideDevelopment(t *testing.T) {
	for _, env := range []config.Env{config.Acceptance, config.Production} {
		_, err := web.New(web.Deps{
			Config: config.Config{Env: env, AuthMode: config.AuthDev},
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		})
		if err == nil {
			t.Errorf("%s: development sign-in was registered", env)
		}
	}
}
