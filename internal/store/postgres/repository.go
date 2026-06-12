// Package postgres provides a PostgreSQL-backed implementation of service.Repository.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/9op/budget/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const pgUniqueViolation = "23505"

// Repository implements service.Repository backed by PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository using the given connection pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// --- Items ---

// CreateItem inserts a new item and returns it with the generated UUID as ID.
func (r *Repository) CreateItem(ctx context.Context, item domain.Item) (domain.Item, error) {
	catID, err := r.categoryIDByName(ctx, item.Category)
	if err != nil {
		return domain.Item{}, err
	}

	const q = `INSERT INTO items (type, name, amount, date, category_id)
		VALUES ($1, $2, $3, $4, $5::uuid) RETURNING id::text`

	if err := r.pool.QueryRow(ctx, q,
		string(item.Type), item.Name, item.Amount, item.Date, catID,
	).Scan(&item.ID); err != nil {
		return domain.Item{}, fmt.Errorf("insert item: %w", err)
	}

	return item, nil
}

// GetItem fetches a single item by UUID.
func (r *Repository) GetItem(ctx context.Context, id string) (domain.Item, error) {
	const q = `SELECT i.id::text, i.type, i.name, i.amount, i.date, c.name
		FROM items i JOIN categories c ON c.id = i.category_id
		WHERE i.id = $1::uuid`

	item, err := scanItem(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Item{}, domain.ErrItemNotFound
	}

	if err != nil {
		return domain.Item{}, fmt.Errorf("get item: %w", err)
	}

	return item, nil
}

// UpdateItem overwrites the item identified by item.ID.
func (r *Repository) UpdateItem(ctx context.Context, item domain.Item) (domain.Item, error) {
	catID, err := r.categoryIDByName(ctx, item.Category)
	if err != nil {
		return domain.Item{}, err
	}

	const q = `UPDATE items SET type=$1, name=$2, amount=$3, date=$4, category_id=$5::uuid
		WHERE id=$6::uuid`

	tag, err := r.pool.Exec(ctx, q,
		string(item.Type), item.Name, item.Amount, item.Date, catID, item.ID,
	)
	if err != nil {
		return domain.Item{}, fmt.Errorf("update item: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.Item{}, domain.ErrItemNotFound
	}

	return item, nil
}

// DeleteItem removes an item by UUID.
func (r *Repository) DeleteItem(ctx context.Context, id string) error {
	const q = `DELETE FROM items WHERE id = $1::uuid`

	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrItemNotFound
	}

	return nil
}

// ListItems returns items matching the given filter.
func (r *Repository) ListItems(ctx context.Context, filter domain.ItemFilter) ([]domain.Item, error) {
	query := `SELECT i.id::text, i.type, i.name, i.amount, i.date, c.name
		FROM items i JOIN categories c ON c.id = i.category_id WHERE TRUE`
	args := []any{}

	if filter.Month != nil {
		args = append(args, filter.Month.Year(), int(filter.Month.Month()))
		query += fmt.Sprintf(
			" AND EXTRACT(YEAR FROM i.date)=$%d AND EXTRACT(MONTH FROM i.date)=$%d",
			len(args)-1,
			len(args),
		)
	} else {
		if filter.From != nil {
			args = append(args, *filter.From)
			query += fmt.Sprintf(" AND i.date >= $%d", len(args))
		}

		if filter.To != nil {
			args = append(args, *filter.To)
			query += fmt.Sprintf(" AND i.date <= $%d", len(args))
		}
	}

	if filter.Type != nil {
		args = append(args, string(*filter.Type))
		query += fmt.Sprintf(" AND i.type = $%d", len(args))
	}

	if filter.Category != nil {
		args = append(args, *filter.Category)
		query += fmt.Sprintf(" AND c.name ILIKE $%d", len(args))
	}

	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		query += fmt.Sprintf(" AND i.name ILIKE $%d", len(args))
	}

	query += " ORDER BY i.date DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	var items []domain.Item

	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate item rows: %w", err)
	}

	return items, nil
}

// --- Categories ---

// CreateCategory inserts a new category, returning ErrCategoryAlreadyExists on duplicate.
func (r *Repository) CreateCategory(ctx context.Context, cat domain.Category) (domain.Category, error) {
	const q = `INSERT INTO categories (name) VALUES ($1)`

	_, err := r.pool.Exec(ctx, q, cat.Name)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return domain.Category{}, domain.ErrCategoryAlreadyExists
		}

		return domain.Category{}, fmt.Errorf("insert category: %w", err)
	}

	return cat, nil
}

