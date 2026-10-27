package auth

import (
	"context"
	"os"
	"os/signal"
	"sensecraft-cloud-terminal/internal/task"
	"sensecraft-cloud-terminal/internal/ui"

	"github.com/spf13/cobra"
)

func NewAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication and local config management",
	}
	authCmd.AddCommand(newInitCmd())
	authCmd.AddCommand(newAddCmd())
	authCmd.AddCommand(newInfoCmd())
	return authCmd
}

func newInitCmd() *cobra.Command {
	var sandbox bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive CLI setup (OAuth Device Flow)",
		Long: `Guides you through browser authorization via OAuth Device Flow and saves an Access Key.

The key is used for SenseCraft Data, SenseCraft AI, and CodeCraft Hub.
The China site supports SenseCraft Data only. Use Global for SenseCraft AI and CodeCraft Hub.

Use --sandbox to try the full interactive flow without network access.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			console := ui.DefaultConsole()

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			return task.RunInitFlow(ctx, console, sandbox)
		},
	}
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: simulate Device Flow without sending real requests")
	return cmd
}
