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

// Entra signs users in with Microsoft Entra ID through OpenID Connect, using the
// authorization code flow with PKCE (Web-App Development Stack v2.0, Section 10.1).
// Entra ID establishes who the person is; the application's users table decides
// whether they may use the application and in which role.
type Entra struct {
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
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
// verified email address, in lower case.
func (e *Entra) Finish(w http.ResponseWriter, r *http.Request) (string, error) {
	c, err := r.Cookie(flowCookie)
	if err != nil {
		return "", errors.New("sign-in flow cookie missing or expired")
	}
	http.SetCookie(w, &http.Cookie{Name: flowCookie, Value: "", Path: "/", MaxAge: -1, Secure: true, HttpOnly: true})

	parts := strings.Split(c.Value, ".")
	if len(parts) != 3 {
		return "", errors.New("sign-in flow cookie malformed")
	}
	state, nonce, verifier := parts[0], parts[1], parts[2]
	if r.URL.Query().Get("state") != state {
		return "", errors.New("sign-in state mismatch")
	}
	if msg := r.URL.Query().Get("error"); msg != "" {
		return "", fmt.Errorf("entra ID returned an error: %s", msg)
	}

	tok, err := e.oauth.Exchange(r.Context(), r.URL.Query().Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		return "", fmt.Errorf("exchanging authorization code: %w", err)
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok {
		return "", errors.New("no ID token in Entra ID response")
	}
	idt, err := e.verifier.Verify(r.Context(), raw)
	if err != nil {
		return "", fmt.Errorf("verifying ID token: %w", err)
	}
	if idt.Nonce != nonce {
		return "", errors.New("ID token nonce mismatch")
	}
	var claims struct {
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := idt.Claims(&claims); err != nil {
		return "", fmt.Errorf("reading ID token claims: %w", err)
	}
	email := claims.Email
	if email == "" {
		email = claims.PreferredUsername
	}
	if email == "" {
		return "", errors.New("ID token has no email or preferred_username claim")
	}
	return strings.ToLower(email), nil
}

func randomString() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
