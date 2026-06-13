// Package service implements the application use-case layer for the budget service.
package service

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	"github.com/9op/budget/internal/domain"
)

const (
	maxCategories    = 100
	maxItemsPerMonth = 500
	dateRangeYears   = 1
	hoursPerDay      = 24
)

// CreateItemInput holds the input data required to create a new item.
type CreateItemInput struct {
	Type     domain.ItemType
	Name     string
	Amount   float64
	Date     time.Time
	Category string
}

// UpdateItemInput holds the fields that can be changed on an existing item.
type UpdateItemInput struct {
	Type     domain.ItemType
	Name     string
	Amount   float64
	Date     time.Time
	Category string
}

// Service implements the application use-case layer.
type Service struct {
	repo Repository
}

// NewService creates a new Service wired to the given repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateItem validates inputs, derives Month from Date, generates an ID, and persists the item.
func (s *Service) CreateItem(ctx context.Context, input CreateItemInput) (domain.Item, error) {
	if err := validateItemType(input.Type); err != nil {
		return domain.Item{}, err
	}

	if strings.TrimSpace(input.Name) == "" {
		return domain.Item{}, domain.ErrMissingItemName
	}

	if input.Amount < 0 {
		return domain.Item{}, domain.ErrNegativeAmount
	}

	if !withinDateRange(input.Date) {
		return domain.Item{}, domain.ErrItemDateOutOfRange
	}

	cats, err := s.repo.ListCategories(ctx)
	if err != nil {
		return domain.Item{}, err
	}

	if !categoryExists(cats, input.Category) {
		return domain.Item{}, domain.ErrCategoryNotFound
	}

	month := input.Date.UTC().Truncate(hoursPerDay * time.Hour)
	month = month.AddDate(0, 0, -month.Day()+1)

	existing, err := s.repo.ListItems(ctx, domain.ItemFilter{Month: &month})
	if err != nil {
		return domain.Item{}, err
	}

	if len(existing) >= maxItemsPerMonth {
		return domain.Item{}, domain.ErrItemsMonthLimitReached
	}

	item := domain.Item{
		Type:     input.Type,
		Name:     strings.TrimSpace(input.Name),
		Amount:   input.Amount,
		Date:     input.Date,
		Category: input.Category,
	}

	return s.repo.CreateItem(ctx, item)
}

// GetItem fetches a single item by its ID.
func (s *Service) GetItem(ctx context.Context, id string) (domain.Item, error) {
	return s.repo.GetItem(ctx, id)
}

// UpdateItem validates inputs and updates the item in the repository.
func (s *Service) UpdateItem(ctx context.Context, id string, input UpdateItemInput) (domain.Item, error) {
	if err := validateItemType(input.Type); err != nil {
		return domain.Item{}, err
	}

	if strings.TrimSpace(input.Name) == "" {
		return domain.Item{}, domain.ErrMissingItemName
	}

	if input.Amount < 0 {
		return domain.Item{}, domain.ErrNegativeAmount
	}

	if !withinDateRange(input.Date) {
		return domain.Item{}, domain.ErrItemDateOutOfRange
	}

	cats, err := s.repo.ListCategories(ctx)
	if err != nil {
		return domain.Item{}, err
	}

	if !categoryExists(cats, input.Category) {
		return domain.Item{}, domain.ErrCategoryNotFound
	}

	item := domain.Item{
		ID:       id,
		Type:     input.Type,
		Name:     strings.TrimSpace(input.Name),
		Amount:   input.Amount,
		Date:     input.Date,
		Category: input.Category,
	}

	return s.repo.UpdateItem(ctx, item)
}

// ListItems returns items matching the filter, sorted newest first.
func (s *Service) ListItems(ctx context.Context, filter domain.ItemFilter) ([]domain.Item, error) {
	items, err := s.repo.ListItems(ctx, filter)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(items, func(a, b domain.Item) int {
		return cmp.Compare(b.Date.Unix(), a.Date.Unix()) // descending
	})

	return items, nil
}

// DeleteItem removes the item with the given id from the repository.
func (s *Service) DeleteItem(ctx context.Context, id string) error {
	return s.repo.DeleteItem(ctx, id)
}

// CreateCategory normalises the name, checks for duplicates, then persists the category.
func (s *Service) CreateCategory(ctx context.Context, name string) (domain.Category, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return domain.Category{}, domain.ErrMissingCategoryName
	}

	cats, err := s.repo.ListCategories(ctx)
	if err != nil {
		return domain.Category{}, err
	}

	if len(cats) >= maxCategories {
		return domain.Category{}, domain.ErrCategoryLimitReached
	}

	if categoryExists(cats, trimmed) {
		return domain.Category{}, domain.ErrCategoryAlreadyExists
	}

	return s.repo.CreateCategory(ctx, domain.Category{Name: trimmed})
}

// ListCategories returns all known categories.
func (s *Service) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return s.repo.ListCategories(ctx)
}

