-- +goose Up
CREATE TABLE categories (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE items (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    type        TEXT    NOT NULL,
    name        TEXT    NOT NULL,
    amount      NUMERIC NOT NULL,
    date        DATE    NOT NULL,
    category_id UUID    NOT NULL REFERENCES categories(id)
);

CREATE TABLE budgets (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID    NOT NULL REFERENCES categories(id),
    month       DATE    NOT NULL,
    amount      NUMERIC NOT NULL,
    UNIQUE (category_id, month)
);

-- +goose Down
DROP TABLE IF EXISTS budgets;
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS categories;
