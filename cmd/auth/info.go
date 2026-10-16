package auth

import (
	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/ui"

	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show authentication info from the local config file",
		Long:  `Displays site, organization ID, Key ID, creation time, and the active profile from sensecraft.json.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := config.LoadConfig()
			if err != nil {
				return err
			}
			configPath, err := config.ResolveConfigPath()
			if err != nil {
				return err
			}
			console := ui.DefaultConsole()
			console.Println(config.FormatCliConfigInfo(cli, configPath))
			return nil
		},
	}
}