// DeleteCategory removes the category with the given name.
func (s *Service) DeleteCategory(ctx context.Context, name string) error {
	cats, err := s.repo.ListCategories(ctx)
	if err != nil {
		return err
	}

	if !categoryExists(cats, name) {
		return domain.ErrCategoryNotFound
	}

	return s.repo.DeleteCategory(ctx, name)
}

// RenameCategory changes the name of an existing category and cascades the rename.
func (s *Service) RenameCategory(ctx context.Context, oldName, newName string) (domain.Category, error) {
	trimmed := strings.TrimSpace(newName)
	if trimmed == "" {
		return domain.Category{}, domain.ErrMissingCategoryName
	}

	cats, err := s.repo.ListCategories(ctx)
	if err != nil {
		return domain.Category{}, err
	}

	if !categoryExists(cats, oldName) {
		return domain.Category{}, domain.ErrCategoryNotFound
	}

	if !strings.EqualFold(oldName, trimmed) && categoryExists(cats, trimmed) {
		return domain.Category{}, domain.ErrCategoryAlreadyExists
	}

	if err := s.repo.RenameCategory(ctx, oldName, trimmed); err != nil {
		return domain.Category{}, err
	}

	return domain.Category{Name: trimmed}, nil
}

// SetBudget validates the category and month, then upserts the budget.
func (s *Service) SetBudget(ctx context.Context, budget domain.Budget) (domain.Budget, error) {
	if budget.Month.IsZero() {
		return domain.Budget{}, domain.ErrInvalidMonth
	}

	if budget.Amount < 0 {
		return domain.Budget{}, domain.ErrNegativeAmount
	}

	if !withinDateRange(budget.Month) {
		return domain.Budget{}, domain.ErrBudgetMonthOutOfRange
	}

	cats, err := s.repo.ListCategories(ctx)
	if err != nil {
		return domain.Budget{}, err
	}

	if !categoryExists(cats, budget.Category) {
		return domain.Budget{}, domain.ErrCategoryNotFound
	}

	stored := domain.Budget{
		Category: budget.Category,
		Month:    budget.Month,
		Amount:   budget.Amount,
	}

	return s.repo.SetBudget(ctx, stored)
}

// DeleteBudget removes the budget for the given category and month.
func (s *Service) DeleteBudget(ctx context.Context, category string, month time.Time) error {
	return s.repo.DeleteBudget(ctx, category, month)
}

// ListBudgets returns the effective budget per category.
// When filter.Month is set, carry-forward logic is applied.
// When filter.Month is nil, raw stored budgets are returned without inheritance.
func (s *Service) ListBudgets(ctx context.Context, filter domain.BudgetFilter) ([]domain.Budget, error) {
	if filter.Month == nil {
		return s.repo.ListBudgets(ctx, filter)
	}

	return s.resolveEffectiveBudgets(ctx, filter)
}

func (s *Service) resolveEffectiveBudgets(ctx context.Context, filter domain.BudgetFilter) ([]domain.Budget, error) {
	requestedMonth := *filter.Month

	cats, err := s.repo.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	allBudgets, err := s.repo.ListBudgets(ctx, domain.BudgetFilter{})
	if err != nil {
		return nil, err
	}

	result := make([]domain.Budget, 0, len(cats))

	for _, cat := range cats {
		if filter.Category != nil && !strings.EqualFold(cat.Name, *filter.Category) {
			continue
		}

		effective, found := findEffectiveBudget(allBudgets, cat.Name, requestedMonth)
		if !found {
			continue
		}

		result = append(result, effective)
	}

	return result, nil
}

// --- Helpers ---

func withinDateRange(t time.Time) bool {
	now := time.Now()
	earliest := now.AddDate(-dateRangeYears, 0, 0)
	latest := now.AddDate(dateRangeYears, 0, 0)

	return !t.Before(earliest) && !t.After(latest)
}

func validateItemType(t domain.ItemType) error {
	switch t {
	case domain.Expense, domain.Income:
		return nil
	default:
		return domain.ErrInvalidItemType
	}
}

func categoryExists(cats []domain.Category, name string) bool {
	for _, c := range cats {
		if strings.EqualFold(c.Name, name) {
			return true
		}
	}

	return false
}

func findEffectiveBudget(budgets []domain.Budget, category string, requestedMonth time.Time) (domain.Budget, bool) {
	var best domain.Budget

	found := false

	for _, b := range budgets {
		if !strings.EqualFold(b.Category, category) {
			continue
		}

		if b.Month.After(requestedMonth) {
			continue
		}

		if !found || b.Month.After(best.Month) {
			best = b
			found = true
		}
	}

	if !found {
		return domain.Budget{}, false
	}

	result := domain.Budget{
		Category: category,
		Month:    requestedMonth,
		Amount:   best.Amount,
	}

	if best.Month.Before(requestedMonth) {
		result.Inherited = true
		result.SourceMonth = best.Month
	}

	return result, true
}
