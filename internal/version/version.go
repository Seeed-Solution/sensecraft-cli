// Package version exposes build metadata injected at link time.
//
// Release builds set these via -ldflags, e.g.
//
//	go build -ldflags "-X sensecraft-cloud-terminal/internal/version.Version=1.2.3 \
//	  -X sensecraft-cloud-terminal/internal/version.Commit=$(git rev-parse --short HEAD) \
//	  -X sensecraft-cloud-terminal/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
package version

import "runtime"

// Values are overwritten by -ldflags on release builds; the defaults mark a
// local build so a bug report can never be mistaken for a published artifact.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
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

// String is the one-line form used by `--version`.
func (i Info) String() string {
	return i.Version + " (" + i.Commit + ", " + i.Platform + ")"
}
