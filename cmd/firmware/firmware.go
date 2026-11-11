package firmware

import (
	"encoding/json"
	"strings"

	"sensecraft-cloud-terminal/internal/service"

	"github.com/spf13/cobra"
)

// NewFirmwareCmd returns firmware helper commands for Agents (no OSS write).
func NewFirmwareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "firmware",
		Short: "Firmware build advice for SenseCraft flash-firmware boards",
	}
	cmd.AddCommand(newFirmwareAdviseCmd())
	return cmd
}

func newFirmwareAdviseCmd() *cobra.Command {
	var deviceID, name string
	cmd := &cobra.Command{
		Use:   "advise",
		Short: "Print per-device firmware artifact / Muse / upload guidance (JSON)",
		Long: `Look up flash-firmware.device_defaults for one board and print a build_recipe.

Agents must run this (or read app blocks → build_recipe) before compiling firmware
for CodeCraft Hub upload. Offline — no auth or network.

Examples:
  sensecraft-cli firmware advise --device-id 100056818
  sensecraft-cli firmware advise --name sticky`,
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := service.FirmwareAdvise(strings.TrimSpace(deviceID), strings.TrimSpace(name))
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(res)
		},
	}
	cmd.Flags().StringVar(&deviceID, "device-id", "", "Material SKU (preferred; example Sticky 100056818)")
	cmd.Flags().StringVar(&name, "name", "", "Board name substring when SKU is unknown (e.g. sticky, watcher, e1001)")
	return cmd
}
