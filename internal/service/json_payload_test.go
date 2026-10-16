package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAppAndModelName(t *testing.T) {
	if err := validateAppName(""); err == nil {
		t.Fatal("empty app name")
	}
	if err := validateAppName(string(make([]rune, 61))); err == nil {
		t.Fatal("app name too long")
	}
	if err := validateModelName("ok"); err != nil {
		t.Fatal(err)
	}
	if err := validateModelName(string(make([]rune, 65))); err == nil {
		t.Fatal("model name too long")
	}
}

func TestRejectForbiddenPublicFlags(t *testing.T) {
	if err := rejectForbiddenPublicFlags(map[string]any{"be_public": float64(1)}); err == nil {
		t.Fatal("expected be_public=1 rejected")
	}
	if err := rejectForbiddenPublicFlags(map[string]any{"privacy": float64(2)}); err != nil {
		t.Fatal(err)
	}
}

func TestLoadJSONObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.json")
	if err := os.WriteFile(path, []byte(`{"app_name":"Hi","excerpt":"e"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := LoadJSONObject(path)
	if err != nil {
		t.Fatal(err)
	}
	if m["app_name"] != "Hi" {
		t.Fatalf("got %#v", m)
	}
}
