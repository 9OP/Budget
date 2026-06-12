package domain

import "errors"

// ErrCategoryNotFound is returned when a referenced category does not exist.
var ErrCategoryNotFound = errors.New("category not found")

// ErrCategoryAlreadyExists is returned when creating a category with a duplicate name.
var ErrCategoryAlreadyExists = errors.New("category already exists")

// ErrInvalidMonth is returned when a month is zero or otherwise invalid.
var ErrInvalidMonth = errors.New("invalid month")

// ErrNegativeAmount is returned when an amount is negative.
var ErrNegativeAmount = errors.New("amount must be non-negative")

// ErrInvalidItemType is returned when the item type is not EXPENSE or INCOME.
var ErrInvalidItemType = errors.New("invalid item type, must be EXPENSE or INCOME")

// ErrMissingItemName is returned when an item name is empty.
var ErrMissingItemName = errors.New("item name is required")

// ErrMissingCategoryName is returned when a category name is empty.
var ErrMissingCategoryName = errors.New("category name is required")

// ErrItemNotFound is returned when no item matches the given ID.
var ErrItemNotFound = errors.New("item not found")

// ErrBudgetNotFound is returned when no budget matches the given category and month.
var ErrBudgetNotFound = errors.New("budget not found")

// ErrUnauthenticated is returned when a repository call lacks a user ID in context.
var ErrUnauthenticated = errors.New("unauthenticated")
