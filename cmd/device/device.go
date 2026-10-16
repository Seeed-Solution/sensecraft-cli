package device

import (
	"fmt"
	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/service"
	"time"

	"github.com/spf13/cobra"
)

func NewDeviceCmd() *cobra.Command {
	deviceCmd := &cobra.Command{
		Use:   "device",
		Short: "Device management",
	}
	deviceCmd.AddCommand(newDeviceListCmd())
	deviceCmd.AddCommand(newDeviceViewCmd())
	deviceCmd.AddCommand(newDeviceChannelsCmd())
	deviceCmd.AddCommand(newDeviceStatusCmd())
	deviceCmd.AddCommand(newDeviceBindCmd())
	deviceCmd.AddCommand(newDeviceUnbindCmd())
	deviceCmd.AddCommand(newDeviceSimCmd())
	deviceCmd.AddCommand(newDevKitCmd())
	return deviceCmd
}

func newDeviceListCmd() *cobra.Command {
	var deviceListType, deviceListGroupUUID string
	var sandbox bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List devices",
		Long: `device_type: 1=gateway, 2=node (default), 3=development kit.
If group_uuid is omitted, devices from all groups are returned.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			logFields := map[string]string{
				"cmd":    "device list",
				"time":   time.Now().String(),
				"params": fmt.Sprintf("device_type=%s, group_uuid=%s", deviceListType, deviceListGroupUUID),
			}

			deviceList, err := service.GetDeviceService(ctx).ListDevice(deviceListType, deviceListGroupUUID)
			if err != nil {
				ctx.LogError(err, logFields)
				return err
			}
			return ctx.PrintJsonData(deviceList)
		},
	}
	cmd.Flags().StringVar(&deviceListType, "device-type", "2", "Device type: 1=gateway, 2=node, 3=development kit")
	cmd.Flags().StringVar(&deviceListGroupUUID, "group-uuid", "", "Filter by group")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	return cmd
}

func newDeviceViewCmd() *cobra.Command {
	var deviceViewEUIs []string
	var deviceViewType string
	cmd := &cobra.Command{
		Use:   "view",
		Short: "Get device details (up to 50 EUIs)",
		Long:  `device_type: 1=gateway, 2=node (default), 3=development kit.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			data, err := service.GetDeviceService(ctx).ViewDevices(deviceViewEUIs, deviceViewType)
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringSliceVar(&deviceViewEUIs, "eui", nil, "Device EUI; may be specified multiple times (required)")
	cmd.Flags().StringVar(&deviceViewType, "device-type", "2", "Device type: 1=gateway, 2=node, 3=development kit")
	_ = cmd.MarkFlagRequired("eui")
	return cmd
}

func newDeviceChannelsCmd() *cobra.Command {
	var deviceChannelsEUIs []string
	cmd := &cobra.Command{
		Use:   "channels",
		Short: "List node channels (up to 50 EUIs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			data, err := service.GetDeviceService(ctx).ListDeviceChannels(deviceChannelsEUIs)
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringSliceVar(&deviceChannelsEUIs, "eui", nil, "Device EUI; may be specified multiple times (required)")
	_ = cmd.MarkFlagRequired("eui")
	return cmd
}

func newDeviceStatusCmd() *cobra.Command {
	var deviceStatusEUIs []string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get device running status (up to 50 EUIs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			data, err := service.GetDeviceService(ctx).ViewDeviceRunningStatus(deviceStatusEUIs)
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringSliceVar(&deviceStatusEUIs, "eui", nil, "Device EUI; may be specified multiple times (required)")
	_ = cmd.MarkFlagRequired("eui")
	return cmd
}

func newDeviceBindCmd() *cobra.Command {
	var (
		deviceBindEUI       string
		deviceBindCode      string
		deviceBindName      string
		deviceBindGroupUUID string
		deviceBindLongitude string
		deviceBindLatitude  string
	)
	cmd := &cobra.Command{
		Use:   "bind",
		Short: "Bind a device to the current account",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			body := map[string]string{
				"eui":  deviceBindEUI,
				"code": deviceBindCode,
			}
			if deviceBindName != "" {
				body["device_name"] = deviceBindName
			}
			if deviceBindGroupUUID != "" {
				body["group_uuid"] = deviceBindGroupUUID
			}
			if deviceBindLongitude != "" {
				body["longitude"] = deviceBindLongitude
			}
			if deviceBindLatitude != "" {
				body["latitude"] = deviceBindLatitude
			}
			data, err := service.GetDeviceService(ctx).BindDevice(body)
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&deviceBindEUI, "eui", "", "Device EUI (required)")
	cmd.Flags().StringVar(&deviceBindCode, "code", "", "Device code (required)")
	cmd.Flags().StringVar(&deviceBindName, "name", "", "Device name")
	cmd.Flags().StringVar(&deviceBindGroupUUID, "group-uuid", "", "Target group UUID")
	cmd.Flags().StringVar(&deviceBindLongitude, "longitude", "", "Longitude")
	cmd.Flags().StringVar(&deviceBindLatitude, "latitude", "", "Latitude")
	_ = cmd.MarkFlagRequired("eui")
	_ = cmd.MarkFlagRequired("code")
	return cmd
}

func newDeviceUnbindCmd() *cobra.Command {
	var deviceUnbindEUIs []string
	var yes bool
	cmd := &cobra.Command{
		Use:   "unbind",
		Short: "Unbind devices (up to 50 EUIs, L2: --yes)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			data, err := service.GetDeviceService(ctx).UnbindDevices(deviceUnbindEUIs, yes)
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringSliceVar(&deviceUnbindEUIs, "eui", nil, "Device EUI; may be specified multiple times (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 unbind")
	_ = cmd.MarkFlagRequired("eui")
	return cmd
}

func newDeviceSimCmd() *cobra.Command {
	var deviceSimICCID string
	cmd := &cobra.Command{
		Use:   "sim",
		Short: "View SIM card status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			data, err := service.GetDeviceService(ctx).ViewSimcard(deviceSimICCID)
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&deviceSimICCID, "iccid", "", "SIM card ICCID (required)")
	_ = cmd.MarkFlagRequired("iccid")
	return cmd
}
