package items

import (
	"strings"
	"testing"
)

func TestValidateAcceptsGoodInput(t *testing.T) {
	c, errs := Input{Title: "  Laptop for front desk ", Status: "open", DueOn: "2026-12-01"}.Validate()
	if errs != nil {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if c.Title != "Laptop for front desk" {
		t.Errorf("title not trimmed: %q", c.Title)
	}
	if c.DueOn == nil || c.DueOn.Format("2006-01-02") != "2026-12-01" {
		t.Errorf("due date = %v", c.DueOn)
	}
}

func TestValidateDefaultsStatusToOpen(t *testing.T) {
	c, errs := Input{Title: "x"}.Validate()
	if errs != nil || c.Status != "open" {
		t.Fatalf("got %q, %v", c.Status, errs)
	}
}

func TestValidateReportsEveryProblem(t *testing.T) {
	_, errs := Input{
		Title:       "",
		Description: strings.Repeat("a", 4001),
		Status:      "deleted",
		DueOn:       "12/01/2026",
	}.Validate()
	for _, field := range []string{"title", "description", "status", "due_on"} {
		if errs[field] == "" {
			t.Errorf("missing error for %s", field)
		}
	}
}

func TestValidateCountsCharactersNotBytes(t *testing.T) {
	// 200 multi-byte characters is within the limit even though it exceeds 200 bytes.
	_, errs := Input{Title: strings.Repeat("é", 200)}.Validate()
	if errs != nil {
		t.Fatalf("unexpected errors: %v", errs)
	}
	_, errs = Input{Title: strings.Repeat("é", 201)}.Validate()
	if errs["title"] == "" {
		t.Fatal("201 characters must be rejected")
	}
}

func TestLikeEscape(t *testing.T) {
	cases := map[string]string{
		"100%":     `100\%`,
		"a_b":      `a\_b`,
		`back\sl`:  `back\\sl`,
		"ordinary": "ordinary",
	}
	for in, want := range cases {
		if got := likeEscape(in); got != want {
			t.Errorf("likeEscape(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeSortAndPageSize(t *testing.T) {
	if NormalizeSort("title; DROP TABLE items") != "" {
		t.Error("unknown sort key must be rejected")
	}
	if NormalizeSort("-due") != "-due" {
		t.Error("known sort key must be kept")
	}
	if NormalizePageSize(1000) != 25 || NormalizePageSize(75) != 75 {
		t.Error("page size normalization")
	}
}
