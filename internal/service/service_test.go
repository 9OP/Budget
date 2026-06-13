package service_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/9op/budget/internal/domain"
	"github.com/9op/budget/internal/service"
)

// month returns a time.Time at the first instant of the given year/month (UTC).
func month(year int, m time.Month) time.Time {
	return time.Date(year, m, 1, 0, 0, 0, 0, time.UTC)
}

// --- fakeRepo ---

type fakeRepo struct {
	items      []domain.Item
	categories []domain.Category
	budgets    []domain.Budget
}

func (f *fakeRepo) CreateItem(_ context.Context, item domain.Item) (domain.Item, error) {
	// Simulate what the store does: assign the sheet row number as the ID.
	item.ID = strconv.Itoa(len(f.items) + 2) // rows start at 2 (row 1 is header)
	f.items = append(f.items, item)

	return item, nil
}

func (f *fakeRepo) GetItem(_ context.Context, id string) (domain.Item, error) {
	for _, item := range f.items {
		if item.ID == id {
			return item, nil
		}
	}

	return domain.Item{}, domain.ErrItemNotFound
}

func (f *fakeRepo) UpdateItem(_ context.Context, item domain.Item) (domain.Item, error) {
	for i, it := range f.items {
		if it.ID == item.ID {
			f.items[i] = item

			return item, nil
		}
	}

	return domain.Item{}, domain.ErrItemNotFound
}

func (f *fakeRepo) ListItems(_ context.Context, filter domain.ItemFilter) ([]domain.Item, error) {
	result := make([]domain.Item, 0, len(f.items))

	for _, item := range f.items {
		if item.Matches(filter) {
			result = append(result, item)
		}
	}

	return result, nil
}

func (f *fakeRepo) DeleteItem(_ context.Context, id string) error {
	for i, item := range f.items {
		if item.ID == id {
			f.items = append(f.items[:i], f.items[i+1:]...)

			return nil
		}
	}

	return domain.ErrItemNotFound
}

func (f *fakeRepo) CreateCategory(_ context.Context, cat domain.Category) (domain.Category, error) {
	for _, c := range f.categories {
		if strings.EqualFold(c.Name, cat.Name) {
			return domain.Category{}, domain.ErrCategoryAlreadyExists
		}
	}

	f.categories = append(f.categories, cat)

	return cat, nil
}

func (f *fakeRepo) ListCategories(_ context.Context) ([]domain.Category, error) {
	return f.categories, nil
}

func (f *fakeRepo) DeleteCategory(_ context.Context, name string) error {
	for i, c := range f.categories {
		if strings.EqualFold(c.Name, name) {
			f.categories = append(f.categories[:i], f.categories[i+1:]...)

			return nil
		}
	}

	return domain.ErrCategoryNotFound
}

func (f *fakeRepo) RenameCategory(_ context.Context, oldName, newName string) error {
	for i, c := range f.categories {
		if strings.EqualFold(c.Name, oldName) {
			f.categories[i].Name = newName

			for j, item := range f.items {
				if strings.EqualFold(item.Category, oldName) {
					f.items[j].Category = newName
				}
			}

			for j, b := range f.budgets {
				if strings.EqualFold(b.Category, oldName) {
					f.budgets[j].Category = newName
				}
			}

			return nil
		}
	}

	return domain.ErrCategoryNotFound
}

func (f *fakeRepo) SetBudget(_ context.Context, budget domain.Budget) (domain.Budget, error) {
	for i, b := range f.budgets {
		if strings.EqualFold(b.Category, budget.Category) && b.Month.Year() == budget.Month.Year() &&
			b.Month.Month() == budget.Month.Month() {
			f.budgets[i] = budget

			return budget, nil
		}
	}

	f.budgets = append(f.budgets, budget)

	return budget, nil
}

func (f *fakeRepo) DeleteBudget(_ context.Context, category string, mon time.Time) error {
	for i, b := range f.budgets {
		if strings.EqualFold(b.Category, category) && b.Month.Year() == mon.Year() && b.Month.Month() == mon.Month() {
			f.budgets = append(f.budgets[:i], f.budgets[i+1:]...)

			return nil
		}
	}

	return domain.ErrBudgetNotFound
}

func (f *fakeRepo) ListBudgets(_ context.Context, filter domain.BudgetFilter) ([]domain.Budget, error) {
	result := make([]domain.Budget, 0, len(f.budgets))

	for _, b := range f.budgets {
		if b.Matches(filter) {
			result = append(result, b)
		}
	}

	return result, nil
}

// --- Item tests ---

