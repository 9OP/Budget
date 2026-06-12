# Budget

## Setup

Required env vars:

| Var            | Description                           |
| -------------- | ------------------------------------- |
| `DATABASE_URL` | Postgres connection string (Supabase) |
| `PORT`         | HTTP port (default: `8080`)           |

## Commands

```bash
make build       # Build binary
make test        # Run tests
make lint        # Run linters
make fmt         # Format code
make all         # fmt + lint + test + build
```

## Migrations

```bash
./budget migrate up       # Apply pending migrations
./budget migrate down     # Roll back last migration
./budget migrate status   # Show migration status
./budget migrate version  # Show current version
```

## Run

```bash
./budget serve
```
