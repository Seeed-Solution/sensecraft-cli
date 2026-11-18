package config

import (
	"os"
	"path/filepath"
	"testing"

	"sensecraft-cloud-terminal/internal/enum"
)

func TestRemoteTypeOf(t *testing.T) {
	if got := RemoteTypeOf(nil); got != "" {
		t.Fatalf("nil=%q", got)
	}
	if got := RemoteTypeOf(&CliConfig{}); got != "" {
		t.Fatalf("empty=%q", got)
	}
	got := RemoteTypeOf(&CliConfig{
		UseApi: "k",
		AppConfigList: []AccessApiConfig{{
			KeyId:      "k",
			RemoteType: " Global ",
		}},
	})
	if got != "Global" {
		t.Fatalf("got %q", got)
	}
	if got := RemoteTypeOf(&CliConfig{UseApi: "missing"}); got != "" {
		t.Fatalf("missing=%q", got)
	}
}

func TestResolveConfigPath(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv(enum.EnvNameConfigFile, "")
	t.Setenv(enum.EnvNameConfigPath, "")

	got, err := ResolveConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(wd, DefaultConfigFileName)
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestResolveConfigPathWithEnv(t *testing.T) {
	t.Setenv(enum.EnvNameConfigFile, "my.json")
	t.Setenv(enum.EnvNameConfigPath, t.TempDir())

	got, err := ResolveConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(os.Getenv(enum.EnvNameConfigPath), "my.json")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLoadAndSaveCliConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultConfigFileName)

	cfg := &CliConfig{
		UseApi: "key-id",
		AppConfigList: []AccessApiConfig{
			{
				RemoteType: "Global",
				KeyId:      "key-id",
				AccessKey:  "access-key",
			},
		},
	}
	if err := SaveCliConfigTo(path, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := loadCliConfigFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.UseApi != cfg.UseApi {
		t.Fatalf("UseApi = %q, want %q", loaded.UseApi, cfg.UseApi)
	}
	if len(loaded.AppConfigList) != 1 || loaded.AppConfigList[0].KeyId != "key-id" {
		t.Fatalf("unexpected config: %#v", loaded)
	}
}

func TestLoadCliConfigMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	cfg, err := loadCliConfigFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil empty config")
	}
	if len(cfg.AppConfigList) != 0 {
		t.Fatalf("expected empty config, got %#v", cfg)
	}
}