func TestCreateItem_Success(t *testing.T) {
	repo := &fakeRepo{categories: []domain.Category{{Name: "food"}}}
	svc := service.NewService(repo)

	now := time.Now().UTC()
	item, err := svc.CreateItem(context.Background(), service.CreateItemInput{
		Type:     domain.Expense,
		Name:     "Groceries",
		Amount:   42.5,
		Date:     now,
		Category: "food",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if item.ID == "" {
		t.Error("expected non-empty ID")
	}

	if item.Date.Year() != now.Year() || item.Date.Month() != now.Month() {
		t.Errorf("expected date in %d-%02d, got %v", now.Year(), now.Month(), item.Date)
	}
}

func TestCreateItem_CategoryNotFound(t *testing.T) {
	svc := service.NewService(&fakeRepo{})

	_, err := svc.CreateItem(context.Background(), service.CreateItemInput{
		Type:     domain.Expense,
		Name:     "Groceries",
		Amount:   42.5,
		Date:     time.Date(2025, 5, 28, 0, 0, 0, 0, time.UTC),
		Category: "nonexistent",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateItem_DatePreserved(t *testing.T) {
	repo := &fakeRepo{categories: []domain.Category{{Name: "food"}}}
	svc := service.NewService(repo)

	date := time.Now().UTC().Truncate(24 * time.Hour)
	item, err := svc.CreateItem(context.Background(), service.CreateItemInput{
		Type:     domain.Income,
		Name:     "Salary",
		Amount:   1000,
		Date:     date,
		Category: "food",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !item.Date.Equal(date) {
		t.Errorf("expected date %v, got %v", date, item.Date)
	}
}

func TestListItems_NoFilter(t *testing.T) {
	repo := &fakeRepo{
		categories: []domain.Category{{Name: "food"}},
		items: []domain.Item{
			{ID: "1", Type: domain.Expense, Name: "a", Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			{ID: "2", Type: domain.Income, Name: "b", Date: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	svc := service.NewService(repo)

	items, err := svc.ListItems(context.Background(), domain.ItemFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestListItems_FilterByMonth(t *testing.T) {
	m := month(2025, time.January)
	repo := &fakeRepo{
		items: []domain.Item{
			{ID: "1", Date: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)},
			{ID: "2", Date: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	svc := service.NewService(repo)

	items, err := svc.ListItems(context.Background(), domain.ItemFilter{Month: &m})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 1 || items[0].ID != "1" {
		t.Errorf("expected 1 item with ID 1, got %v", items)
	}
}

func TestListItems_FilterByType(t *testing.T) {
	itemType := domain.Expense
	repo := &fakeRepo{
		items: []domain.Item{
			{ID: "1", Type: domain.Expense},
			{ID: "2", Type: domain.Income},
		},
	}
	svc := service.NewService(repo)

	items, err := svc.ListItems(context.Background(), domain.ItemFilter{Type: &itemType})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 1 || items[0].ID != "1" {
		t.Errorf("expected 1 expense item, got %v", items)
	}
}

func TestListItems_FilterByDateRange(t *testing.T) {
	from := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 5, 31, 0, 0, 0, 0, time.UTC)

	repo := &fakeRepo{
		items: []domain.Item{
			{ID: "1", Date: time.Date(2025, 5, 10, 0, 0, 0, 0, time.UTC)},
			{ID: "2", Date: time.Date(2025, 4, 30, 0, 0, 0, 0, time.UTC)},
			{ID: "3", Date: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	svc := service.NewService(repo)

	items, err := svc.ListItems(context.Background(), domain.ItemFilter{From: &from, To: &to})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 1 || items[0].ID != "1" {
		t.Errorf("expected 1 item in range, got %v", items)
	}
}

// --- Category tests ---

func TestCreateCategory_Success(t *testing.T) {
	svc := service.NewService(&fakeRepo{})

	cat, err := svc.CreateCategory(context.Background(), "food")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cat.Name != "food" {
		t.Errorf("expected name food, got %s", cat.Name)
	}
}

func TestCreateCategory_DuplicateCaseInsensitive(t *testing.T) {
	repo := &fakeRepo{categories: []domain.Category{{Name: "food"}}}
	svc := service.NewService(repo)

	_, err := svc.CreateCategory(context.Background(), "Food")
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
}

// --- Budget tests ---

func TestSetBudget_CategoryNotFound(t *testing.T) {
	svc := service.NewService(&fakeRepo{})

	_, err := svc.SetBudget(context.Background(), domain.Budget{
		Category: "food", Month: month(2025, time.May), Amount: 300,
	})

	if err == nil {
		t.Fatal("expected error for missing category")
	}
}

func TestSetBudget_InvalidMonth(t *testing.T) {
	repo := &fakeRepo{categories: []domain.Category{{Name: "food"}}}
	svc := service.NewService(repo)

	_, err := svc.SetBudget(context.Background(), domain.Budget{
		Category: "food", Month: time.Time{}, Amount: 300,
	})

	if err == nil {
		t.Fatal("expected error for zero month")
	}
}

func TestSetBudget_Upsert(t *testing.T) {
	repo := &fakeRepo{categories: []domain.Category{{Name: "food"}}}
	svc := service.NewService(repo)

	now := time.Now().UTC()
	currentMonth := month(now.Year(), now.Month())

	_, err := svc.SetBudget(context.Background(), domain.Budget{
		Category: "food", Month: currentMonth, Amount: 300,
	})
	if err != nil {
		t.Fatalf("first set: %v", err)
	}

	updated, err := svc.SetBudget(context.Background(), domain.Budget{
		Category: "food", Month: currentMonth, Amount: 400,
	})
	if err != nil {
		t.Fatalf("second set: %v", err)
	}

	if updated.Amount != 400 {
		t.Errorf("expected amount 400, got %f", updated.Amount)
	}

	budgets, err := svc.ListBudgets(context.Background(), domain.BudgetFilter{})
	if err != nil {
		t.Fatalf("list budgets: %v", err)
	}

	if len(budgets) != 1 {
		t.Errorf("expected 1 budget after upsert, got %d", len(budgets))
	}
}

func TestListBudgets_WithMonth_Explicit(t *testing.T) {
	m := month(2025, time.May)
	repo := &fakeRepo{
		categories: []domain.Category{{Name: "food"}},
		budgets:    []domain.Budget{{Category: "food", Month: month(2025, time.May), Amount: 300}},
	}
	svc := service.NewService(repo)

	budgets, err := svc.ListBudgets(context.Background(), domain.BudgetFilter{Month: &m})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(budgets) != 1 {
		t.Fatalf("expected 1 budget, got %d", len(budgets))
	}

	if budgets[0].Inherited {
		t.Error("expected inherited=false for explicit budget")
	}
}

func TestListBudgets_WithMonth_CarryForward(t *testing.T) {
	m := month(2025, time.May)
	repo := &fakeRepo{
		categories: []domain.Category{{Name: "food"}},
		budgets:    []domain.Budget{{Category: "food", Month: month(2025, time.January), Amount: 200}},
	}
	svc := service.NewService(repo)

	budgets, err := svc.ListBudgets(context.Background(), domain.BudgetFilter{Month: &m})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(budgets) != 1 {
		t.Fatalf("expected 1 budget via carry-forward, got %d", len(budgets))
	}

	if !budgets[0].Inherited {
		t.Error("expected inherited=true for carry-forward budget")
	}

	if budgets[0].SourceMonth != month(2025, time.January) {
		t.Errorf("expected source_month 2025-01, got %v", budgets[0].SourceMonth)
	}

	if budgets[0].Month != month(2025, time.May) {
		t.Errorf("expected month 2025-05, got %v", budgets[0].Month)
	}
}

func TestListBudgets_WithMonth_PicksMostRecent(t *testing.T) {
	m := month(2025, time.May)
	repo := &fakeRepo{
		categories: []domain.Category{{Name: "food"}},
		budgets: []domain.Budget{
			{Category: "food", Month: month(2025, time.January), Amount: 100},
			{Category: "food", Month: month(2025, time.March), Amount: 200},
		},
	}
	svc := service.NewService(repo)

	budgets, err := svc.ListBudgets(context.Background(), domain.BudgetFilter{Month: &m})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(budgets) != 1 {
		t.Fatalf("expected 1 budget, got %d", len(budgets))
	}

	if budgets[0].Amount != 200 {
		t.Errorf("expected amount 200 (most recent), got %f", budgets[0].Amount)
	}

	if budgets[0].SourceMonth != month(2025, time.March) {
		t.Errorf("expected source_month 2025-03, got %v", budgets[0].SourceMonth)
	}
}

func TestListBudgets_WithMonth_ExplicitOverridesPrior(t *testing.T) {
	m := month(2025, time.May)
	repo := &fakeRepo{
		categories: []domain.Category{{Name: "food"}},
		budgets: []domain.Budget{
			{Category: "food", Month: month(2025, time.January), Amount: 100},
			{Category: "food", Month: month(2025, time.May), Amount: 300},
		},
	}
	svc := service.NewService(repo)

	budgets, err := svc.ListBudgets(context.Background(), domain.BudgetFilter{Month: &m})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(budgets) != 1 || budgets[0].Amount != 300 {
		t.Errorf("expected explicit budget with amount 300, got %v", budgets)
	}

	if budgets[0].Inherited {
		t.Error("expected inherited=false for explicit override")
	}
}

func TestListBudgets_WithMonth_OmitsCategoryWithNoBudget(t *testing.T) {
	m := month(2025, time.May)
	repo := &fakeRepo{
		categories: []domain.Category{{Name: "food"}, {Name: "transport"}},
		budgets:    []domain.Budget{{Category: "food", Month: month(2025, time.May), Amount: 300}},
	}
	svc := service.NewService(repo)

	budgets, err := svc.ListBudgets(context.Background(), domain.BudgetFilter{Month: &m})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(budgets) != 1 || budgets[0].Category != "food" {
		t.Errorf("expected only food budget, got %v", budgets)
	}
}

func TestListBudgets_WithoutMonth_RawRows(t *testing.T) {
	repo := &fakeRepo{
		categories: []domain.Category{{Name: "food"}},
		budgets: []domain.Budget{
			{Category: "food", Month: month(2025, time.January), Amount: 100},
			{Category: "food", Month: month(2025, time.March), Amount: 200},
		},
	}
	svc := service.NewService(repo)

	budgets, err := svc.ListBudgets(context.Background(), domain.BudgetFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(budgets) != 2 {
		t.Errorf("expected 2 raw budgets, got %d", len(budgets))
	}
}
