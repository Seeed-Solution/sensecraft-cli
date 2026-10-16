package config

import (
	"errors"
	"fmt"
	"os"

	"sensecraft-cloud-terminal/internal/utils"
)

// LoadCliConfig 加载 CLI 配置文件。
//
// 配置文件不存在时返回空的 CliConfig（便于首次使用）；其他读取/解析错误会返回 error。
func loadCliConfig() (*CliConfig, error) {
	path, err := ResolveConfigPath()
	if err != nil {
		return nil, err
	}
	return loadCliConfigFrom(path)
}

// loadCliConfigFrom 从指定路径加载 CLI 配置。
func loadCliConfigFrom(path string) (*CliConfig, error) {
	path, err := utils.AbsPath(path)
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}

	if !utils.Exists(path) {
		return &CliConfig{}, nil
	}

	cfg := &CliConfig{}
	if err := utils.ReadJSON(path, cfg); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &CliConfig{}, nil
		}
		return nil, fmt.Errorf("load config %s: %w", path, err)
	}
	return cfg, nil
}

// SaveCliConfig 将配置写入默认解析路径。
func SaveCliConfig(cfg *CliConfig) error {
	path, err := ResolveConfigPath()
	if err != nil {
		return err
	}
	return SaveCliConfigTo(path, cfg)
}

// SaveCliConfigTo 将配置写入指定路径。
func SaveCliConfigTo(path string, cfg *CliConfig) error {
	if cfg == nil {
		return errors.New("config cannot be nil")
	}
	path, err := utils.AbsPath(path)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	if err := utils.WriteJSON(path, cfg, true); err != nil {
		return fmt.Errorf("save config %s: %w", path, err)
	}
	return nil
}
