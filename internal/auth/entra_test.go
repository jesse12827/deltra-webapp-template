package auth

import (
	"strings"
	"testing"
)

const (
	testTenant = "00000000-0000-4000-8000-0000000000aa"
	testObject = "00000000-0000-4000-8000-0000000000bb"
)

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
