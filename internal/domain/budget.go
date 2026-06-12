// Package domain defines the core domain types and errors.
package domain

import (
	"strings"
	"time"
)

// Budget represents a monthly spending target for a category.
// Month and SourceMonth are always set to the first instant of their respective month (UTC).
type Budget struct {
	Category    string    `json:"category"`
	Month       time.Time `json:"month"`
	Amount      float64   `json:"amount"`
	Inherited   bool      `json:"inherited"`
	SourceMonth time.Time `json:"source_month"`
}

// BudgetFilter defines optional filters for listing budgets.
type BudgetFilter struct {
	Month    *time.Time
	Category *string
}

// Matches reports whether b passes all constraints in f.
func (b Budget) Matches(f BudgetFilter) bool {
	if f.Month != nil && (b.Month.Year() != f.Month.Year() || b.Month.Month() != f.Month.Month()) {
		return false
	}

	if f.Category != nil && !strings.EqualFold(b.Category, *f.Category) {
		return false
	}

	return true
}
