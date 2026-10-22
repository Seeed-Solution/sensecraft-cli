// Package update implements `sensecraft-cli update`.
package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"sensecraft-cloud-terminal/internal/selfupdate"
	"sensecraft-cloud-terminal/internal/update"
	"sensecraft-cloud-terminal/internal/version"

	"github.com/spf13/cobra"
)

const (
	fetchTimeout   = 15 * time.Second
	installTimeout = 10 * time.Minute
)

// Options is the update flow, with Fetch and Updater replaced in tests.
type Options struct {
	Current string
	Check   bool
	Force   bool
	Stderr  io.Writer
	Fetch   func(context.Context) (string, error)
	Updater *selfupdate.Updater
}

// NewCmd returns the update command.
func NewCmd() *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update sensecraft-cli to the latest npm release",
		Long: `Install the latest @seeed-studio/sensecraft-cli release from npm.

When this binary was installed via npm and npm is on PATH, the command runs:
  npm install -g @seeed-studio/sensecraft-cli@<version>

That install refreshes the packaged binary and skills even when the local version
string already matches the registry. Access Key config is left in place.
A binary that was not installed via npm prints the manual commands instead.

Use --check to only report whether a newer release exists.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			opts.Current = version.Version
			opts.Stderr = cmd.ErrOrStderr()
			opts.Fetch = update.FetchLatest
			opts.Updater = selfupdate.New()
			return Run(ctx, *opts)
		},
	}
	cmd.Flags().BoolVar(&opts.Check, "check", false, "only check for updates, do not install")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "accepted for compatibility; update always reinstalls")
	return cmd
}

// Run checks the registry and, unless --check, installs via npm when possible.
func Run(ctx context.Context, opts Options) error {
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.Fetch == nil {
		return errors.New("update: missing fetch")
	}
	if opts.Updater == nil {
		opts.Updater = selfupdate.New()
	}

	fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	latest, err := opts.Fetch(fetchCtx)
	if err != nil {
		return fmt.Errorf("检查最新版本失败: %w", err)
	}
	if !update.Valid(latest) {
		return fmt.Errorf("registry 返回的版本号无效: %s", latest)
	}
	if err := update.Remember(latest); err != nil {
		fmt.Fprintf(opts.Stderr, "警告: 无法记录版本检查结果: %v\n", err)
	}

	newer := update.IsNewer(latest, opts.Current)
	if opts.Check {
		return reportCheck(opts, latest, newer)
	}

	opts.Updater.CleanupStaleFiles()
	detect := opts.Updater.Detect()
	if !detect.CanAutoUpdate() {
		writeManual(opts.Stderr, latest, detect)
		return nil
	}

	if newer {
		fmt.Fprintf(opts.Stderr, "正在通过 npm 将 sensecraft-cli %s 更新到 %s ...\n", opts.Current, latest)
	} else {
		fmt.Fprintf(opts.Stderr, "正在通过 npm 重新安装 sensecraft-cli %s ...\n", latest)
	}
	restore, err := opts.Updater.PrepareSelfReplace()
	if err != nil {
		return err
	}
	installCtx, installCancel := context.WithTimeout(ctx, installTimeout)
	defer installCancel()
	if err := opts.Updater.Install(installCtx, latest); err != nil {
		restore()
		writeInstallFailure(opts.Stderr, err)
		return fmt.Errorf("npm 安装失败: %w", err)
	}
	if err := opts.Updater.Verify(ctx, latest); err != nil {
		restore()
		return fmt.Errorf("新版本校验失败: %w", err)
	}
	if newer {
		fmt.Fprintf(opts.Stderr, "sensecraft-cli 已从 %s 更新到 %s\n", opts.Current, latest)
	} else {
		fmt.Fprintf(opts.Stderr, "sensecraft-cli %s 已重新安装\n", latest)
	}
	return nil
}

func reportCheck(opts Options, latest string, newer bool) error {
	if !newer {
		fmt.Fprintf(opts.Stderr, "已是最新版本: %s\n", opts.Current)
		return nil
	}
	fmt.Fprintf(opts.Stderr, "发现新版本: %s → %s\n", opts.Current, latest)
	detect := opts.Updater.Detect()
	if detect.CanAutoUpdate() {
		fmt.Fprintln(opts.Stderr, "运行 sensecraft-cli update 安装")
		return nil
	}
	writeManual(opts.Stderr, latest, detect)
	return nil
}

func writeManual(w io.Writer, latest string, detect selfupdate.DetectResult) {
	fmt.Fprintf(w, "当前安装不能自动更新（%s）。\n", detect.Reason())
	fmt.Fprintf(w, "请运行:\n  npm install -g %s@%s\n  sensecraft-cli --version\n", selfupdate.NpmPackage, latest)
}

func writeInstallFailure(w io.Writer, err error) {
	var inst *selfupdate.InstallError
	if !errors.As(err, &inst) || inst.Output == "" {
		return
	}
	fmt.Fprint(w, inst.Output)
	if !strings.HasSuffix(inst.Output, "\n") {
		fmt.Fprintln(w)
	}
	if strings.Contains(inst.Output, "EACCES") {
		fmt.Fprintln(w, "权限不足。可改用 sudo sensecraft-cli update，或调整 npm 全局目录权限。")
	}
}
