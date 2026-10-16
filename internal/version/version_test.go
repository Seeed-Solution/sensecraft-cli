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
