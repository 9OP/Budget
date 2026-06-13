-- +goose Up
ALTER TYPE item_type ADD VALUE 'INVESTMENT';

-- +goose Down
-- Postgres does not support removing enum values natively.
-- To roll back: recreate the enum without INVESTMENT and cast the column.
