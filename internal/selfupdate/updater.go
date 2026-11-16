// Package selfupdate detects an npm install and replaces it with a newer package.
package selfupdate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const NpmPackage = "@seeed-studio/sensecraft-cli"

const (
	installMethodNpm    = "npm"
	installMethodManual = "manual"
	verifyTimeout       = 15 * time.Second
)

// DetectResult describes how the running binary was installed.
type DetectResult struct {
	Method       string
	Path         string
	NpmAvailable bool
}

// CanAutoUpdate reports whether npm install -g can replace this install.
func (d DetectResult) CanAutoUpdate() bool {
	return d.Method == installMethodNpm && d.NpmAvailable
}

// Reason explains why automatic update is unavailable.
func (d DetectResult) Reason() string {
	if d.Method == installMethodNpm && !d.NpmAvailable {
		return "已通过 npm 安装，但 PATH 中没有 npm"
	}
	return "不是通过 npm 安装"
}

// InstallError is a failed npm install, including combined command output.
type InstallError struct {
	Err    error
	Output string
}

func (e *InstallError) Error() string {
	if e.Err == nil {
		return "npm install failed"
	}
	return e.Err.Error()
}

func (e *InstallError) Unwrap() error { return e.Err }

// Updater performs install detection and npm replacement.
// Fields are replaced in tests.
type Updater struct {
	lookPath     func(string) (string, error)
	executable   func() (string, error)
	evalSymlinks func(string) (string, error)
	install      func(ctx context.Context, version string) error
	versionOut   func(ctx context.Context, path string) (string, error)
}

// Hooks replaces process and npm calls in tests. Nil fields keep the real behavior.
type Hooks struct {
	LookPath      func(string) (string, error)
	Executable    func() (string, error)
	EvalSymlinks  func(string) (string, error)
	Install       func(context.Context, string) error
	VersionOutput func(context.Context, string) (string, error)
}

// New returns an updater that uses the real process and npm.
func New() *Updater {
	return &Updater{
		lookPath:     exec.LookPath,
		executable:   os.Executable,
		evalSymlinks: filepath.EvalSymlinks,
		versionOut:   runVersion,
	}
}

// NewFrom returns an updater with the given test hooks.
func NewFrom(h Hooks) *Updater {
	u := New()
	if h.LookPath != nil {
		u.lookPath = h.LookPath
	}
	if h.Executable != nil {
		u.executable = h.Executable
	}
	if h.EvalSymlinks != nil {
		u.evalSymlinks = h.EvalSymlinks
	}
	if h.Install != nil {
		u.install = h.Install
	}
	if h.VersionOutput != nil {
		u.versionOut = h.VersionOutput
	}
	return u
}

// Detect decides whether this process is the binary shipped inside the npm package.
func (u *Updater) Detect() DetectResult {
	exe, err := u.executable()
	if err != nil {
		return DetectResult{Method: installMethodManual}
	}
	resolved, err := u.evalSymlinks(exe)
	if err != nil {
		resolved = exe
	}
	method := installMethodManual
	if strings.Contains(resolved, "node_modules") {
		method = installMethodNpm
	}
	npmOK := false
	if method == installMethodNpm {
		if _, err := u.lookPath("npm"); err == nil {
			npmOK = true
		}
	}
	return DetectResult{Method: method, Path: resolved, NpmAvailable: npmOK}
}

// Install runs npm install -g @seeed-studio/sensecraft-cli@version.
func (u *Updater) Install(ctx context.Context, version string) error {
	if u.install != nil {
		return u.install(ctx, version)
	}
	npmPath, err := u.lookPath("npm")
	if err != nil {
		return fmt.Errorf("npm not found: %w", err)
	}
	cmd := exec.CommandContext(ctx, npmPath, "install", "-g", NpmPackage+"@"+version)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			err = fmt.Errorf("npm install timed out: %w", err)
		}
		return &InstallError{Err: err, Output: buf.String()}
	}
	return nil
}

// Verify runs the sensecraft-cli on PATH and checks that it reports version.
func (u *Updater) Verify(ctx context.Context, version string) error {
	vctx, cancel := context.WithTimeout(ctx, verifyTimeout)
	defer cancel()
	path, err := u.lookPath("sensecraft-cli")
	if err != nil {
		return fmt.Errorf("sensecraft-cli not on PATH: %w", err)
	}
	out, err := u.versionOut(vctx, path)
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(out))
	}
	if !strings.Contains(out, version) {
		return fmt.Errorf("version output %q does not contain %s", strings.TrimSpace(out), version)
	}
	return nil
}

func runVersion(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, path, "--version")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Executable path after symlink resolution. Empty when the process path is unknown.
func (u *Updater) resolvedExe() (string, error) {
	exe, err := u.executable()
	if err != nil {
		return "", err
	}
	resolved, err := u.evalSymlinks(exe)
	if err != nil {
		return exe, nil
	}
	return resolved, nil
}

// ErrNotFound matches exec.ErrNotFound for tests that simulate a missing npm.
var ErrNotFound = errors.New("executable file not found")
