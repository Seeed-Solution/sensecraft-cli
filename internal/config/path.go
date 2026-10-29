package config

import (
	"os"

	"sensecraft-cloud-terminal/internal/enum"
	"sensecraft-cloud-terminal/internal/utils"
)

// DefaultConfigFileName is the config file name without a directory.
const DefaultConfigFileName = "sensecraft.json"

// ResolveConfigPath returns the absolute path of the CLI config file.
//
// Priority:
//  1. SENSECRAFT_CONFIG_FILE: a config file path, absolute or relative.
//     A relative path is based on SENSECRAFT_CLI_PATH, or the working directory when that is unset.
//  2. SENSECRAFT_CLI_PATH alone: {SENSECRAFT_CLI_PATH}/sensecraft.json
//  3. Neither set: {working directory}/sensecraft.json
func ResolveConfigPath() (string, error) {
	configFile := os.Getenv(enum.EnvNameConfigFile)
	configDir := os.Getenv(enum.EnvNameConfigPath)
	return utils.ResolveFilePath(configFile, configDir, DefaultConfigFileName)
}
