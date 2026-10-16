package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFilePath(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		configFile  string
		baseDir     string
		defaultName string
		want        string
	}{
		{
			name:        "default in working dir",
			configFile:  "",
			baseDir:     "",
			defaultName: "sensecraft.json",
			want:        filepath.Join(wd, "sensecraft.json"),
		},
		{
			name:        "custom dir with default name",
			configFile:  "",
			baseDir:     "/tmp/cli-config",
			defaultName: "sensecraft.json",
			want:        filepath.Join("/tmp/cli-config", "sensecraft.json"),
		},
		{
			name:        "absolute config file",
			configFile:  "/etc/sensecraft/app.json",
			baseDir:     "/ignored",
			defaultName: "sensecraft.json",
			want:        "/etc/sensecraft/app.json",
		},
		{
			name:        "relative config file in base dir",
			configFile:  "custom.json",
			baseDir:     "/tmp/cli-config",
			defaultName: "sensecraft.json",
			want:        filepath.Join("/tmp/cli-config", "custom.json"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveFilePath(tt.configFile, tt.baseDir, tt.defaultName)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteAndReadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	payload := map[string]string{"hello": "world"}
	if err := WriteJSON(path, payload, true); err != nil {
		t.Fatal(err)
	}

	var loaded map[string]string
	if err := ReadJSON(path, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded["hello"] != "world" {
		t.Fatalf("unexpected payload: %#v", loaded)
	}
}
