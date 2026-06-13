-- +goose Up

-- Add user_id to all tables. Existing rows get a temporary random UUID
-- (they won't be accessible to any real user and can be cleaned up manually).
ALTER TABLE categories ADD COLUMN user_id UUID NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE items      ADD COLUMN user_id UUID NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE budgets    ADD COLUMN user_id UUID NOT NULL DEFAULT gen_random_uuid();

-- Remove temporary defaults.
ALTER TABLE categories ALTER COLUMN user_id DROP DEFAULT;
ALTER TABLE items      ALTER COLUMN user_id DROP DEFAULT;
ALTER TABLE budgets    ALTER COLUMN user_id DROP DEFAULT;

-- Categories: uniqueness is now per-user.
ALTER TABLE categories DROP CONSTRAINT categories_name_key;
ALTER TABLE categories ADD CONSTRAINT categories_user_name_key UNIQUE (user_id, name);

-- Budgets: uniqueness is now per-user.
ALTER TABLE budgets DROP CONSTRAINT budgets_category_id_month_key;
ALTER TABLE budgets ADD CONSTRAINT budgets_user_category_month_key UNIQUE (user_id, category_id, month);

-- Indexes for user-scoped filtering.
CREATE INDEX ON items      (user_id);
CREATE INDEX ON categories (user_id);
CREATE INDEX ON budgets    (user_id);

-- +goose Down
DROP INDEX IF EXISTS budgets_user_id_idx;
DROP INDEX IF EXISTS categories_user_id_idx;
DROP INDEX IF EXISTS items_user_id_idx;

ALTER TABLE budgets    DROP CONSTRAINT budgets_user_category_month_key;
ALTER TABLE budgets    ADD CONSTRAINT budgets_category_id_month_key UNIQUE (category_id, month);

ALTER TABLE categories DROP CONSTRAINT categories_user_name_key;
ALTER TABLE categories ADD CONSTRAINT categories_name_key UNIQUE (name);

ALTER TABLE budgets    DROP COLUMN user_id;
ALTER TABLE items      DROP COLUMN user_id;
ALTER TABLE categories DROP COLUMN user_id;
