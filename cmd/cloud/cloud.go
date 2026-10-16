package cloud

import (
	cmdctx "sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/service"

	"github.com/spf13/cobra"
)

func NewCloudCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud",
		Short: "SenseCraft AI cloud devices / Jetson deploy (Access Key /openapi)",
		Long: `Cloud kit devices via {ai-host}/openapi /device/* (Access Key Basic; not SenseCAP OpenAPI device).

Distinct from:
  sensecraft-cli device …     → SenseCAP nodes/gateways (Access Key)
  sensecraft-cli recamera …   → classic reCamera 40 check / open Console deploy page`,
	}
	cmd.AddCommand(newCloudDeviceCmd())
	cmd.AddCommand(newCloudDeployCmd())
	cmd.AddCommand(newCloudOrderCmd())
	return cmd
}

func newCloudDeviceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device",
		Short: "List / view AI cloud devices (list_node / view_device)",
	}
	cmd.AddCommand(newCloudDeviceListCmd())
	cmd.AddCommand(newCloudDeviceViewCmd())
	return cmd
}

func newCloudDeviceListCmd() *cobra.Command {
	var (
		uniformType int
		name        string
		page        int
		length      int
		adapteds    string
		sandbox     bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List cloud devices (GET /device/list_node)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetCloudService(ctx).ListDevices(service.CloudDeviceListParams{
				UniformType: uniformType,
				Name:        name,
				Page:        page,
				Length:      length,
				Adapteds:    adapteds,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().IntVar(&uniformType, "uniform-type", 26, "Device uniform_type (Jetson cloud UI often uses 26)")
	cmd.Flags().StringVar(&name, "name", "", "Filter by device_name")
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&length, "length", 20, "Page size")
	cmd.Flags().StringVar(&adapteds, "adapteds", "", "Optional adapteds filter")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode")
	return cmd
}

func newCloudDeviceViewCmd() *cobra.Command {
	var (
		eui     string
		sandbox bool
	)
	cmd := &cobra.Command{
		Use:   "view",
		Short: "View cloud device (GET /device/view_device)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetCloudService(ctx).ViewDevice(service.CloudDeviceViewParams{EUI: eui})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&eui, "eui", "", "Device EUI (required)")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode")
	_ = cmd.MarkFlagRequired("eui")
	return cmd
}

func newCloudDeployCmd() *cobra.Command {
	var (
		eui      string
		modelID  string
		fromJSON string
		yes      bool
		sandbox  bool
	)
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy model to cloud device via update_stream (L2: --yes)",
		Long: `POST /device/update_stream

Requires --eui, --model-id, and usually --from-json with stream setting.
Poll result with: sensecraft-cli cloud order --request-id <id> [--device-uuid <uuid>]

This is for AI cloud kits / Jetson (list_node via /openapi). It is NOT reCamera Pro
(uniform_type=53) local IP/USB RC2 or ALPKG deploy — guide users to Console for Pro.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetCloudService(ctx).DeployModel(service.CloudDeployParams{
				EUI:      eui,
				ModelID:  modelID,
				FromJSON: fromJSON,
				Yes:      yes,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&eui, "eui", "", "Device EUI (required)")
	cmd.Flags().StringVar(&modelID, "model-id", "", "Model id (required)")
	cmd.Flags().StringVar(&fromJSON, "from-json", "", "Stream setting JSON object")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 deploy")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode")
	_ = cmd.MarkFlagRequired("eui")
	_ = cmd.MarkFlagRequired("model-id")
	return cmd
}

func newCloudOrderCmd() *cobra.Command {
	var (
		requestID  string
		deviceUUID string
		sandbox    bool
	)
	cmd := &cobra.Command{
		Use:   "order",
		Short: "Fetch deploy order record (GET get_order_record; single request)",
		Long: `GET /device/get_order_record

Console also sends device_uuid with request_id. Pass --device-uuid when available.
This command returns one snapshot; Agents should re-call until execStatus settles.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetCloudService(ctx).GetOrderRecord(service.CloudOrderParams{
				RequestID:  requestID,
				DeviceUUID: deviceUUID,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&requestID, "request-id", "", "request_id from deploy (required)")
	cmd.Flags().StringVar(&deviceUUID, "device-uuid", "", "device_uuid (recommended; matches Console)")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode")
	_ = cmd.MarkFlagRequired("request-id")
	return cmd
}
