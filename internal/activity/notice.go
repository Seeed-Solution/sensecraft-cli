// Package activity reminds the user about ongoing SenseCraft activities.
// The list endpoint needs no Access Key. A missing parameter list is the
// activities that have not ended, so the CLI does not filter start and end times.
package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"sensecraft-cloud-terminal/internal/enum"
	"sensecraft-cloud-terminal/internal/update"
)

const (
	cacheTTL     = 24 * time.Hour
	fetchTimeout = 5 * time.Second
	stateFile    = "activity-state.json"
	maxBody      = 1 << 20
	envNoNotify  = "SENSECRAFT_CLI_NO_ACTIVITY_NOTIFIER"
)

// Item is one row from /openapi/activity/list.
type Item struct {
	ID             string `json:"activity_id"`
	Name           string `json:"name"`
	GuideURL       string `json:"guide_url"`
	DescriptionURL string `json:"description_url"`
	StartTime      string `json:"start_time"`
	EndTime        string `json:"end_time"`
}

// Link is the page to open when the user wants to join.
func (it Item) Link() string {
	if it.GuideURL != "" {
		return it.GuideURL
	}
	return it.DescriptionURL
}

type state struct {
	FetchedAt int64    `json:"fetched_at"`
	ShownAt   int64    `json:"shown_at,omitempty"`
	Dismissed []string `json:"dismissed,omitempty"`
	List      []Item   `json:"activities"`
}

// fetchActivities is replaced in tests.
var fetchActivities = fetchFromNetwork

// activityEndpoint is replaced in tests that need a local server.
var activityEndpoint = Endpoint

func fetchFromNetwork(ctx context.Context) ([]Item, error) {
	endpoint, err := activityEndpoint()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sensecraft-cli")
	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("activity list: HTTP %d", resp.StatusCode)
	}
	return parseList(body)
}

// Endpoint is {AI host}/openapi/activity/list. SENSECRAFT_AI_HOST replaces the gateway.
func Endpoint() (string, error) {
	host, err := enum.EffectiveAIHost("")
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(host, "/") + "/openapi" + enum.AIApiPathActivityList, nil
}

// ShouldSkip reports whether activity reminders stay quiet.
func ShouldSkip() bool {
	if os.Getenv(envNoNotify) != "" {
		return true
	}
	for _, key := range []string{"CI", "BUILD_NUMBER", "RUN_ID"} {
		if os.Getenv(key) != "" {
			return true
		}
	}
	return false
}

// Due returns activities to mention. The same reminder is due at most once
// per 24 hours. A fresh list is fetched only when that window has elapsed
// and the cached list is older than 24 hours.
func Due(now time.Time) []Item {
	if ShouldSkip() {
		return nil
	}
	st, _ := loadState()
	if st != nil && st.ShownAt > 0 && now.Sub(time.Unix(st.ShownAt, 0)) < cacheTTL {
		return nil
	}
	fresh := st != nil && st.FetchedAt > 0 && now.Sub(time.Unix(st.FetchedAt, 0)) < cacheTTL
	if !fresh {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		items, err := fetchActivities(ctx)
		cancel()
		if st == nil {
			st = &state{}
		}
		st.FetchedAt = now.Unix()
		if err != nil {
			if st.List == nil {
				st.List = []Item{}
			}
			_ = saveState(st)
			return nil
		}
		st.List = items
		_ = saveState(st)
	}
	if st == nil {
		return nil
	}
	items := visible(st)
	if len(items) == 0 {
		return nil
	}
	return items
}

// MarkShown records that the reminder was printed.
func MarkShown(now time.Time) error {
	st, err := loadState()
	if err != nil || st == nil {
		return err
	}
	st.ShownAt = now.Unix()
	return saveState(st)
}

// Acknowledge hides these activity ids from later reminders.
func Acknowledge(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	st, err := loadState()
	if err != nil || st == nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, id := range st.Dismissed {
		seen[id] = struct{}{}
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		st.Dismissed = append(st.Dismissed, id)
	}
	return saveState(st)
}

// Format renders one stderr line per activity.
func Format(items []Item) string {
	lines := make([]string, 0, len(items))
	for _, it := range items {
		name := it.Name
		if name == "" {
			name = it.ID
		}
		lines = append(lines, formatNotice(name, it.StartTime, it.EndTime, it.Link()))
	}
	return strings.Join(lines, "\n")
}

// activityDates are calendar days in UTC+8. The list API stores those
// midnights and end-of-day instants as unix seconds.
var activityZone = time.FixedZone("CST", 8*60*60)

func formatNotice(name, startRaw, endRaw, link string) string {
	text := "Notice: SenseCraft AI is hosting the " + name + " activity"
	if start, end, ok := formatActivityRange(startRaw, endRaw); ok {
		text += " from " + start + " to " + end
	}
	text += ". You can take part by creating and publishing apps, or draw inspiration from other developers."
	if link != "" {
		text += " Visit " + link + " to learn more and join in."
	}
	return text
}

func formatActivityRange(startRaw, endRaw string) (string, string, bool) {
	start, okStart := formatActivityDay(startRaw)
	end, okEnd := formatActivityDay(endRaw)
	if !okStart || !okEnd {
		return "", "", false
	}
	return start, end, true
}

func formatActivityDay(raw string) (string, bool) {
	sec, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return "", false
	}
	return time.Unix(sec, 0).In(activityZone).Format("January 2"), true
}

// IDs returns the activity ids that can be acknowledged.
func IDs(items []Item) []string {
	ids := make([]string, 0, len(items))
	for _, it := range items {
		if it.ID != "" {
			ids = append(ids, it.ID)
		}
	}
	return ids
}

func visible(st *state) []Item {
	dismissed := map[string]struct{}{}
	for _, id := range st.Dismissed {
		dismissed[id] = struct{}{}
	}
	out := make([]Item, 0, len(st.List))
	for _, it := range st.List {
		it.ID = strings.TrimSpace(it.ID)
		it.Name = strings.TrimSpace(it.Name)
		it.GuideURL = strings.TrimSpace(it.GuideURL)
		it.DescriptionURL = strings.TrimSpace(it.DescriptionURL)
		if it.ID == "" && it.Name == "" {
			continue
		}
		if it.ID != "" {
			if _, ok := dismissed[it.ID]; ok {
				continue
			}
		}
		out = append(out, it)
	}
	return out
}

func parseList(body []byte) ([]Item, error) {
	var envelope struct {
		Code json.RawMessage `json:"code"`
		Msg  string          `json:"msg"`
		Data struct {
			List []Item `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if code, ok := parseCode(envelope.Code); ok && code != 0 {
		return nil, fmt.Errorf("activity list code=%d %s", code, envelope.Msg)
	}
	if envelope.Data.List == nil {
		return []Item{}, nil
	}
	return envelope.Data.List, nil
}

func parseCode(raw json.RawMessage) (int, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err == nil {
			return n, true
		}
	}
	return 0, false
}

func statePath() string {
	return filepath.Join(update.ConfigDir(), stateFile)
}

func loadState() (*state, error) {
	data, err := os.ReadFile(statePath())
	if err != nil {
		return nil, err
	}
	var st state
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func saveState(st *state) error {
	dir := update.ConfigDir()
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
