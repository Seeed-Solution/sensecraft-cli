package utils

import (
	"os"
	"path/filepath"
)

// JoinPath 使用操作系统路径规则拼接路径（跨平台）。
func JoinPath(elem ...string) string {
	return filepath.Join(elem...)
}

// AbsPath 将路径转为绝对路径；空字符串返回空。
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

// WorkingDir 返回当前工作目录。
func WorkingDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(wd), nil
}

// ResolveFilePath 解析配置文件路径。
//
// 规则：
//  1. configFile 非空时作为目标文件；绝对路径直接使用，相对路径基于 baseDir 拼接。
//  2. configFile 为空时使用 defaultFileName。
//  3. baseDir 非空时作为目录；为空时使用当前工作目录。
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
