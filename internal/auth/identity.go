package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"deltra/webapp/internal/dbgen"
)

// FindUser returns the application user linked to an Entra ID identity. It
// matches only on the tenant ID and object ID, never on email, because email
// and preferred_username can be changed by directory administrators (Phase 0
// security review, 2026-11-05, finding 3). It returns pgx.ErrNoRows
// when no user is linked to the identity.
func FindUser(ctx context.Context, q *dbgen.Queries, id Identity) (dbgen.User, error) {
	tid, err := uuidParam(id.TenantID)
	if err != nil {
		return dbgen.User{}, fmt.Errorf("tenant ID: %w", err)
	}
	oid, err := uuidParam(id.ObjectID)
	if err != nil {
		return dbgen.User{}, fmt.Errorf("object ID: %w", err)
	}
	return q.GetUserByEntraID(ctx, dbgen.GetUserByEntraIDParams{EntraTid: tid, EntraOid: oid})
}

// LinkUser records the Entra ID identity of an application user. Linking
// decides who can sign in as the user, so a caller must run it in the same
// transaction as an audit event (for example "user.link_entra"), passing the
// transaction's Queries. Linking replaces any earlier link.
func LinkUser(ctx context.Context, q *dbgen.Queries, userID int64, id Identity) error {
	tid, err := uuidParam(id.TenantID)
	if err != nil {
		return fmt.Errorf("tenant ID: %w", err)
	}
	oid, err := uuidParam(id.ObjectID)
	if err != nil {
		return fmt.Errorf("object ID: %w", err)
	}
	return q.SetUserEntraID(ctx, dbgen.SetUserEntraIDParams{ID: userID, EntraTid: tid, EntraOid: oid})
}

// parseGUID returns s in canonical lower-case form, or an error if s is not a GUID.
func parseGUID(s string) (string, error) {
	u, err := uuidParam(s)
	if err != nil {
		return "", err
	}
	v, err := u.Value()
	if err != nil {
		return "", err
	}
	str, _ := v.(string)
	return str, nil
}

func uuidParam(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if strings.TrimSpace(s) == "" {
		return u, errors.New("missing")
	}
	if err := u.Scan(strings.TrimSpace(s)); err != nil {
		return u, fmt.Errorf("not a GUID: %q", s)
	}
	return u, nil
}
