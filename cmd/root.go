package main

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/spf13/cobra"
)

type RunE = func(cmd *cobra.Command, args []string) error

func withRecovery(runE RunE) RunE {
	return func(cmd *cobra.Command, args []string) error {
		defer func() {
			if rcv := recover(); rcv != nil {
				stack := debug.Stack()
				slog.Error("panic", slog.Any("recover", rcv))
				fmt.Println(string(stack))
			}
		}()
		return runE(cmd, args)
	}
}

func wrapsWithRecover(cmd *cobra.Command) {
	if cmd.RunE != nil {
		originalRunE := cmd.RunE
		cmd.RunE = withRecovery(originalRunE)
	}
	for _, sub := range cmd.Commands() {
		wrapsWithRecover(sub)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "budget",
		Short:         "Budget management tool",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newServeCmd())
	root.AddCommand(newMigrateCmd())

	root.SetHelpCommand(&cobra.Command{Hidden: true})
	root.CompletionOptions.DisableDefaultCmd = true
	root.SilenceErrors = true
	root.SilenceUsage = true

	wrapsWithRecover(root) //nolint:contextcheck

	return root
}
