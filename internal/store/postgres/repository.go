// Package postgres provides a PostgreSQL-backed implementation of service.Repository.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/9op/budget/internal/auth"
	"github.com/9op/budget/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	pgUniqueViolation = "23505"
	pgFKViolation     = "23503"
)

// Repository implements service.Repository backed by PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository using the given connection pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// mustUserID extracts the user UUID from ctx or returns ErrUnauthenticated.
func mustUserID(ctx context.Context) (string, error) {
	id, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return "", domain.ErrUnauthenticated
	}

	return id, nil
}

// --- Items ---

// CreateItem inserts a new item and returns it with the generated UUID as ID.
func (r *Repository) CreateItem(ctx context.Context, item domain.Item) (domain.Item, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return domain.Item{}, err
	}

	catID, err := r.categoryIDByName(ctx, userID, item.Category)
	if err != nil {
		return domain.Item{}, err
	}

	const q = `INSERT INTO items (user_id, type, name, amount, date, category_id)
		VALUES ($1::uuid, $2, $3, $4, $5, $6::uuid) RETURNING id::text`

	if err := r.pool.QueryRow(ctx, q,
		userID, string(item.Type), item.Name, item.Amount, item.Date, catID,
	).Scan(&item.ID); err != nil {
		return domain.Item{}, fmt.Errorf("insert item: %w", err)
	}

	return item, nil
}

// GetItem fetches a single item by UUID, scoped to the authenticated user.
func (r *Repository) GetItem(ctx context.Context, id string) (domain.Item, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return domain.Item{}, err
	}

	const q = `SELECT i.id::text, i.type, i.name, i.amount, i.date, c.name
		FROM items i JOIN categories c ON c.id = i.category_id
		WHERE i.id = $1::uuid AND i.user_id = $2::uuid`

	item, err := scanItem(r.pool.QueryRow(ctx, q, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Item{}, domain.ErrItemNotFound
	}

	if err != nil {
		return domain.Item{}, fmt.Errorf("get item: %w", err)
	}

	return item, nil
}

// UpdateItem overwrites the item identified by item.ID, scoped to the authenticated user.
func (r *Repository) UpdateItem(ctx context.Context, item domain.Item) (domain.Item, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return domain.Item{}, err
	}

	catID, err := r.categoryIDByName(ctx, userID, item.Category)
	if err != nil {
		return domain.Item{}, err
	}

	const q = `UPDATE items SET type=$1, name=$2, amount=$3, date=$4, category_id=$5::uuid
		WHERE id=$6::uuid AND user_id=$7::uuid`

	tag, err := r.pool.Exec(ctx, q,
		string(item.Type), item.Name, item.Amount, item.Date, catID, item.ID, userID,
	)
	if err != nil {
		return domain.Item{}, fmt.Errorf("update item: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.Item{}, domain.ErrItemNotFound
	}

	return item, nil
}

// DeleteItem removes an item by UUID, scoped to the authenticated user.
func (r *Repository) DeleteItem(ctx context.Context, id string) error {
	userID, err := mustUserID(ctx)
	if err != nil {
		return err
	}

	const q = `DELETE FROM items WHERE id = $1::uuid AND user_id = $2::uuid`

	tag, err := r.pool.Exec(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrItemNotFound
	}

	return nil
}

// ListItems returns items matching the given filter, scoped to the authenticated user.
func (r *Repository) ListItems(ctx context.Context, filter domain.ItemFilter) ([]domain.Item, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return nil, err
	}

	args := []any{userID}
	query := `SELECT i.id::text, i.type, i.name, i.amount, i.date, c.name
		FROM items i JOIN categories c ON c.id = i.category_id
		WHERE i.user_id = $1::uuid`

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

