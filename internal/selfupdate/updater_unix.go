//go:build !windows

package selfupdate

// PrepareSelfReplace is a no-op on Unix. A running binary can be replaced
// because the process keeps the old inode open.
func (u *Updater) PrepareSelfReplace() (func(), error) {
	return func() {}, nil
}

// CleanupStaleFiles is a no-op on Unix.
func (u *Updater) CleanupStaleFiles() {}
