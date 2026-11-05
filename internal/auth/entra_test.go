package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

const (
	testTenant = "00000000-0000-4000-8000-0000000000aa"
	testObject = "00000000-0000-4000-8000-0000000000bb"
)

// The callback's error parameter is untrusted text from a public route, so it
// never reaches the error that the handler logs (security final review
// 2026-11-05, finding 1).
func TestFinishDoesNotCopyCallbackErrorText(t *testing.T) {
	const injected = "attacker-supplied-text-0123456789"
	q := url.Values{"state": {"s"}, "error": {injected}, "error_description": {injected}}
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://app.example.test/auth/callback?"+q.Encode(), nil)
	r.AddCookie(&http.Cookie{Name: flowCookie, Value: "s.n.v", Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})

	_, err := (&Entra{}).Finish(httptest.NewRecorder(), r)
	if !errors.Is(err, errEntraReturnedError) {
		t.Fatalf("got %v, want errEntraReturnedError", err)
	}
	if strings.Contains(err.Error(), injected) {
		t.Fatalf("error %q contains the callback's error text", err)
	}
}

// When Entra ID rejects the code exchange, the logged error keeps only the HTTP
// status and the OAuth error code, never the response body, which could repeat
// the authorization code (security final re-review 2026-11-05, finding 1).
func TestFinishLogsOnlyStatusAndCodeWhenExchangeFails(t *testing.T) {
	const authCode = "synthetic-authorization-code-0123456789"
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"The code ` + authCode + ` has expired.","trace_id":"body-only-marker"}`))
	}))
	t.Cleanup(tokenServer.Close)

	e := &Entra{oauth: oauth2.Config{ClientID: "c", Endpoint: oauth2.Endpoint{TokenURL: tokenServer.URL}}}
	q := url.Values{"state": {"s"}, "code": {authCode}}
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://app.example.test/auth/callback?"+q.Encode(), nil)
	r.AddCookie(&http.Cookie{Name: flowCookie, Value: "s.n.v", Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})

	_, err := e.Finish(httptest.NewRecorder(), r)
	if err == nil {
		t.Fatal("want an error from a rejected exchange")
	}
	msg := err.Error()
	for _, want := range []string{"HTTP 400", `"invalid_grant"`} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not contain %s", msg, want)
		}
	}
	for _, leak := range []string{authCode, "body-only-marker", "expired"} {
		if strings.Contains(msg, leak) {
			t.Errorf("error %q contains response body text %q", msg, leak)
		}
	}
}

// Users are identified by the tid and oid claims, never by email or
// preferred_username (security review 2026-11-05, finding 3).
func TestIdentityFromClaims(t *testing.T) {
	tests := []struct {
		name    string
		claims  idClaims
		wantErr string
	}{
		{"missing tid", idClaims{OID: testObject, Email: "a@synthetic.test"}, "tid"},
		{"missing oid", idClaims{TID: testTenant, Email: "a@synthetic.test"}, "oid"},
		{"email only", idClaims{Email: "a@synthetic.test", PreferredUsername: "a@synthetic.test"}, "tid"},
		{"oid not a GUID", idClaims{TID: testTenant, OID: "a@synthetic.test"}, "oid"},
		{"another tenant", idClaims{TID: "00000000-0000-4000-8000-0000000000cc", OID: testObject}, "another tenant"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := identityFromClaims(tc.claims, testTenant)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("got %v, want error mentioning %q", err, tc.wantErr)
			}
		})
	}

	id, err := identityFromClaims(idClaims{
		TID: strings.ToUpper(testTenant), OID: testObject, PreferredUsername: "A@Synthetic.Test",
	}, testTenant)
	if err != nil {
		t.Fatal(err)
	}
	if id.TenantID != testTenant || id.ObjectID != testObject || id.Email != "a@synthetic.test" {
		t.Fatalf("identity = %+v", id)
	}
}