// CreateCategory inserts a new category for the authenticated user, returning ErrCategoryAlreadyExists on duplicate.
func (r *Repository) CreateCategory(ctx context.Context, cat domain.Category) (domain.Category, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return domain.Category{}, err
	}

	const q = `INSERT INTO categories (user_id, name) VALUES ($1::uuid, $2)`

	_, err = r.pool.Exec(ctx, q, userID, cat.Name)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return domain.Category{}, domain.ErrCategoryAlreadyExists
		}

		return domain.Category{}, fmt.Errorf("insert category: %w", err)
	}

	return cat, nil
}

// ListCategories returns all categories for the authenticated user, sorted by name.
func (r *Repository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return nil, err
	}

	const q = `SELECT name FROM categories WHERE user_id = $1::uuid ORDER BY name`

	rows, err := r.pool.Query(ctx, q, userID)
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

// RenameCategory updates the category name for the authenticated user.
func (r *Repository) RenameCategory(ctx context.Context, oldName, newName string) error {
	userID, err := mustUserID(ctx)
	if err != nil {
		return err
	}

	tag, err := r.pool.Exec(ctx,
		`UPDATE categories SET name=$1 WHERE name=$2 AND user_id=$3::uuid`,
		newName, oldName, userID,
	)
	if err != nil {
		return fmt.Errorf("rename category: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrCategoryNotFound
	}

	return nil
}

// DeleteCategory removes a category by name for the authenticated user.
func (r *Repository) DeleteCategory(ctx context.Context, name string) error {
	userID, err := mustUserID(ctx)
	if err != nil {
		return err
	}

	tag, err := r.pool.Exec(ctx,
		`DELETE FROM categories WHERE name ILIKE $1 AND user_id = $2::uuid`,
		name, userID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgFKViolation {
			return domain.ErrCategoryInUse
		}

		return fmt.Errorf("delete category: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrCategoryNotFound
	}

	return nil
}

// --- Budgets ---

// SetBudget upserts a budget row for (user, category, month).
func (r *Repository) SetBudget(ctx context.Context, budget domain.Budget) (domain.Budget, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return domain.Budget{}, err
	}

	catID, err := r.categoryIDByName(ctx, userID, budget.Category)
	if err != nil {
		return domain.Budget{}, err
	}

	const q = `INSERT INTO budgets (user_id, category_id, month, amount) VALUES ($1::uuid, $2::uuid, $3, $4)
		ON CONFLICT (user_id, category_id, month) DO UPDATE SET amount = EXCLUDED.amount`

	month := firstOfMonth(budget.Month)

	if _, err := r.pool.Exec(ctx, q, userID, catID, month, budget.Amount); err != nil {
		return domain.Budget{}, fmt.Errorf("upsert budget: %w", err)
	}

	budget.Month = month

	return budget, nil
}

// ListBudgets returns budgets matching the given filter, scoped to the authenticated user.
func (r *Repository) ListBudgets(ctx context.Context, filter domain.BudgetFilter) ([]domain.Budget, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return nil, err
	}

	args := []any{userID}
	query := `SELECT c.name, b.month, b.amount
		FROM budgets b JOIN categories c ON c.id = b.category_id
		WHERE b.user_id = $1::uuid`

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

// DeleteBudget removes the budget for (user, category, month).
func (r *Repository) DeleteBudget(ctx context.Context, category string, month time.Time) error {
	userID, err := mustUserID(ctx)
	if err != nil {
		return err
	}

	const q = `DELETE FROM budgets
		WHERE user_id = $1::uuid
		AND category_id = (SELECT id FROM categories WHERE name ILIKE $2 AND user_id = $1::uuid)
		AND month = $3`

	tag, err := r.pool.Exec(ctx, q, userID, category, firstOfMonth(month))
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

// categoryIDByName looks up the UUID of a category by name for a given user (case-insensitive).
func (r *Repository) categoryIDByName(ctx context.Context, userID, name string) (string, error) {
	var id string

	err := r.pool.QueryRow(ctx,
		`SELECT id::text FROM categories WHERE name ILIKE $1 AND user_id = $2::uuid`,
		name, userID,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrCategoryNotFound
	}

	if err != nil {
		return "", fmt.Errorf("lookup category %q: %w", name, err)
	}

	return id, nil
}
