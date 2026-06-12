package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/9op/budget/internal/config"
	"github.com/9op/budget/internal/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Manage database migrations",
	}

	cmd.AddCommand(
		newMigrateSubCmd("up", "Apply all pending migrations", goose.UpContext),
		newMigrateSubCmd("down", "Roll back the last applied migration", goose.DownContext),
		newMigrateSubCmd("status", "Show migration status", goose.StatusContext),
		newMigrateSubCmd("version", "Show current migration version", goose.VersionContext),
	)

	return cmd
}

type gooseFunc func(ctx context.Context, db *sql.DB, dir string, opts ...goose.OptionsFunc) error

func newMigrateSubCmd(use, short string, fn gooseFunc) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withDB(func(db *sql.DB) error {
				return fn(cmd.Context(), db, ".")
			})
		},
	}
}

func withDB(fn func(*sql.DB) error) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close() //nolint:errcheck // best-effort close on exit

	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	return fn(db)
}
