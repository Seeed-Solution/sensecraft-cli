package config

import (
	"os"

	"sensecraft-cloud-terminal/internal/enum"
	"sensecraft-cloud-terminal/internal/utils"
)

// DefaultConfigFileName 默认配置文件名（不含目录）。
const DefaultConfigFileName = "sensecraft.json"

// ResolveConfigPath 解析 CLI 配置文件的完整路径。
//
// 优先级：
//  1. 环境变量 SENSECRAFT_CONFIG_FILE：配置文件路径（可为绝对或相对路径）。
//     相对路径时，基于 SENSECRAFT_CLI_PATH；若未设置则基于当前工作目录。
//  2. 仅设置 SENSECRAFT_CLI_PATH：{SENSECRAFT_CLI_PATH}/sensecraft.json
//  3. 均未设置：{当前工作目录}/sensecraft.json
func ResolveConfigPath() (string, error) {
	configFile := os.Getenv(enum.EnvNameConfigFile)
	configDir := os.Getenv(enum.EnvNameConfigPath)
	return utils.ResolveFilePath(configFile, configDir, DefaultConfigFileName)
}
