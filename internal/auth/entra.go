package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"deltra/webapp/internal/config"
)

// flowCookie holds the state, nonce, and PKCE verifier for one sign-in attempt.
const flowCookie = "__Host-signin-flow"

// errEntraReturnedError is returned when the callback carries an error
// parameter instead of an authorization code.
var errEntraReturnedError = errors.New("entra ID returned an error instead of an authorization code")

// Entra signs users in with Microsoft Entra ID through OpenID Connect, using the
// authorization code flow with PKCE (Web-App Development Stack, Section 10.1).
// Entra ID establishes who the person is; the application's users table decides
// whether they may use the application and in which role.
type Entra struct {
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
	tenantID string
}

// Identity is a person's Entra ID account, from a verified ID token.
// TenantID and ObjectID together identify the account and never change;
// Email is informational only and is never used to find a user.
type Identity struct {
	TenantID string // tid claim, a GUID
	ObjectID string // oid claim, a GUID
	Email    string // email or preferred_username claim, lower case; may be empty
}

// NewEntra discovers the tenant's OpenID Connect configuration.
func NewEntra(ctx context.Context, c config.EntraConfig) (*Entra, error) {
	issuer := "https://login.microsoftonline.com/" + c.TenantID + "/v2.0"
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discovering Entra ID tenant: %w", err)
	}
	return &Entra{
		oauth: oauth2.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			RedirectURL:  c.RedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: c.ClientID}),
		tenantID: c.TenantID,
	}, nil
}

// Begin starts a sign-in: it stores the flow values in a short-lived cookie and
// returns the Entra ID URL to redirect the browser to.
func (e *Entra) Begin(w http.ResponseWriter) (string, error) {
	state, err := randomString()
	if err != nil {
		return "", err
	}
	nonce, err := randomString()
	if err != nil {
		return "", err
	}
	verifier := oauth2.GenerateVerifier()
	http.SetCookie(w, &http.Cookie{
		Name: flowCookie, Value: state + "." + nonce + "." + verifier, Path: "/",
		MaxAge: int((10 * time.Minute).Seconds()), Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	return e.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), nil
}

// Finish completes a sign-in from the callback request and returns the
// verified Entra ID identity.
func (e *Entra) Finish(w http.ResponseWriter, r *http.Request) (Identity, error) {
	c, err := r.Cookie(flowCookie)
	if err != nil {
		return Identity{}, errors.New("sign-in flow cookie missing or expired")
	}
	http.SetCookie(w, &http.Cookie{Name: flowCookie, Value: "", Path: "/", MaxAge: -1, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})

	parts := strings.Split(c.Value, ".")
	if len(parts) != 3 {
		return Identity{}, errors.New("sign-in flow cookie malformed")
	}
	state, nonce, verifier := parts[0], parts[1], parts[2]
	if r.URL.Query().Get("state") != state {
		return Identity{}, errors.New("sign-in state mismatch")
	}
	if r.URL.Query().Get("error") != "" {
		// The callback is public, so the error parameter is untrusted text.
		// It is not copied into the error, which the caller logs.
		return Identity{}, errEntraReturnedError
	}

	tok, err := e.oauth.Exchange(r.Context(), r.URL.Query().Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		return Identity{}, exchangeError(err)
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok {
		return Identity{}, errors.New("no ID token in Entra ID response")
	}
	idt, err := e.verifier.Verify(r.Context(), raw)
	if err != nil {
		return Identity{}, fmt.Errorf("verifying ID token: %w", err)
	}
	if idt.Nonce != nonce {
		return Identity{}, errors.New("ID token nonce mismatch")
	}
	var claims idClaims
	if err := idt.Claims(&claims); err != nil {
		return Identity{}, fmt.Errorf("reading ID token claims: %w", err)
	}
	return identityFromClaims(claims, e.tenantID)
}

// idClaims are the ID token claims the application reads.
type idClaims struct {
	TID               string `json:"tid"`
	OID               string `json:"oid"`
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
}

// identityFromClaims checks that the token carries a tenant ID and object ID,
// and that the tenant is the application's own.
func identityFromClaims(c idClaims, tenantID string) (Identity, error) {
	tid, err := parseGUID(c.TID)
	if err != nil {
		return Identity{}, fmt.Errorf("ID token tid claim: %w", err)
	}
	oid, err := parseGUID(c.OID)
	if err != nil {
		return Identity{}, fmt.Errorf("ID token oid claim: %w", err)
	}
	want, err := parseGUID(tenantID)
	if err != nil {
		return Identity{}, fmt.Errorf("configured tenant ID: %w", err)
	}
	if tid != want {
		return Identity{}, errors.New("ID token is from another tenant")
	}
	email := c.Email
	if email == "" {
		email = c.PreferredUsername
	}
	return Identity{TenantID: tid, ObjectID: oid, Email: strings.ToLower(email)}, nil
}

// exchangeError describes a failed authorization-code exchange for the log.
// When Entra ID rejects the exchange, it keeps only the HTTP status and the
// OAuth error code, never the response body, which could repeat the code
// (Stack Section 12). Other failures, such as
// network errors, carry no response body and are wrapped unchanged.
func exchangeError(err error) error {
	var re *oauth2.RetrieveError
	if !errors.As(err, &re) {
		return fmt.Errorf("exchanging authorization code: %w", err)
	}
	status := 0
	if re.Response != nil {
		status = re.Response.StatusCode
	}
	code := re.ErrorCode
	if len(code) > 64 {
		code = code[:64]
	}
	return fmt.Errorf("exchanging authorization code: Entra ID returned HTTP %d, error code %q", status, code)
}

func randomString() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
