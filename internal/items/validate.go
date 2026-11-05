// Package items is the EXAMPLE DOMAIN. It demonstrates the standard pattern for
// a business area: server-side validation with field-level errors, a service
// that runs each change and its audit event in one transaction, and
// concurrent-edit protection. Copy its shape for the application's own domains.
package items

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Statuses an item may have. Mirrors the items_status_valid constraint.
var Statuses = []string{"open", "closed"}

// Input is the raw form input for creating or editing an item.
type Input struct {
	Title       string
	Description string
	Status      string
	DueOn       string // YYYY-MM-DD, or empty
}

// Clean is validated input, ready to store.
type Clean struct {
	Title       string
	Description string
	Status      string
	DueOn       *time.Time
}

// FieldErrors maps a form field name to a message the user can act on.
type FieldErrors map[string]string

// Validate checks every field and returns all problems at once, so the user
// can correct them in a single pass (Web-App Development Stack, Section 10.5).
func (in Input) Validate() (Clean, FieldErrors) {
	errs := FieldErrors{}
	c := Clean{
		Title:       strings.TrimSpace(in.Title),
		Description: strings.TrimSpace(in.Description),
		Status:      strings.TrimSpace(in.Status),
	}

	switch n := utf8.RuneCountInString(c.Title); {
	case n == 0:
		errs["title"] = "Enter a title."
	case n > 200:
		errs["title"] = "Title must be 200 characters or fewer."
	}
	if utf8.RuneCountInString(c.Description) > 4000 {
		errs["description"] = "Description must be 4,000 characters or fewer."
	}
	if c.Status == "" {
		c.Status = "open"
	}
	if !validStatus(c.Status) {
		errs["status"] = "Choose a status from the list."
	}
	if s := strings.TrimSpace(in.DueOn); s != "" {
		d, err := time.Parse(time.DateOnly, s)
		if err != nil {
			errs["due_on"] = "Enter the date as YYYY-MM-DD."
		} else {
			c.DueOn = &d
		}
	}

	if len(errs) > 0 {
		return Clean{}, errs
	}
	return c, nil
}

func validStatus(s string) bool {
	for _, v := range Statuses {
		if s == v {
			return true
		}
	}
	return false
}

// Sort keys accepted from the list page. Anything else falls back to the default.
var sortKeys = map[string]bool{"title": true, "-title": true, "due": true, "-due": true}

// NormalizeSort returns s if it is an allowed sort key, otherwise "" (most recently updated first).
func NormalizeSort(s string) string {
	if sortKeys[s] {
		return s
	}
	return ""
}

// PageSizes are the page sizes users may choose.
var PageSizes = []int{25, 50, 75, 100}

// NormalizePageSize returns n if it is an allowed page size, otherwise 25.
func NormalizePageSize(n int) int {
	for _, v := range PageSizes {
		if n == v {
			return n
		}
	}
	return PageSizes[0]
}

// likeEscape escapes LIKE wildcards so a search for "100%" matches the
// characters "100%" rather than everything beginning with "100".
func likeEscape(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}
