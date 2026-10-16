package device

import (
	"fmt"
	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/enum"
	"sensecraft-cloud-terminal/internal/service"
	"strings"

	"github.com/spf13/cobra"
)

func newDevKitCmd() *cobra.Command {
	devKitCmd := &cobra.Command{
		Use:   "devkit",
		Short: "Development kit management",
	}
	devKitCmd.AddCommand(newDevKitCreateCmd())
	devKitCmd.AddCommand(newDevKitKeyCmd())
	return devKitCmd
}

func newDevKitCreateCmd() *cobra.Command {
	var sku, devName string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a development kit device",
		Long: fmt.Sprintf(`Up to 5 development kits per SKU within the same organization.

Allowed SKUs:
  %s  Blank Device
  %s  xiao
  %s  Grove Vision AI V2`, enum.DevKitSKUBlankDevice, enum.DevKitSKUXiao, enum.DevKitSKUGroveVisionAI),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			resp, err := service.GetDevKitService(ctx).CreateDevelopmentKit(sku, devName, nil)
			if err != nil {
				return err
			}
			return ctx.PrintJsonData(resp)
		},
	}
	cmd.Flags().StringVar(&sku, "sku", "", "Development kit SKU (required)")
	cmd.Flags().StringVar(&devName, "name", "", "Device name")
	_ = cmd.MarkFlagRequired("sku")
	return cmd
}

func newDevKitKeyCmd() *cobra.Command {
	var eui string
	cmd := &cobra.Command{
		Use:   "key",
		Short: "View development kit device_key and token",
		Long: `Look up access credentials for a development kit device by its 16-character EUI.

Only supported for development kit devices (Blank Device / xiao / Grove Vision AI V2).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			eui = strings.ToUpper(strings.TrimSpace(eui))
			resp, err := service.GetDevKitService(ctx).ViewNodeKey(eui)
			if err != nil {
				return err
			}
			return ctx.PrintJsonData(resp)
		},
	}
	cmd.Flags().StringVar(&eui, "eui", "", "Development kit device EUI, 16 characters (required)")
	_ = cmd.MarkFlagRequired("eui")
	return cmd
}
