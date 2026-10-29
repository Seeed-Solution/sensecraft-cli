package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const defaultFilePerm = 0o600

// ReadBytes reads the whole file.
func ReadBytes(path string) ([]byte, error) {
	path = filepath.Clean(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}
	return data, nil
}

// ReadText reads a UTF-8 text file.
func ReadText(path string) (string, error) {
	data, err := ReadBytes(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteBytes writes a file, creating parent directories and replacing the target atomically.
func WriteBytes(path string, data []byte, perm os.FileMode) error {
	path = filepath.Clean(path)
	if perm == 0 {
		perm = defaultFilePerm
	}

	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write temp file %s: %w", tmpPath, err)
	}
	if err := tmp.Chmod(perm); err != nil {
		cleanup()
		return fmt.Errorf("set file permissions %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		if removeErr := os.Remove(path); removeErr == nil {
			if renameErr := os.Rename(tmpPath, path); renameErr == nil {
				return nil
			}
		}
		cleanup()
		return fmt.Errorf("write file %s: %w", path, err)
	}
	return nil
}

// WriteText writes a UTF-8 text file.
func WriteText(path string, content string, perm os.FileMode) error {
	return WriteBytes(path, []byte(content), perm)
}

// ReadJSON reads a file and unmarshals JSON into v.
func ReadJSON(path string, v any) error {
	data, err := ReadBytes(path)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return errors.New("config file is empty")
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse JSON %s: %w", path, err)
	}
	return nil
}

// WriteJSON marshals v as JSON and writes it to a file.
func WriteJSON(path string, v any, indent bool) error {
	var data []byte
	var err error
	if indent {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	data = append(data, '\n')
	return WriteBytes(path, data, defaultFilePerm)
}

// Exists reports whether path exists.
func Exists(path string) bool {
	_, err := os.Stat(filepath.Clean(path))
	return err == nil
}

// IsNotExist reports whether err means the file does not exist.
func IsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

// EnsureDir creates the directory and its parents. An existing directory is not an error.
func EnsureDir(dir string) error {
	dir = filepath.Clean(dir)
	if dir == "" || dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	return nil
}
