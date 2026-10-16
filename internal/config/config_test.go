package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sensecraft-cloud-terminal/internal/enum"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	//_, filePath, _, _ := runtime.Caller(0)
	configPath := filepath.Join(getRootPath(), "/test")
	fmt.Println(configPath)
	path, _ := os.Executable()
	fmt.Println(path)
	//envs := os.Environ()
	_ = os.Setenv(enum.EnvNameConfigFile, "config.json")
	_ = os.Setenv(enum.EnvNameConfigPath, configPath)

	cmdConfig, err := loadCliConfig()
	if err != nil {
		fmt.Printf("error while loading config: %v", err)
	} else {
		fmt.Printf("%#v\n", cmdConfig)
	}
}

func getRootPath() string {
	_, filePath, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filePath), "../../")
}
