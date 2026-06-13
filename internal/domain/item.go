package domain

import (
	"strings"
	"time"
)

// ItemType represents the type of a financial item.
type ItemType string

// Expense, Income and Investment are the valid item types.
const (
	Expense    ItemType = "EXPENSE"
	Income     ItemType = "INCOME"
	Investment ItemType = "INVESTMENT"
)

// Item represents a financial transaction.
type Item struct {
	ID       string    `json:"id"`
	Type     ItemType  `json:"type"`
	Name     string    `json:"name"`
	Amount   float64   `json:"amount"`
	Date     time.Time `json:"date"`
	Category string    `json:"category"`
}

// ItemFilter defines optional filters for listing items.
type ItemFilter struct {
	From     *time.Time
	To       *time.Time
	Month    *time.Time
	Type     *ItemType
	Category *string
	Search   string // case-insensitive substring match on Name
}

// Matches reports whether item passes all constraints in f.
func (item Item) Matches(f ItemFilter) bool {
	if f.Month != nil {
		if item.Date.Year() != f.Month.Year() || item.Date.Month() != f.Month.Month() {
			return false
		}
	} else {
		if f.From != nil && item.Date.Before(*f.From) {
			return false
		}

		if f.To != nil && item.Date.After(*f.To) {
			return false
		}
	}

	if f.Type != nil && item.Type != *f.Type {
		return false
	}

	if f.Category != nil && !strings.EqualFold(item.Category, *f.Category) {
		return false
	}

	if f.Search != "" && !strings.Contains(strings.ToLower(item.Name), strings.ToLower(f.Search)) {
		return false
	}

	return true
}
