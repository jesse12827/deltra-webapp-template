package auth

import (
	"bytes"
	"testing"
)

func TestNewTokenIsRandomAndHashed(t *testing.T) {
	t1, h1, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	t2, h2, _ := NewToken()
	if t1 == t2 || bytes.Equal(h1, h2) {
		t.Fatal("tokens must differ")
	}
	if len(h1) != 32 {
		t.Fatalf("hash length = %d, want 32 (matches sessions_hash_length constraint)", len(h1))
	}
	if !bytes.Equal(HashToken(t1), h1) {
		t.Fatal("HashToken must reproduce the stored hash")
	}
	if len(t1) < 43 {
		t.Fatalf("token too short: %d characters", len(t1))
	}
}

func TestAdminHoldsEveryRole(t *testing.T) {
	admin := User{Role: RoleAdmin}
	tech := User{Role: RoleTech}
	if !admin.Has(RoleTech) || !admin.Has(RoleAdmin) {
		t.Error("admin must hold every role")
	}
	if tech.Has(RoleAdmin) {
		t.Error("tech must not hold admin")
	}
}
