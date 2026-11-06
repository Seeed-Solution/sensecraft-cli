package auth

import (
	"encoding/json"

	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/ui"

	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show authentication info from the local config file",
		Long:  `Displays site, organization ID, Key ID, creation time, and the active profile from sensecraft.json. Use --json for profile metadata only.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := config.LoadConfig()
			if err != nil {
				return err
			}
			if asJSON {
				profiles := make([]authProfileInfo, 0)
				if cli != nil {
					for _, profile := range cli.AppConfigList {
						site := profile.RemoteType
						if site == "" {
							site = "Global"
						}
						profiles = append(profiles, authProfileInfo{
							Site: site, OrgID: profile.OrgId, KeyID: profile.KeyId,
							Kind: config.ProfileKindSenseCAP, CreatedAt: profile.CreateTime,
							Active: cli.UseApi != "" && cli.UseApi == profile.KeyId,
						})
					}
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{"profiles": profiles})
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
	cmd.Flags().BoolVar(&asJSON, "json", false, "Print profile metadata as JSON")
	return cmd
}

type authProfileInfo struct {
	Site      string `json:"site"`
	OrgID     string `json:"org_id"`
	KeyID     string `json:"key_id"`
	Kind      string `json:"kind"`
	CreatedAt string `json:"created_at"`
	Active    bool   `json:"active"`
}
