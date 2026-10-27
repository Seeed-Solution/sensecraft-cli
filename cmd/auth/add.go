package auth

import (
	"context"
	"os"
	"os/signal"

	"sensecraft-cloud-terminal/internal/task"
	"sensecraft-cloud-terminal/internal/ui"

	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	var (
		keyID     string
		accessKey string
		site      string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Manually add an Access Key to local config",
		Long: `Add an existing Access Key by Key ID and Access Key, verify it via OpenAPI
(view_organization), save it to the local config, and set it as the active profile.

Site defaults to Global (International) when omitted.
The China site supports SenseCraft Data only. Use Global for SenseCraft AI and CodeCraft Hub.
In an interactive terminal, missing Key ID / Access Key will be prompted.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			console := ui.DefaultConsole()

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			return task.RunAddAccessFlow(ctx, console, task.AddAccessOptions{
				KeyID:     keyID,
				AccessKey: accessKey,
				Site:      site,
			})
		},
	}
	cmd.Flags().StringVar(&keyID, "key-id", "", "Access Key ID (API ID)")
	cmd.Flags().StringVar(&accessKey, "access-key", "", "Access Key secret")
	cmd.Flags().StringVar(&site, "site", "", "Site: Global or China (default: Global)")
	return cmd
}
