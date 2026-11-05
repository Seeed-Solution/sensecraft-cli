// Package uploadguard checks SenseCraft upload size on the CLI before
// acquire_file_url. Rate limits belong on the server; this package does not
// count or block upload frequency.
package uploadguard

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// Business caps from the OpenAPI upload spec. Every image is 5MB.
const (
	MaxAppImageBytes       int64 = 5 * 1024 * 1024
	MaxNonImageUploadBytes int64 = 100 * 1024 * 1024
)

var ErrEmptyUpload = errors.New("empty file uploads are not allowed (0 bytes)")

// StoredFileName returns the explicit stored name or the local basename with spaces removed.
// StoredFileName 返回指定的存储文件名，默认使用去除空格的本地文件名。
func StoredFileName(path, name string) (string, error) {
	if name == "" {
		name = strings.ReplaceAll(filepath.Base(path), " ", "")
	}
	if strings.TrimSpace(name) == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("invalid file name: use a file name without directory separators or control characters")
	}
	return name, nil
}

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
