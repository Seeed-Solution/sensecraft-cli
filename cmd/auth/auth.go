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
	var opts task.SplitAuthOptions
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Sign in and save an Access Key",
		Long: `Sign in with OAuth Device Flow and save an Access Key.

With no extra flags in a terminal, this runs the setup wizard.
The China site supports SenseCraft Data only. Use Global for SenseCraft AI and CodeCraft Hub.

From a shell that cannot type into the same process, use the split flow.
Each command prints JSON and exits:

  sensecraft-cli auth init --site Global --no-wait
  sensecraft-cli auth init --device-code <device_code>
  sensecraft-cli auth init --device-code <device_code> --key-id <access_id>
  sensecraft-cli auth init --device-code <device_code> --label <name> --access-level read-write

Send verification_url to the user and end the turn before polling.
--json is accepted; the split flow always prints JSON.
Use --sandbox to try the wizard, or one split step, without network access.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			console := ui.DefaultConsole()

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			return task.RunAuthInit(ctx, console, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.Sandbox, "sandbox", false, "Simulate this step without sending real requests")
	cmd.Flags().StringVar(&opts.Site, "site", "", "Site for --no-wait: Global or China")
	cmd.Flags().BoolVar(&opts.NoWait, "no-wait", false, "Request a device code and return immediately")
	cmd.Flags().StringVar(&opts.DeviceCode, "device-code", "", "Continue a previous --no-wait login")
	cmd.Flags().StringVar(&opts.KeyID, "key-id", "", "Save this existing Access Key id after authorization")
	cmd.Flags().StringVar(&opts.Label, "label", "", "Create an Access Key with this label after authorization")
	cmd.Flags().StringVar(&opts.AccessLevel, "access-level", "", "Permission for --label: read-write (default) or only-read")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Accepted for agents; the split flow always prints JSON")
	return cmd
}
