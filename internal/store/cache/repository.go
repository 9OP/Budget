// Package cache provides a caching layer over service.Repository using an LRU
// with TTL-based expiration. The full dataset for each entity type is fetched
// once and stored under a single key; subsequent reads are served from memory
// with filters applied in-process. Any write invalidates the relevant entry
// so the next read re-fetches fresh data from the underlying repository.
package cache

import (
	"context"
	"time"

	"github.com/9op/budget/internal/domain"
	"github.com/9op/budget/internal/service"
	lru "github.com/hashicorp/golang-lru/v2/expirable"
)

const cacheKey = "all"

// Repository wraps a service.Repository with per-entity LRU caches.
type Repository struct {
	repo       service.Repository
	items      *lru.LRU[string, []domain.Item]
	categories *lru.LRU[string, []domain.Category]
	budgets    *lru.LRU[string, []domain.Budget]
}

// NewRepository returns a Repository that caches results from repo for ttl.
// A ttl of 0 disables expiration (entries live until evicted by capacity).
func NewRepository(repo service.Repository, ttl time.Duration) *Repository {
	return &Repository{
		repo:       repo,
		items:      lru.NewLRU[string, []domain.Item](1, nil, ttl),
		categories: lru.NewLRU[string, []domain.Category](1, nil, ttl),
		budgets:    lru.NewLRU[string, []domain.Budget](1, nil, ttl),
	}
}

// ListItems returns items matching filter, fetching all from the underlying
// repository on a cache miss and applying the filter in-memory.
func (r *Repository) ListItems(ctx context.Context, filter domain.ItemFilter) ([]domain.Item, error) {
	all, err := r.allItems(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Item, 0, len(all))
	for _, item := range all {
		if item.Matches(filter) {
			result = append(result, item)
		}
	}

	return result, nil
}

// CreateItem delegates to the underlying repository and invalidates the items cache.
func (r *Repository) CreateItem(ctx context.Context, item domain.Item) (domain.Item, error) {
	created, err := r.repo.CreateItem(ctx, item)
	if err != nil {
		return domain.Item{}, err
	}

	r.items.Remove(cacheKey)

	return created, nil
}

// GetItem returns the item with the given ID from the cache, fetching all items on a miss.
func (r *Repository) GetItem(ctx context.Context, id string) (domain.Item, error) {
	all, err := r.allItems(ctx)
	if err != nil {
		return domain.Item{}, err
	}

	for _, item := range all {
		if item.ID == id {
			return item, nil
		}
	}

	return domain.Item{}, domain.ErrItemNotFound
}

// UpdateItem delegates to the underlying repository and invalidates the items cache.
func (r *Repository) UpdateItem(ctx context.Context, item domain.Item) (domain.Item, error) {
	updated, err := r.repo.UpdateItem(ctx, item)
	if err != nil {
		return domain.Item{}, err
	}

	r.items.Remove(cacheKey)

	return updated, nil
}

// ListCategories returns all categories, fetching from the underlying repository on a miss.
func (r *Repository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return r.allCategories(ctx)
}

// CreateCategory delegates to the underlying repository and invalidates the categories cache.
func (r *Repository) CreateCategory(ctx context.Context, cat domain.Category) (domain.Category, error) {
	created, err := r.repo.CreateCategory(ctx, cat)
	if err != nil {
		return domain.Category{}, err
	}

	r.categories.Remove(cacheKey)

	return created, nil
}

// DeleteCategory delegates to the underlying repository and invalidates the categories cache.
func (r *Repository) DeleteCategory(ctx context.Context, name string) error {
	if err := r.repo.DeleteCategory(ctx, name); err != nil {
		return err
	}

	r.categories.Remove(cacheKey)

	return nil
}

// RenameCategory delegates to the underlying repository and invalidates the categories, items, and budgets caches.
func (r *Repository) RenameCategory(ctx context.Context, oldName, newName string) error {
	if err := r.repo.RenameCategory(ctx, oldName, newName); err != nil {
		return err
	}

	r.categories.Remove(cacheKey)
	r.items.Remove(cacheKey)
	r.budgets.Remove(cacheKey)

	return nil
}

// ListBudgets returns budgets matching filter, fetching all from the underlying
// repository on a cache miss and applying the filter in-memory.
func (r *Repository) ListBudgets(ctx context.Context, filter domain.BudgetFilter) ([]domain.Budget, error) {
	all, err := r.allBudgets(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Budget, 0, len(all))
	for _, b := range all {
		if b.Matches(filter) {
			result = append(result, b)
		}
	}

	return result, nil
}

// DeleteItem delegates to the underlying repository and invalidates the items cache.
func (r *Repository) DeleteItem(ctx context.Context, id string) error {
	if err := r.repo.DeleteItem(ctx, id); err != nil {
		return err
	}

	r.items.Remove(cacheKey)

	return nil
}

// SetBudget delegates to the underlying repository and invalidates the budgets cache.
func (r *Repository) SetBudget(ctx context.Context, budget domain.Budget) (domain.Budget, error) {
	set, err := r.repo.SetBudget(ctx, budget)
	if err != nil {
		return domain.Budget{}, err
	}

	r.budgets.Remove(cacheKey)

	return set, nil
}

// DeleteBudget delegates to the underlying repository and invalidates the budgets cache.
func (r *Repository) DeleteBudget(ctx context.Context, category string, month time.Time) error {
	if err := r.repo.DeleteBudget(ctx, category, month); err != nil {
		return err
	}

	r.budgets.Remove(cacheKey)

	return nil
}

// --- cache helpers ---

func (r *Repository) allItems(ctx context.Context) ([]domain.Item, error) {
	if cached, ok := r.items.Get(cacheKey); ok {
		return cached, nil
	}

	all, err := r.repo.ListItems(ctx, domain.ItemFilter{})
	if err != nil {
		return nil, err
	}

	r.items.Add(cacheKey, all)

	return all, nil
}

func (r *Repository) allCategories(ctx context.Context) ([]domain.Category, error) {
	if cached, ok := r.categories.Get(cacheKey); ok {
		return cached, nil
	}

	all, err := r.repo.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	r.categories.Add(cacheKey, all)

	return all, nil
}

func (r *Repository) allBudgets(ctx context.Context) ([]domain.Budget, error) {
	if cached, ok := r.budgets.Get(cacheKey); ok {
		return cached, nil
	}

	all, err := r.repo.ListBudgets(ctx, domain.BudgetFilter{})
	if err != nil {
		return nil, err
	}

	r.budgets.Add(cacheKey, all)

	return all, nil
}
