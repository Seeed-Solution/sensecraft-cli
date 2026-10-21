package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"sensecraft-cloud-terminal/cmd/app"
	"sensecraft-cloud-terminal/cmd/auth"
	"sensecraft-cloud-terminal/cmd/cloud"
	"sensecraft-cloud-terminal/cmd/data"
	"sensecraft-cloud-terminal/cmd/device"
	"sensecraft-cloud-terminal/cmd/file"
	"sensecraft-cloud-terminal/cmd/group"
	"sensecraft-cloud-terminal/cmd/image"
	"sensecraft-cloud-terminal/cmd/model"
	"sensecraft-cloud-terminal/cmd/recamera"
	cmdupdate "sensecraft-cloud-terminal/cmd/update"
	"sensecraft-cloud-terminal/internal/activity"
	"sensecraft-cloud-terminal/internal/ui"
	"sensecraft-cloud-terminal/internal/update"
	"sensecraft-cloud-terminal/internal/version"

	"github.com/spf13/cobra"
)

var (
	startUpdateRefresh              = update.StartRefresh
	updateNotice                    = update.CachedNotice
	markUpdateShown                 = update.MarkShown
	noticeWriter          io.Writer = os.Stderr
	activityDue                     = activity.Due
	markActivityShown               = activity.MarkShown
	acknowledgeActivities           = activity.Acknowledge
	activityInteractive             = func() bool { return ui.DefaultConsole().IsInteractive() }
	activityConfirm                 = confirmActivity
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
SenseCraft AI OpenAPI:   https://api-gateway.seeed.cc/openapi
Console:                 https://sensecraft.seeed.cc`,
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
	rootCmd.AddCommand(image.NewImageCmd())
	rootCmd.AddCommand(cloud.NewCloudCmd())
	rootCmd.AddCommand(cmdupdate.NewCmd())
	return rootCmd
}

func Execute() error {
	return execute(os.Args[1:])
}

func execute(args []string) error {
	startUpdateRefresh(version.Version)
	root := GetRootCmd()
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	maybeNotify(args)
	return nil
}

func maybeNotify(args []string) {
	if !notifyUpdate(args) {
		return
	}
	if msg := updateNotice(version.Version); msg != "" {
		fmt.Fprintln(noticeWriter, msg)
		_ = markUpdateShown()
	}
	items := activityDue(time.Now())
	if len(items) == 0 {
		return
	}
	if text := activity.Format(items); text != "" {
		fmt.Fprintln(noticeWriter, text)
	}
	_ = markActivityShown(time.Now())
	if activityInteractive == nil || !activityInteractive() {
		return
	}
	ignore, err := activityConfirm()
	if err != nil || !ignore {
		return
	}
	_ = acknowledgeActivities(activity.IDs(items))
}

func confirmActivity() (bool, error) {
	fmt.Fprint(noticeWriter, "Ignore this activity? [y/N]: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false, err
	}
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes", nil
}

func notifyUpdate(args []string) bool {
	if len(args) > 0 && args[0] == "update" {
		return false
	}
	for _, arg := range args {
		if arg == "--version" || arg == "-v" {
			return false
		}
	}
	return true
}
