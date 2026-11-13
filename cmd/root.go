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
	"sensecraft-cloud-terminal/cmd/firmware"
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
	waitUpdate                      = update.WaitRefresh
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
		Version: version.Line(),
		Short:   "SenseCraft cloud platform CLI",
		Long: `Terminal for the SenseCraft cloud platform.

SenseCraft Data: devices, groups, and telemetry (device, group, data)
SenseCraft AI:   model library and reCamera (model, recamera, cloud)
CodeCraft Hub:   applications (app)
                 https://codecraft.seeed.cc/hub

All three platforms are accessed with an open Access Key.
  sensecraft-cli auth init --site Global --no-wait
Show the verification URL and end the turn. In a terminal, sensecraft-cli auth init runs the wizard.
Create or manage a key: https://account.seeed.cc/account/access-keys

The China site supports SenseCraft Data only. Use Global for SenseCraft AI and CodeCraft Hub.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	rootCmd.AddCommand(auth.NewAuthCmd())
	rootCmd.AddCommand(device.NewDeviceCmd())
	rootCmd.AddCommand(group.NewGroupCmd())
	rootCmd.AddCommand(data.NewDataCmd())
	rootCmd.AddCommand(app.NewAppCmd())
	rootCmd.AddCommand(model.NewModelCmd())
	rootCmd.AddCommand(recamera.NewRecameraCmd())
	rootCmd.AddCommand(file.NewFileCmd())
	rootCmd.AddCommand(firmware.NewFirmwareCmd())
	rootCmd.AddCommand(image.NewImageCmd())
	rootCmd.AddCommand(cloud.NewCloudCmd())
	rootCmd.AddCommand(cmdupdate.NewCmd())
	return rootCmd
}

func Execute() error {
	return execute(os.Args[1:])
}

func execute(args []string) error {
	if notifyUpdate(args) {
		startUpdateRefresh(version.Version)
	}
	root := GetRootCmd()
	root.SetArgs(args)
	err := root.Execute()
	if notifyUpdate(args) {
		waitUpdate()
	}
	if err != nil {
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
