// Package auth handles sign-in, server-side sessions, and authorization checks
// (Web-App Development Stack, Sections 10.1–10.3).
package auth

import "context"

// Role is an application role. Roles are stored on the users table.
type Role string

const (
	RoleAdmin Role = "admin"
	RoleTech  Role = "tech"
)

// User is the signed-in user for a request.
type User struct {
	ID          int64
	Email       string
	DisplayName string
	Role        Role
}

// IsAdmin reports whether the user holds the administrator role.
func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

// Has reports whether the user holds role. Administrators hold every role.
func (u User) Has(role Role) bool { return u.Role == role || u.Role == RoleAdmin }

type userKey struct{}

// WithUser returns a copy of ctx carrying the signed-in user.
func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, userKey{}, u)
}

// UserFrom returns the signed-in user in ctx, if any.
func UserFrom(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey{}).(User)
	return u, ok
}
