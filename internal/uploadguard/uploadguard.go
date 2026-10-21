// Package uploadguard checks SenseCraft upload size on the CLI before
// acquire_file_url. Rate limits belong on the server; this package does not
// count or block upload frequency.
package uploadguard

import (
	"errors"
	"fmt"
)

// Business caps aligned with Console upload.vue / infoForm / model/add.vue.
const (
	MaxAppImageBytes       int64 = 5 * 1024 * 1024
	MaxModelImageBytes     int64 = 2 * 1024 * 1024
	MaxNonImageUploadBytes int64 = 100 * 1024 * 1024
)

var ErrEmptyUpload = errors.New("empty file uploads are not allowed (0 bytes)")

// ValidateOSSUploadSize mirrors Console upload.vue size gates.
// fileType: 1=image, 2–6=non-image. forAppImage selects 5MB vs 2MB for type 1.
func ValidateOSSUploadSize(fileType int, size int64, forAppImage bool) error {
	if size <= 0 {
		return ErrEmptyUpload
	}
	max := MaxNonImageUploadBytes
	if fileType == 1 {
		if forAppImage {
			max = MaxAppImageBytes
		} else {
			max = MaxModelImageBytes
		}
	}
	if size > max {
		if fileType == 1 {
			if forAppImage {
				return fmt.Errorf("app image exceeds Console limit of 5 MB (%d bytes); got %d bytes", max, size)
			}
			return fmt.Errorf("image exceeds Console limit of 2 MB (%d bytes); got %d bytes (model pic_url); use --for-app for app images up to 5 MB", max, size)
		}
		return fmt.Errorf("file exceeds Console limit of 100 MB (%d bytes); got %d bytes (file_type=%d)", max, size, fileType)
	}
	return nil
}
