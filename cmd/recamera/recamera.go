package recamera

import (
	"fmt"

	cmdctx "sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/service"

	"github.com/spf13/cobra"
)

func NewRecameraCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recamera",
		Short: "Classic reCamera (uniform_type 40) pick-and-flash helpers",
		Long: `Check whether an app or model targets classic reCamera, then open the Console AI deploy page.

Flash is local IP/USB in the browser (L3). This command does not download public models,
does not cloud-deploy, and does not talk to the device.`,
	}
	cmd.AddCommand(newRecameraCheckCmd())
	cmd.AddCommand(newRecameraOpenCmd())
	return cmd
}

func newRecameraCheckCmd() *cobra.Command {
	var (
		modelID string
		appID   string
		sandbox bool
	)
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Verify an app or model is classic reCamera (uniform_type 40)",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, id, err := recameraTarget(modelID, appID)
			if err != nil {
				return err
			}
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			result, err := service.GetRecameraService(ctx).Check(kind, id)
			if err != nil {
				return err
			}
			if printErr := ctx.PrintJsonData(result); printErr != nil {
				return printErr
			}
			if !result.OK {
				return fmt.Errorf("%s", result.Reason)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&modelID, "model-id", "", "Model id to check")
	cmd.Flags().StringVar(&appID, "app-id", "", "App id to check")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	return cmd
}

func newRecameraOpenCmd() *cobra.Command {
	var (
		modelID   string
		appID     string
		printOnly bool
		sandbox   bool
	)
	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open the Console AI deploy page for a checked reCamera app or model",
		Long: `Opens the model or app detail page where the user clicks Deploy and connects
classic reCamera over IP or USB. Use --print-only to print the URL without opening a browser.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, id, err := recameraTarget(modelID, appID)
			if err != nil {
				return err
			}
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			result, _, err := service.GetRecameraService(ctx).Open(kind, id, printOnly)
			if printErr := ctx.PrintJsonData(result); printErr != nil {
				return printErr
			}
			return err
		},
	}
	cmd.Flags().StringVar(&modelID, "model-id", "", "Model id to open")
	cmd.Flags().StringVar(&appID, "app-id", "", "App id to open")
	cmd.Flags().BoolVar(&printOnly, "print-only", false, "Print console_url only; do not open a browser")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; never open a browser")
	return cmd
}

func recameraTarget(modelID, appID string) (kind, id string, err error) {
	hasModel := modelID != ""
	hasApp := appID != ""
	if hasModel == hasApp {
		return "", "", fmt.Errorf("exactly one of --model-id or --app-id is required")
	}
	if hasModel {
		return "model", modelID, nil
	}
	return "app", appID, nil
}
