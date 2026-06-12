// Package main is the entry point for the budget CLI.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lmittmann/tint"
)

func main() {
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: "15:04:05.000",
	})))

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	err := newRootCmd().ExecuteContext(ctx)
	cancel()

	if err != nil {
		slog.Error("command error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
