//go:build windows

package selfupdate

import (
	"fmt"
	"os"
)

// PrepareSelfReplace renames the running executable to .old so npm can write
// the new binary. Windows does not allow replacing a locked exe in place.
// The returned function moves .old back when installation fails.
func (u *Updater) PrepareSelfReplace() (func(), error) {
	noop := func() {}
	exe, err := u.resolvedExe()
	if err != nil || exe == "" {
		return noop, nil
	}
	oldPath := exe + ".old"
	_ = os.Remove(oldPath)
	if err := os.Rename(exe, oldPath); err != nil {
		return noop, fmt.Errorf("cannot rename binary for update: %w", err)
	}
	restore := func() {
		if _, err := os.Stat(oldPath); err != nil {
			return
		}
		_ = os.Remove(exe)
		_ = os.Rename(oldPath, exe)
	}
	return restore, nil
}

// CleanupStaleFiles removes a leftover .old binary, or restores it when the
// real executable disappeared mid-update.
func (u *Updater) CleanupStaleFiles() {
	exe, err := u.resolvedExe()
	if err != nil || exe == "" {
		return
	}
	oldPath := exe + ".old"
	if _, err := os.Stat(oldPath); err != nil {
		return
	}
	if _, err := os.Stat(exe); err != nil {
		_ = os.Rename(oldPath, exe)
		return
	}
	_ = os.Remove(oldPath)
}
