// Package uploadguard checks SenseCraft upload size on the CLI before
// acquire_file_url. Rate limits belong on the server; this package does not
// count or block upload frequency.
package uploadguard

import (
	"errors"
	"fmt"
)

// Business caps from the OpenAPI upload spec. Every image is 5MB.
const (
	MaxAppImageBytes       int64 = 5 * 1024 * 1024
	MaxNonImageUploadBytes int64 = 100 * 1024 * 1024
)

var ErrEmptyUpload = errors.New("empty file uploads are not allowed (0 bytes)")

// ValidateOSSUploadSize checks the local file before acquire_file_url.
// fileType 1 is an image (5MB). Types 2–6 are 100MB.
func ValidateOSSUploadSize(fileType int, size int64) error {
	if size <= 0 {
		return ErrEmptyUpload
	}
	max := MaxNonImageUploadBytes
	if fileType == 1 {
		max = MaxAppImageBytes
	}
	if size > max {
		if fileType == 1 {
			return fmt.Errorf("image exceeds limit of 5 MB (%d bytes); got %d bytes", max, size)
		}
		return fmt.Errorf("file exceeds limit of 100 MB (%d bytes); got %d bytes (file_type=%d)", max, size, fileType)
	}
	return nil
}
