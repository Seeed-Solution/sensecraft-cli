package version

import (
	"strings"
	"testing"
)

func TestGetDefaultsMarkLocalBuild(t *testing.T) {
	info := Get()
	if info.Version == "" {
		t.Fatal("Version empty")
	}
	if info.GoVersion == "" || !strings.HasPrefix(info.GoVersion, "go") {
		t.Fatalf("GoVersion=%q", info.GoVersion)
	}
	if !strings.Contains(info.Platform, "/") {
		t.Fatalf("Platform=%q", info.Platform)
	}
	s := info.String()
	if !strings.Contains(s, info.Version) || !strings.Contains(s, info.Platform) {
		t.Fatalf("String=%q", s)
	}
}

func TestLineAppendsLocalDevDate(t *testing.T) {
	origV, origD := Version, Date
	t.Cleanup(func() {
		Version, Date = origV, origD
	})

	Version, Date = "dev", "2026-11-13 11:20"
	if got := Line(); got != "dev 2026-11-13 11:20" {
		t.Fatalf("Line()=%q", got)
	}

	Version, Date = "0.4.1", "2026-11-13T03:20:00Z"
	if got := Line(); got != "0.4.1" {
		t.Fatalf("release Line() should stay bare version, got %q", got)
	}
}
