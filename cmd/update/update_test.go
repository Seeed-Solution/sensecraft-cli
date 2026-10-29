package update

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/selfupdate"
)

func testUpdater(auto bool) *selfupdate.Updater {
	path := "/tmp/sensecraft-cli"
	if auto {
		path = "/usr/lib/node_modules/@seeed-studio/sensecraft-cli/vendor/sensecraft-cli"
	}
	return selfupdate.NewFrom(selfupdate.Hooks{
		Executable:   func() (string, error) { return path, nil },
		EvalSymlinks: func(string) (string, error) { return path, nil },
		LookPath: func(name string) (string, error) {
			if name == "npm" || name == "sensecraft-cli" {
				return "/usr/bin/" + name, nil
			}
			return "", selfupdate.ErrNotFound
		},
	})
}

func TestRunCheckAndInstall(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SENSECRAFT_CLI_PATH", dir)

	var installed string
	updater := withInstall(func(_ context.Context, version string) error {
		installed = version
		return nil
	}, func(context.Context, string) (string, error) {
		return "sensecraft-cli version 0.2.0\n", nil
	})

	var buf bytes.Buffer
	err := Run(context.Background(), Options{
		Current: "0.1.4",
		Check:   true,
		Stderr:  &buf,
		Fetch:   func(context.Context) (string, error) { return "0.2.0", nil },
		Updater: updater,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "New version available: 0.1.4 -> 0.2.0") || !strings.Contains(buf.String(), "sensecraft-cli update") {
		t.Fatalf("check output:\n%s", buf.String())
	}
	if installed != "" {
		t.Fatal("check must not install")
	}

	buf.Reset()
	err = Run(context.Background(), Options{
		Current: "0.1.4",
		Stderr:  &buf,
		Fetch:   func(context.Context) (string, error) { return "0.2.0", nil },
		Updater: updater,
	})
	if err != nil {
		t.Fatal(err)
	}
	if installed != "0.2.0" || !strings.Contains(buf.String(), "Updated sensecraft-cli from 0.1.4 to 0.2.0") {
		t.Fatalf("install=%q output:\n%s", installed, buf.String())
	}
}

func TestRunReinstallsMatchingVersion(t *testing.T) {
	t.Setenv("SENSECRAFT_CLI_PATH", t.TempDir())
	var installed string
	updater := withInstall(func(_ context.Context, version string) error {
		installed = version
		return nil
	}, func(context.Context, string) (string, error) {
		return "sensecraft-cli version 0.2.0\n", nil
	})
	var buf bytes.Buffer
	err := Run(context.Background(), Options{
		Current: "0.2.0",
		Stderr:  &buf,
		Fetch:   func(context.Context) (string, error) { return "0.2.0", nil },
		Updater: updater,
	})
	if err != nil {
		t.Fatal(err)
	}
	if installed != "0.2.0" || !strings.Contains(buf.String(), "Reinstalled sensecraft-cli 0.2.0") {
		t.Fatalf("install=%q output:\n%s", installed, buf.String())
	}
	if strings.Contains(buf.String(), "Already up to date") {
		t.Fatal("update must reinstall when the version already matches")
	}

	buf.Reset()
	installed = ""
	err = Run(context.Background(), Options{
		Current: "0.2.0",
		Check:   true,
		Stderr:  &buf,
		Fetch:   func(context.Context) (string, error) { return "0.2.0", nil },
		Updater: updater,
	})
	if err != nil {
		t.Fatal(err)
	}
	if installed != "" || !strings.Contains(buf.String(), "Already up to date") {
		t.Fatalf("install=%q output:\n%s", installed, buf.String())
	}
}

func TestRunAlreadyLatestAndManual(t *testing.T) {
	t.Setenv("SENSECRAFT_CLI_PATH", t.TempDir())
	var buf bytes.Buffer
	err := Run(context.Background(), Options{
		Current: "0.2.0",
		Check:   true,
		Stderr:  &buf,
		Fetch:   func(context.Context) (string, error) { return "0.2.0", nil },
		Updater: testUpdater(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Already up to date") {
		t.Fatalf("output:\n%s", buf.String())
	}

	buf.Reset()
	err = Run(context.Background(), Options{
		Current: "0.1.0",
		Stderr:  &buf,
		Fetch:   func(context.Context) (string, error) { return "0.2.0", nil },
		Updater: testUpdater(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "cannot update itself") || !strings.Contains(buf.String(), "npm install -g") {
		t.Fatalf("output:\n%s", buf.String())
	}
}

func TestRunInstallFailure(t *testing.T) {
	t.Setenv("SENSECRAFT_CLI_PATH", t.TempDir())
	updater := withInstall(func(context.Context, string) error {
		return &selfupdate.InstallError{Err: errors.New("exit status 1"), Output: "npm ERR! EACCES"}
	}, nil)
	var buf bytes.Buffer
	err := Run(context.Background(), Options{
		Current: "0.1.4",
		Stderr:  &buf,
		Fetch:   func(context.Context) (string, error) { return "0.2.0", nil },
		Updater: updater,
	})
	if err == nil || !strings.Contains(buf.String(), "Permission denied") {
		t.Fatalf("err=%v output:\n%s", err, buf.String())
	}
}

func TestRunInvalidRegistryVersion(t *testing.T) {
	t.Setenv("SENSECRAFT_CLI_PATH", t.TempDir())
	err := Run(context.Background(), Options{
		Current: "0.1.4",
		Stderr:  ioDiscard(),
		Fetch:   func(context.Context) (string, error) { return "latest", nil },
		Updater: testUpdater(true),
	})
	if err == nil || !strings.Contains(err.Error(), "invalid version") {
		t.Fatalf("err=%v", err)
	}
}

func withInstall(install func(context.Context, string) error, version func(context.Context, string) (string, error)) *selfupdate.Updater {
	path := "/usr/lib/node_modules/@seeed-studio/sensecraft-cli/vendor/sensecraft-cli"
	return selfupdate.NewFrom(selfupdate.Hooks{
		Executable:    func() (string, error) { return path, nil },
		EvalSymlinks:  func(string) (string, error) { return path, nil },
		LookPath:      func(name string) (string, error) { return "/usr/bin/" + name, nil },
		Install:       install,
		VersionOutput: version,
	})
}

func ioDiscard() *bytes.Buffer { return &bytes.Buffer{} }
