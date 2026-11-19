//go:build !windows

package utils

func restrictToCurrentUser(string) error {
	return nil
}
