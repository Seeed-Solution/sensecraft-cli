package selfupdate

import (
	"context"
	"errors"
	"runtime"
	"testing"
)

func TestDetectNpmInstall(t *testing.T) {
	u := &Updater{
		executable: func() (string, error) { return "/tmp/sensecraft-cli", nil },
		evalSymlinks: func(string) (string, error) {
			return "/usr/lib/node_modules/@seeed-studio/sensecraft-cli/vendor/sensecraft-cli", nil
		},
		lookPath: func(name string) (string, error) {
			if name == "npm" {
				return "/usr/bin/npm", nil
			}
			return "", ErrNotFound
		},
	}
	got := u.Detect()
	if !got.CanAutoUpdate() {
		t.Fatalf("detect=%+v", got)
	}
	if got.Path == "" {
		t.Fatal("empty path")
	}
}

func TestDetectManualAndMissingNpm(t *testing.T) {
	manual := &Updater{
		executable:   func() (string, error) { return "/tmp/sensecraft-cli", nil },
		evalSymlinks: func(string) (string, error) { return "/tmp/sensecraft-cli", nil },
		lookPath:     func(string) (string, error) { return "/usr/bin/npm", nil },
	}
	got := manual.Detect()
	if got.CanAutoUpdate() || got.Reason() != "不是通过 npm 安装" {
		t.Fatalf("manual=%+v reason=%s", got, got.Reason())
	}

	noNpm := &Updater{
		executable:   func() (string, error) { return "/tmp/sensecraft-cli", nil },
		evalSymlinks: func(string) (string, error) { return "/usr/lib/node_modules/pkg/vendor/sensecraft-cli", nil },
		lookPath:     func(string) (string, error) { return "", ErrNotFound },
	}
	got = noNpm.Detect()
	if got.CanAutoUpdate() || got.Reason() != "已通过 npm 安装，但 PATH 中没有 npm" {
		t.Fatalf("no npm=%+v reason=%s", got, got.Reason())
	}
}

func TestVerifyReadsVersionOutput(t *testing.T) {
	u := &Updater{
		lookPath: func(name string) (string, error) {
			if name == "sensecraft-cli" {
				return "/usr/local/bin/sensecraft-cli", nil
			}
			return "", ErrNotFound
		},
		versionOut: func(context.Context, string) (string, error) {
			return "sensecraft-cli version 0.2.0 (abc, darwin/arm64)\n", nil
		},
	}
	if err := u.Verify(context.Background(), "0.2.0"); err != nil {
		t.Fatal(err)
	}
	u.versionOut = func(context.Context, string) (string, error) {
		return "sensecraft-cli version 0.1.4\n", nil
	}
	if err := u.Verify(context.Background(), "0.2.0"); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestInstallUsesOverride(t *testing.T) {
	var got string
	u := &Updater{
		install: func(_ context.Context, version string) error {
			got = version
			return &InstallError{Err: errors.New("exit 1"), Output: "EACCES"}
		},
	}
	err := u.Install(context.Background(), "0.2.0")
	if got != "0.2.0" {
		t.Fatalf("version=%s", got)
	}
	var inst *InstallError
	if !errors.As(err, &inst) || inst.Output != "EACCES" {
		t.Fatalf("err=%v", err)
	}
}

func TestPrepareSelfReplaceNoop(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PrepareSelfReplace renames the running executable")
	}
	restore, err := New().PrepareSelfReplace()
	if err != nil {
		t.Fatal(err)
	}
	restore()
	New().CleanupStaleFiles()
}
