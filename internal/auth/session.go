package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"deltra/webapp/internal/dbgen"
)

// CookieName is the session cookie. The __Host- prefix makes browsers require
// Secure, Path=/, and no Domain attribute (Web-App Development Stack v2.0, Section 10.2).
const CookieName = "__Host-session"

// ErrNoSession means the request has no valid session.
var ErrNoSession = errors.New("no valid session")

// touchInterval limits how often last_seen_at is written for an active session.
const touchInterval = time.Minute

// Sessions stores server-side sessions in PostgreSQL.
type Sessions struct {
	q        *dbgen.Queries
	idle     time.Duration
	absolute time.Duration
	now      func() time.Time
}

// NewSessions returns a session store with the given idle and absolute lifetimes.
func NewSessions(db dbgen.DBTX, idle, absolute time.Duration) *Sessions {
	return &Sessions{q: dbgen.New(db), idle: idle, absolute: absolute, now: time.Now}
}

// WithTx returns a copy of the store that works inside tx, so that a session
// change and its audit event commit or fail together.
func (s *Sessions) WithTx(tx pgx.Tx) *Sessions {
	c := *s
	c.q = s.q.WithTx(tx)
	return &c
}

// Create starts a new session for userID and returns its token.
// Callers create a new session at every sign-in, so a token is never reused.
func (s *Sessions) Create(ctx context.Context, userID int64) (token string, expires time.Time, err error) {
	token, hash, err := NewToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires = s.now().Add(s.absolute)
	if err := s.q.CreateSession(ctx, dbgen.CreateSessionParams{TokenHash: hash, UserID: userID, ExpiresAt: expires}); err != nil {
		return "", time.Time{}, fmt.Errorf("creating session: %w", err)
	}
	return token, expires, nil
}

// Lookup returns the user for token. Expired, idle, and deactivated-user
// sessions are deleted and reported as ErrNoSession.
func (s *Sessions) Lookup(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrNoSession
	}
	hash := HashToken(token)
	row, err := s.q.GetSession(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNoSession
	} else if err != nil {
		return User{}, fmt.Errorf("reading session: %w", err)
	}
	now := s.now()
	if !row.Active || now.After(row.ExpiresAt) || now.Sub(row.LastSeenAt) > s.idle {
		if err := s.q.DeleteSession(ctx, hash); err != nil {
			return User{}, fmt.Errorf("deleting session: %w", err)
		}
		return User{}, ErrNoSession
	}
	if now.Sub(row.LastSeenAt) > touchInterval {
		if err := s.q.TouchSession(ctx, hash); err != nil {
			return User{}, fmt.Errorf("touching session: %w", err)
		}
	}
	return User{ID: row.UserID, Email: row.Email, DisplayName: row.DisplayName, Role: Role(row.Role)}, nil
}

// Delete ends the session for token.
func (s *Sessions) Delete(ctx context.Context, token string) error {
	return s.q.DeleteSession(ctx, HashToken(token))
}

// DeleteForUser ends every session of userID, for example when the user is deactivated.
func (s *Sessions) DeleteForUser(ctx context.Context, userID int64) error {
	return s.q.DeleteUserSessions(ctx, userID)
}

// DeleteExpired removes expired sessions. The server calls it periodically.
func (s *Sessions) DeleteExpired(ctx context.Context) error {
	return s.q.DeleteExpiredSessions(ctx)
}

// SetCookie writes the session cookie.
func SetCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: token, Path: "/", Expires: expires,
		Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}

// ClearCookie removes the session cookie from the browser.
func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: "", Path: "/", MaxAge: -1,
		Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}
