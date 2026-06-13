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
var ErrInvalidItemType = errors.New("invalid item type, must be EXPENSE, INCOME or INVESTMENT")

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

// ErrCategoryInUse is returned when deleting a category that still has items or budgets referencing it.
var ErrCategoryInUse = errors.New("category still has items or budgets referencing it")

// ErrCategoryLimitReached is returned when the user already has the maximum number of categories.
var ErrCategoryLimitReached = errors.New("category limit reached (max 100)")

// ErrItemsMonthLimitReached is returned when the user already has the maximum number of items for a given month.
var ErrItemsMonthLimitReached = errors.New("items per month limit reached (max 500)")

// ErrItemDateOutOfRange is returned when an item date is more than one year in the past or future.
var ErrItemDateOutOfRange = errors.New("item date must be within one year of today")

// ErrBudgetMonthOutOfRange is returned when a budget month is more than one year in the past or future.
var ErrBudgetMonthOutOfRange = errors.New("budget month must be within one year of today")
