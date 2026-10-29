package config

import (
	"errors"
	"fmt"
	"os"

	"sensecraft-cloud-terminal/internal/utils"
)

// LoadCliConfig loads the CLI config file.
//
// A missing file returns an empty CliConfig so first use can continue. Other read or parse errors are returned.
func loadCliConfig() (*CliConfig, error) {
	path, err := ResolveConfigPath()
	if err != nil {
		return nil, err
	}
	return loadCliConfigFrom(path)
}

// loadCliConfigFrom loads the CLI config from path.
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

// SaveCliConfig writes the config to the resolved default path.
func SaveCliConfig(cfg *CliConfig) error {
	path, err := ResolveConfigPath()
	if err != nil {
		return err
	}
	return SaveCliConfigTo(path, cfg)
}

// SaveCliConfigTo writes the config to path.
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
