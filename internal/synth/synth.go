// Package synth defines synthetic data for development, acceptance testing, and
// automated tests (AI Policy Section 26A.7; Web-App Development Stack,
// Section 13.1). Output depends only on the seed, so results are repeatable.
// No value here may be copied from a real client, staff member, or production system.
package synth

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// User is a synthetic application user.
type User struct {
	Email, DisplayName, Role string
	Active                   bool
}

// Item is a synthetic example-domain item.
type Item struct {
	Title, Description, Status string
	DueOn                      *time.Time
}

// Users returns the fixed set of synthetic users: one administrator, three
// techs, and one inactive user. All addresses use the reserved .test domain.
func Users() []User {
	return []User{
		{"admin@synthetic.test", "Synthetic Admin", "admin", true},
		{"tech.one@synthetic.test", "Synthetic Tech One", "tech", true},
		{"tech.two@synthetic.test", "Synthetic Tech Two", "tech", true},
		{"tech.three@synthetic.test", "Synthetic Tech Three", "tech", true},
		{"inactive@synthetic.test", "Synthetic Inactive User", "tech", false},
	}
}

var (
	nouns   = []string{"Laptop", "Firewall", "Switch", "License", "Monitor", "Printer", "Access point", "UPS", "Server", "Docking station"}
	purpose = []string{"for front desk", "for branch office", "replacement", "renewal", "upgrade", "for new hire", "spare", "for training room"}
)

// Items returns n synthetic items generated from seed. The set deliberately
// includes edge cases: empty descriptions, no due date, maximum-length titles,
// and titles containing LIKE wildcard characters.
func Items(seed uint64, n int) []Item {
	r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)) //nolint:gosec // seeded and repeatable by design; not used for security
	base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	out := make([]Item, 0, n)
	for i := range n {
		it := Item{
			Title:  fmt.Sprintf("Synthetic %s %s #%04d", nouns[r.IntN(len(nouns))], purpose[r.IntN(len(purpose))], i+1),
			Status: []string{"open", "open", "closed"}[r.IntN(3)],
		}
		switch i % 10 {
		case 0: // no description, no due date
		case 1:
			it.Title = fmt.Sprintf("%-200.200s", fmt.Sprintf("Synthetic maximum-length title #%04d ", i+1)+longPad)
		case 2:
			it.Title = fmt.Sprintf("Synthetic 100%% coverage_item #%04d", i+1)
			it.Description = "Title contains LIKE wildcard characters."
		default:
			it.Description = fmt.Sprintf("Synthetic description for item %d.", i+1)
			d := base.AddDate(0, 0, r.IntN(365))
			it.DueOn = &d
		}
		out = append(out, it)
	}
	return out
}

const longPad = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
