// Package update checks the npm registry for a newer sensecraft-cli release
// and caches the result next to the CLI config.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"sensecraft-cloud-terminal/internal/enum"
)

const (
	registryURL = "https://registry.npmjs.org/@seeed-studio/sensecraft-cli/latest"
	cacheTTL    = 24 * time.Hour
	stateFile   = "update-state.json"
	maxBody     = 256 << 10
)

// Notice is the one-line hint printed after a command when a newer release exists.
func Notice(_, latest string) string {
	return fmt.Sprintf("Warning: A new version of sensecraft-cli (%s) has been released. Please run sensecraft-cli update to upgrade for more complete features and broader protection.", latest)
}

// Valid reports whether v is a comparable X.Y.Z version, with an optional
// prerelease suffix. A leading v is accepted.
func Valid(v string) bool {
	return parseVersion(v) != nil
}

// IsNewer reports whether latest should replace current.
// An unparsable current version is treated as older than any valid latest.
// An unparsable latest is never newer.
func IsNewer(latest, current string) bool {
	remote := parseVersion(latest)
	if remote == nil {
		return false
	}
	local := parseVersion(current)
	if local == nil {
		return true
	}
	for i := 0; i < 3; i++ {
		if remote.core[i] != local.core[i] {
			return remote.core[i] > local.core[i]
		}
	}
	return comparePrerelease(remote.pre, local.pre) > 0
}

// ShouldSkip reports whether background checks and notices must stay quiet.
// Local builds ("dev"), git-describe builds, CI, and an explicit opt-out skip.
func ShouldSkip(current string) bool {
	if os.Getenv("SENSECRAFT_CLI_NO_UPDATE_NOTIFIER") != "" {
		return true
	}
	if isCI() {
		return true
	}
	return !isReleased(current)
}

func isCI() bool {
	for _, key := range []string{"CI", "BUILD_NUMBER", "RUN_ID"} {
		if os.Getenv(key) != "" {
			return true
		}
	}
	return false
}

var gitDescribePattern = regexp.MustCompile(`-\d+-g[0-9a-f]{7,}`)

func isReleased(version string) bool {
	v := strings.TrimSpace(version)
	if v == "" || strings.EqualFold(v, "dev") || strings.Contains(v, "-dirty") {
		return false
	}
	if gitDescribePattern.MatchString(v) {
		return false
	}
	return parseVersion(v) != nil
}

type parsedVersion struct {
	core [3]int
	pre  string
}

func parseVersion(v string) *parsedVersion {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if i := strings.Index(v, "+"); i >= 0 {
		v = v[:i]
	}
	pre := ""
	if i := strings.Index(v, "-"); i >= 0 {
		pre = v[i+1:]
		v = v[:i]
		if pre == "" {
			return nil
		}
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return nil
	}
	var nums [3]int
	for i, p := range parts {
		if len(p) > 1 && p[0] == '0' {
			return nil
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return nil
		}
		nums[i] = n
	}
	return &parsedVersion{core: nums, pre: pre}
}

func comparePrerelease(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return 1
	}
	if b == "" {
		return -1
	}
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	n := len(ap)
	if len(bp) < n {
		n = len(bp)
	}
	for i := 0; i < n; i++ {
		an, aErr := strconv.Atoi(ap[i])
		bn, bErr := strconv.Atoi(bp[i])
		switch {
		case aErr == nil && bErr == nil && an != bn:
			if an > bn {
				return 1
			}
			return -1
		case aErr == nil && bErr != nil:
			return -1
		case aErr != nil && bErr == nil:
			return 1
		default:
			if ap[i] != bp[i] {
				if ap[i] > bp[i] {
					return 1
				}
				return -1
			}
		}
	}
	if len(ap) != len(bp) {
		if len(ap) > len(bp) {
			return 1
		}
		return -1
	}
	return 0
}

type updateState struct {
	LatestVersion string `json:"latest_version"`
	CheckedAt     int64  `json:"checked_at"`
	ShownAt       int64  `json:"shown_at,omitempty"`
	ShownVersion  string `json:"shown_version,omitempty"`
}

