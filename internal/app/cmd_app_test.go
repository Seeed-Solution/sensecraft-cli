package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sensecraft-cloud-terminal/internal/enum"
	"testing"
)

func TestPrintWorkHome(t *testing.T) {
	fmt.Println(os.Getwd())
}

func TestCmdContext(t *testing.T) {
	configPath := filepath.Join(getRootPath(), "/test")
	fmt.Println(configPath)
	path, _ := os.Executable()
	fmt.Println(path)
	_ = os.Setenv(enum.EnvNameConfigFile, "config.json")
	_ = os.Setenv(enum.EnvNameConfigPath, configPath)

	cmdCtx, err := GetCmdContext()
	if err != nil {
		fmt.Printf("error while create cmd app: %v", err)
	} else {
		fmt.Printf("%#v\n", cmdCtx)
	}
}

func getRootPath() string {
	_, filePath, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filePath), "../../")
}
