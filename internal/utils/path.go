package utils

import (
	"os"
	"path/filepath"
)

// JoinPath joins path elements with the OS separator.
func JoinPath(elem ...string) string {
	return filepath.Join(elem...)
}

// AbsPath returns the absolute path. An empty path stays empty.
func AbsPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	return abs, nil
}

// WorkingDir returns the current working directory.
func WorkingDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(wd), nil
}

// ResolveFilePath resolves a config file path.
//
// Rules:
//  1. A non-empty configFile is the target. An absolute path is used as-is; a relative path is joined to baseDir.
//  2. An empty configFile uses defaultFileName.
//  3. A non-empty baseDir is the directory. An empty baseDir uses the working directory.
func ResolveFilePath(configFile, baseDir, defaultFileName string) (string, error) {
	dir := baseDir
	if dir == "" {
		wd, err := WorkingDir()
		if err != nil {
			return "", err
		}
		dir = wd
	} else if !filepath.IsAbs(dir) {
		wd, err := WorkingDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(wd, dir)
	}

	file := configFile
	if file == "" {
		file = defaultFileName
	}

	if filepath.IsAbs(file) {
		return filepath.Clean(file), nil
	}
	return filepath.Clean(filepath.Join(dir, file)), nil
}
