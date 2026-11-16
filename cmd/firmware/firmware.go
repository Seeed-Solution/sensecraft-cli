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
		Short: "Firmware upload advice for SenseCraft flash-firmware boards",
	}
	cmd.AddCommand(newFirmwareAdviseCmd())
	return cmd
}

func newFirmwareAdviseCmd() *cobra.Command {
	var deviceID, name string
	var list bool
	cmd := &cobra.Command{
		Use:   "advise",
		Short: "Print which firmware to upload for a board's default flash_address (JSON)",
		Long: `Look up flash-firmware.device_defaults and print upload guidance.

The CLI does not compile firmware. It tells Agents, for each supported device's
default flash_address, what shape of binary to upload (merged vs app-region vs
muse-gadget.bin), what not to upload, and which config flags to set.

Offline — no auth or network.

Examples:
  sensecraft-cli firmware advise --list
  sensecraft-cli firmware advise --device-id 100056818
  sensecraft-cli firmware advise --name sticky`,
		RunE: func(cmd *cobra.Command, args []string) error {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			if list {
				return enc.Encode(map[string]any{
					"rule": "Match device default flash_address → upload that firmware shape. CLI does not generate binaries.",
					"address_to_upload": map[string]string{
						"0x0":     "merged firmware (bootloader+partitions+app), except Grove .bin/.hex and ordinary XIAO dual-path",
						"0x90000": "application-region .bin only (E100x / e-paper)",
						"0x4000":  "application-region .bin only (Wio)",
						"0x10000": "ordinary XIAO application package only",
						"muse":    "muse-gadget.bin only; keep device default flash_address in config (never write 0x20000)",
					},
					"devices": service.FirmwareAdviseList(),
				})
			}
			res, err := service.FirmwareAdvise(strings.TrimSpace(deviceID), strings.TrimSpace(name))
			if err != nil {
				return err
			}
			return enc.Encode(res)
		},
	}
	cmd.Flags().StringVar(&deviceID, "device-id", "", "Material SKU (preferred; example Sticky 100056818)")
	cmd.Flags().StringVar(&name, "name", "", "Board name substring when SKU is unknown (e.g. sticky, watcher, e1001, epaper)")
	cmd.Flags().BoolVar(&list, "list", false, "List every supported board: default address → what to upload")
	return cmd
}