// ListCategories returns all categories sorted by name.
func (r *Repository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	const q = `SELECT name FROM categories ORDER BY name`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var cats []domain.Category

	for rows.Next() {
		var cat domain.Category
		if err := rows.Scan(&cat.Name); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}

		cats = append(cats, cat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate category rows: %w", err)
	}

	return cats, nil
}

// RenameCategory updates the category name. Because items and budgets reference
// categories by UUID, no cascade is needed — the JOIN picks up the new name automatically.
func (r *Repository) RenameCategory(ctx context.Context, oldName, newName string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE categories SET name=$1 WHERE name=$2`, newName, oldName)
	if err != nil {
		return fmt.Errorf("rename category: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrCategoryNotFound
	}

	return nil
}

// DeleteCategory removes a category by name. Fails if items or budgets still reference it.
func (r *Repository) DeleteCategory(ctx context.Context, name string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM categories WHERE name ILIKE $1`, name)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrCategoryNotFound
	}

	return nil
}

// --- Budgets ---

// SetBudget upserts a budget row for (category, month).
func (r *Repository) SetBudget(ctx context.Context, budget domain.Budget) (domain.Budget, error) {
	catID, err := r.categoryIDByName(ctx, budget.Category)
	if err != nil {
		return domain.Budget{}, err
	}

	const q = `INSERT INTO budgets (category_id, month, amount) VALUES ($1::uuid, $2, $3)
		ON CONFLICT (category_id, month) DO UPDATE SET amount = EXCLUDED.amount`

	month := firstOfMonth(budget.Month)

	if _, err := r.pool.Exec(ctx, q, catID, month, budget.Amount); err != nil {
		return domain.Budget{}, fmt.Errorf("upsert budget: %w", err)
	}

	budget.Month = month

	return budget, nil
}

// ListBudgets returns budgets matching the given filter.
func (r *Repository) ListBudgets(ctx context.Context, filter domain.BudgetFilter) ([]domain.Budget, error) {
	query := `SELECT c.name, b.month, b.amount
		FROM budgets b JOIN categories c ON c.id = b.category_id WHERE TRUE`
	args := []any{}

	if filter.Month != nil {
		args = append(args, firstOfMonth(*filter.Month))
		query += fmt.Sprintf(" AND b.month = $%d", len(args))
	}

	if filter.Category != nil {
		args = append(args, *filter.Category)
		query += fmt.Sprintf(" AND c.name ILIKE $%d", len(args))
	}

	query += " ORDER BY b.month DESC, c.name"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	defer rows.Close()

	var budgets []domain.Budget

	for rows.Next() {
		b, err := scanBudget(rows)
		if err != nil {
			return nil, fmt.Errorf("scan budget: %w", err)
		}

		budgets = append(budgets, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate budget rows: %w", err)
	}

	return budgets, nil
}

// DeleteBudget removes the budget for (category, month).
func (r *Repository) DeleteBudget(ctx context.Context, category string, month time.Time) error {
	const q = `DELETE FROM budgets WHERE category_id = (SELECT id FROM categories WHERE name ILIKE $1)
		AND month = $2`

	tag, err := r.pool.Exec(ctx, q, category, firstOfMonth(month))
	if err != nil {
		return fmt.Errorf("delete budget: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrBudgetNotFound
	}

	return nil
}

// --- helpers ---

type scanner interface {
	Scan(dest ...any) error
}

func scanItem(row scanner) (domain.Item, error) {
	var item domain.Item
	var t string

	if err := row.Scan(&item.ID, &t, &item.Name, &item.Amount, &item.Date, &item.Category); err != nil {
		return domain.Item{}, err
	}

	item.Type = domain.ItemType(t)
	item.Date = item.Date.UTC()

	return item, nil
}

func scanBudget(row scanner) (domain.Budget, error) {
	var b domain.Budget

	if err := row.Scan(&b.Category, &b.Month, &b.Amount); err != nil {
		return domain.Budget{}, err
	}

	b.Month = firstOfMonth(b.Month)

	return b, nil
}

func firstOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// categoryIDByName looks up the UUID of a category by name (case-insensitive).
func (r *Repository) categoryIDByName(ctx context.Context, name string) (string, error) {
	var id string

	err := r.pool.QueryRow(ctx, `SELECT id::text FROM categories WHERE name ILIKE $1`, name).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrCategoryNotFound
	}

	if err != nil {
		return "", fmt.Errorf("lookup category %q: %w", name, err)
	}

	return id, nil
}
