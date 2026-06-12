package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/9op/budget/internal/config"
	"github.com/9op/budget/internal/service"
	"github.com/9op/budget/internal/store/cache"
	"github.com/9op/budget/internal/store/postgres"
	"github.com/9op/budget/internal/web"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runServer(cmd.Context()) },
	}
}

const shutdownTimeout = 10 * time.Second

// Run starts the HTTP server and blocks until ctx is cancelled or a fatal error occurs.
func runServer(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	pgRepo := postgres.NewRepository(pool)
	repo := cache.NewRepository(pgRepo, cfg.CacheTTL)
	svc := service.NewService(repo)

	handler, err := web.NewServer(svc)
	if err != nil {
		return fmt.Errorf("build server: %w", err)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: handler} //nolint:gosec // port is from trusted config

	slog.Info("starting server", slog.String("addr", srv.Addr))

	context.AfterFunc(ctx, func() {
		slog.Info("shutting down server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown server", slog.String("error", err.Error()))
		}
	})

	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}
