// Package version exposes build metadata injected at link time.
//
// Release builds set these via -ldflags. script/build.sh reads
// packages/cli/package.json "version" unless VERSION is already set.
// A plain go build leaves Version as "dev" (with a local Date stamp) and
// does not check for updates.
//
//	go build -ldflags "-X sensecraft-cloud-terminal/internal/version.Version=1.2.3 \
//	  -X sensecraft-cloud-terminal/internal/version.Commit=$(git rev-parse --short HEAD) \
//	  -X sensecraft-cloud-terminal/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
package version

import (
	"runtime"
	"strings"
)

// Values are overwritten by -ldflags on release builds; the defaults mark a
// local build so a bug report can never be mistaken for a published artifact.
var (
	Version = "dev"
	Commit  = "none"
	// Date stamps a plain `go build` so `--version` can tell local binaries apart.
	Date = "2026-11-13 11:20"
)

type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: Date,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
}

// Line is what `sensecraft-cli --version` prints. Local/dev builds append Date
// (e.g. "dev 2026-11-13 11:20"); release builds keep the bare package version.
func Line() string {
	v := strings.TrimSpace(Version)
	d := strings.TrimSpace(Date)
	if v == "dev" && d != "" && d != "unknown" {
		return v + " " + d
	}
	return v
}

// String includes the commit and platform. `--version` uses Line instead.
func (i Info) String() string {
	base := i.Version
	if i.Version == "dev" && strings.TrimSpace(i.BuildDate) != "" && i.BuildDate != "unknown" {
		base = i.Version + " " + i.BuildDate
	}
	return base + " (" + i.Commit + ", " + i.Platform + ")"
}
