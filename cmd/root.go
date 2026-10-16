package cmd

import (
	"fmt"
	"os"
	"sensecraft-cloud-terminal/cmd/app"
	"sensecraft-cloud-terminal/cmd/auth"
	"sensecraft-cloud-terminal/cmd/cloud"
	"sensecraft-cloud-terminal/cmd/data"
	"sensecraft-cloud-terminal/cmd/device"
	"sensecraft-cloud-terminal/cmd/file"
	"sensecraft-cloud-terminal/cmd/group"
	"sensecraft-cloud-terminal/cmd/model"
	"sensecraft-cloud-terminal/cmd/recamera"
	"sensecraft-cloud-terminal/internal/version"

	"github.com/spf13/cobra"
)

func GetRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "sensecraft-cli",
		Version: version.Get().String(),
		Short:   "SenseCAP / SenseCraft AI cloud CLI",
		Long: `Manage SenseCAP devices/data and SenseCraft AI apps/models via cloud APIs.

SenseCAP + SenseCraft AI OpenAPI use Access Key:
  sensecraft-cli auth init --kind sensecap

Global SenseCAP OpenAPI: https://sensecap.seeed.cc/openapi
SenseCraft AI OpenAPI:   https://sensecraft.seeed.cc/openapi`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(auth.NewAuthCmd())
	rootCmd.AddCommand(device.NewDeviceCmd())
	rootCmd.AddCommand(group.NewGroupCmd())
	rootCmd.AddCommand(data.NewDataCmd())
	rootCmd.AddCommand(app.NewAppCmd())
	rootCmd.AddCommand(model.NewModelCmd())
	rootCmd.AddCommand(recamera.NewRecameraCmd())
	rootCmd.AddCommand(file.NewFileCmd())
	rootCmd.AddCommand(cloud.NewCloudCmd())
	return rootCmd
}

func Execute() error {
	if err := GetRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
