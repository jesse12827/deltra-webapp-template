// Package requestid gives every HTTP request an identifier that appears in
// logs, audit events, and user-facing error pages, so a problem a user reports
// can be traced (Web-App Development Stack v2.1, Sections 10.7 and 12).
package requestid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type ctxKey struct{}

// New returns a random 16-character hexadecimal identifier.
func New() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error on supported platforms
	return hex.EncodeToString(b)
}

// With returns a copy of ctx carrying id.
func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// From returns the request identifier in ctx, or "" if there is none.
func From(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}
