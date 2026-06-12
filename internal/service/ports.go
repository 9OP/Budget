package service

import (
	"context"
	"time"

	"github.com/9op/budget/internal/domain"
)

// Repository is the port for data persistence.
type Repository interface {
	CreateItem(ctx context.Context, item domain.Item) (domain.Item, error)
	GetItem(ctx context.Context, id string) (domain.Item, error)
	UpdateItem(ctx context.Context, item domain.Item) (domain.Item, error)
	ListItems(ctx context.Context, filter domain.ItemFilter) ([]domain.Item, error)
	DeleteItem(ctx context.Context, id string) error
	CreateCategory(ctx context.Context, cat domain.Category) (domain.Category, error)
	ListCategories(ctx context.Context) ([]domain.Category, error)
	RenameCategory(ctx context.Context, oldName, newName string) error
	DeleteCategory(ctx context.Context, name string) error
	SetBudget(ctx context.Context, budget domain.Budget) (domain.Budget, error)
	ListBudgets(ctx context.Context, filter domain.BudgetFilter) ([]domain.Budget, error)
	DeleteBudget(ctx context.Context, category string, month time.Time) error
}
