package synth

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"deltra/webapp/internal/items"
)

func TestItemsAreRepeatable(t *testing.T) {
	if !reflect.DeepEqual(Items(42, 50), Items(42, 50)) {
		t.Fatal("same seed must produce the same data")
	}
	if reflect.DeepEqual(Items(1, 50), Items(2, 50)) {
		t.Fatal("different seeds should produce different data")
	}
}

func TestItemsPassValidationAndCoverEdgeCases(t *testing.T) {
	var maxLen, wildcard, noDue bool
	for i, it := range Items(7, 100) {
		due := ""
		if it.DueOn != nil {
			due = it.DueOn.Format("2006-01-02")
		} else {
			noDue = true
		}
		in := items.Input{Title: it.Title, Description: it.Description, Status: it.Status, DueOn: due}
		if _, errs := in.Validate(); errs != nil {
			t.Fatalf("item %d fails validation: %v", i, errs)
		}
		if utf8.RuneCountInString(strings.TrimSpace(it.Title)) == 200 {
			maxLen = true
		}
		if strings.Contains(it.Title, "%") && strings.Contains(it.Title, "_") {
			wildcard = true
		}
	}
	if !maxLen || !wildcard || !noDue {
		t.Fatalf("edge cases missing: maxLen=%v wildcard=%v noDue=%v", maxLen, wildcard, noDue)
	}
}

func TestUsersAreSynthetic(t *testing.T) {
	var admins, inactive int
	for _, u := range Users() {
		if !strings.HasSuffix(u.Email, "@synthetic.test") {
			t.Errorf("non-synthetic email: %s", u.Email)
		}
		if u.Role == "admin" {
			admins++
		}
		if !u.Active {
			inactive++
		}
	}
	if admins != 1 || inactive != 1 {
		t.Fatalf("admins=%d inactive=%d", admins, inactive)
	}
}
