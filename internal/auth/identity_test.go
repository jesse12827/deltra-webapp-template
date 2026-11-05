package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"deltra/webapp/internal/auth"
	"deltra/webapp/internal/dbgen"
	"deltra/webapp/internal/testdb"
)

const (
	tenant      = "00000000-0000-4000-8000-0000000000aa"
	otherTenant = "00000000-0000-4000-8000-0000000000cc"
	objectA     = "00000000-0000-4000-8000-0000000000a1"
	objectB     = "00000000-0000-4000-8000-0000000000b1"
)

// FindUser matches on the tenant ID and object ID only, never on email
// (security review 2026-11-05, finding 3).
func TestFindUserMatchesOnTenantAndObjectID(t *testing.T) {
	ctx := context.Background()
	q := dbgen.New(testdb.New(t))
	mk := func(email string) dbgen.User {
		u, err := q.CreateUser(ctx, dbgen.CreateUserParams{Email: email, DisplayName: email, Role: "tech", Active: true})
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	linked := mk("linked@synthetic.test")
	unlinked := mk("unlinked@synthetic.test")
	if err := auth.LinkUser(ctx, q, linked.ID, auth.Identity{TenantID: tenant, ObjectID: objectA}); err != nil {
		t.Fatal(err)
	}

	got, err := auth.FindUser(ctx, q, auth.Identity{TenantID: tenant, ObjectID: objectA, Email: "someone-else@synthetic.test"})
	if err != nil || got.ID != linked.ID {
		t.Fatalf("matching tid and oid: got user %d, %v; want %d", got.ID, err, linked.ID)
	}

	misses := map[string]auth.Identity{
		"email of an unlinked user":  {TenantID: tenant, ObjectID: objectB, Email: unlinked.Email},
		"email of a linked user":     {TenantID: tenant, ObjectID: objectB, Email: linked.Email},
		"same oid in another tenant": {TenantID: otherTenant, ObjectID: objectA, Email: linked.Email},
	}
	for name, id := range misses {
		if _, err := auth.FindUser(ctx, q, id); !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("%s: got %v, want no user", name, err)
		}
	}
}

func TestEntraIdentityConstraints(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	q := dbgen.New(pool)
	var ids []int64
	for _, email := range []string{"one@synthetic.test", "two@synthetic.test"} {
		u, err := q.CreateUser(ctx, dbgen.CreateUserParams{Email: email, DisplayName: email, Role: "tech", Active: true})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, u.ID)
	}
	id := auth.Identity{TenantID: tenant, ObjectID: objectA}
	if err := auth.LinkUser(ctx, q, ids[0], id); err != nil {
		t.Fatal(err)
	}
	if err := auth.LinkUser(ctx, q, ids[1], id); err == nil {
		t.Error("two users linked to one Entra ID identity")
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET entra_oid = $1 WHERE id = $2", objectB, ids[1]); err == nil {
		t.Error("object ID stored without a tenant ID")
	}
}
