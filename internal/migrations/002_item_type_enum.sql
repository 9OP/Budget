-- +goose Up
CREATE TYPE item_type AS ENUM ('INCOME', 'EXPENSE');
ALTER TABLE items ALTER COLUMN type TYPE item_type USING UPPER(type)::item_type;

-- +goose Down
ALTER TABLE items ALTER COLUMN type TYPE TEXT;
DROP TYPE item_type;
