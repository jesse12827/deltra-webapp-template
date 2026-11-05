package web

import (
	"strings"
	"testing"

	"deltra/webapp/internal/auth"
)

// The denial recorded for an unlinked Entra ID account names the tenant and
// object IDs and never the email claim (security re-review 2026-11-05, finding 1).
func TestUnlinkedReasonNamesIDsNotEmail(t *testing.T) {
	id := auth.Identity{
		TenantID: "00000000-0000-4000-8000-0000000000aa",
		ObjectID: "00000000-0000-4000-8000-0000000000bb",
		Email:    "person@synthetic.test",
	}
	got := unlinkedReason(id)
	for _, want := range []string{id.TenantID, id.ObjectID} {
		if !strings.Contains(got, want) {
			t.Errorf("reason %q does not contain %q", got, want)
		}
	}
	if strings.Contains(got, id.Email) || strings.Contains(got, "synthetic.test") {
		t.Errorf("reason %q contains the email claim", got)
	}
}