// ConfigDir is the directory that holds update-state.json.
// SENSECRAFT_CLI_PATH wins; otherwise the npm installer's default is used.
func ConfigDir() string {
	if dir := strings.TrimSpace(os.Getenv(enum.EnvNameConfigPath)); dir != "" {
		return dir
	}
	if runtime.GOOS == "windows" {
		base := os.Getenv("APPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return filepath.Join(os.TempDir(), "sensecraft-cli")
			}
			base = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(base, "sensecraft-cli")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "sensecraft-cli")
	}
	return filepath.Join(home, ".config", "sensecraft-cli")
}

func statePath() string {
	return filepath.Join(ConfigDir(), stateFile)
}

func loadState() (*updateState, error) {
	data, err := os.ReadFile(statePath())
	if err != nil {
		return nil, err
	}
	var state updateState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// Remember stores a successful registry check.
func Remember(latest string) error {
	return rememberAt(latest, time.Now())
}

func rememberAt(latest string, at time.Time) error {
	st := updateState{LatestVersion: latest, CheckedAt: at.Unix()}
	if prev, err := loadState(); err == nil && prev != nil {
		st.ShownAt = prev.ShownAt
		st.ShownVersion = prev.ShownVersion
	}
	return writeState(&st)
}

func writeState(st *updateState) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	target := statePath()
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}

// MarkShown records that the current cached release was printed.
func MarkShown() error {
	return markShownAt(time.Now())
}

func markShownAt(at time.Time) error {
	st, err := loadState()
	if err != nil || st == nil || st.LatestVersion == "" {
		return err
	}
	st.ShownAt = at.Unix()
	st.ShownVersion = st.LatestVersion
	return writeState(st)
}

// CachedNotice returns the hint for a cached newer release, or "" when
// nothing should be printed. The same latest version is printed at most
// once per 24 hours.
func CachedNotice(current string) string {
	if ShouldSkip(current) {
		return ""
	}
	state, err := loadState()
	if err != nil || state.LatestVersion == "" {
		return ""
	}
	if !IsNewer(state.LatestVersion, current) {
		return ""
	}
	if state.ShownVersion == state.LatestVersion && state.ShownAt > 0 && time.Since(time.Unix(state.ShownAt, 0)) < cacheTTL {
		return ""
	}
	return Notice(current, state.LatestVersion)
}

// StartRefresh begins a registry check when the cache is missing or older
// than 24 hours. It overlaps the command and never reports errors.
// WaitRefresh returns after that check has been stored. A fresh cache makes
// both calls return immediately, so a command is not delayed every time.
func StartRefresh(current string) {
	refreshMu.Lock()
	defer refreshMu.Unlock()
	refreshDone = nil
	if ShouldSkip(current) {
		return
	}
	state, err := loadState()
	if err == nil && state.LatestVersion != "" && time.Since(time.Unix(state.CheckedAt, 0)) < cacheTTL {
		return
	}
	done := make(chan struct{})
	refreshDone = done
	go func() {
		defer close(done)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		latest, err := fetchLatestVersion(ctx)
		if err != nil || parseVersion(latest) == nil {
			return
		}
		_ = Remember(latest)
	}()
}

// WaitRefresh blocks until the check started by StartRefresh is finished.
func WaitRefresh() {
	refreshMu.Lock()
	done := refreshDone
	refreshMu.Unlock()
	if done == nil {
		return
	}
	<-done
}

// fetchLatestVersion is the registry call used by StartRefresh. Tests replace it.
var fetchLatestVersion = FetchLatest

var (
	refreshMu   sync.Mutex
	refreshDone chan struct{}
)

func resetRefresh() {
	refreshMu.Lock()
	refreshDone = nil
	refreshMu.Unlock()
}

// FetchLatest reads the version field from the npm latest document.
func FetchLatest(ctx context.Context) (string, error) {
	return fetchLatest(ctx, registryURL)
}

func fetchLatest(ctx context.Context, endpoint string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sensecraft-cli")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("npm registry: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return "", err
	}
	var doc struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", err
	}
	if strings.TrimSpace(doc.Version) == "" {
		return "", fmt.Errorf("npm registry: empty version")
	}
	return doc.Version, nil
}
