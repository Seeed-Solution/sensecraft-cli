package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const defaultFilePerm = 0o600

// ReadBytes 读取文件全部内容。
func ReadBytes(path string) ([]byte, error) {
	path = filepath.Clean(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}
	return data, nil
}

// ReadText 读取 UTF-8 文本文件。
func ReadText(path string) (string, error) {
	data, err := ReadBytes(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteBytes 写入文件；自动创建父目录，并使用临时文件原子替换。
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
	if ownerOnly(perm) {
		if err := restrictToCurrentUser(tmpPath); err != nil {
			cleanup()
			return fmt.Errorf("set file permissions %s: %w", path, err)
		}
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

// WriteText 写入 UTF-8 文本文件。
func WriteText(path string, content string, perm os.FileMode) error {
	return WriteBytes(path, []byte(content), perm)
}

// ReadJSON 从文件读取并反序列化 JSON。
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

// WriteJSON 将 v 序列化为 JSON 并写入文件。
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

// Exists 判断路径是否存在。
func Exists(path string) bool {
	_, err := os.Stat(filepath.Clean(path))
	return err == nil
}

// IsNotExist 判断错误是否为文件不存在。
func IsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

// ownerOnly 表示权限没有分给同组用户和其他用户。
func ownerOnly(perm os.FileMode) bool {
	return perm.Perm()&0o077 == 0
}

// EnsureDir 创建目录（含父级），已存在则忽略。
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
