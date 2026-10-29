package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/enum"
)

// activityNow is the clock for "in progress". Tests replace it.
var activityNow = time.Now

type ongoingActivity struct {
	ActivityID int64
	Name       string
	GuideURL   string
	TagIDs     []string
	StartTime  *int64
	EndTime    *int64
}

// activityView is the user-facing row. Tag ids stay on ongoingActivity.
type activityView struct {
	ActivityID int64  `json:"activity_id"`
	Name       string `json:"name"`
	GuideURL   string `json:"guide_url"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
}

// ListPublicActivities lists activities in progress. It does not send an Access Key.
func ListPublicActivities(remoteType string, sandbox bool) (json.RawMessage, error) {
	return AppService{remoteType: remoteType, sandbox: sandbox}.ListActivities()
}

func (s AppService) ListActivities() (json.RawMessage, error) {
	items, err := s.ongoingActivities()
	if err != nil {
		return nil, err
	}
	views := currentActivityViews(items, activityNow())
	raw, err := json.Marshal(map[string]any{"list": views})
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func currentActivityViews(items []ongoingActivity, now time.Time) []activityView {
	out := make([]activityView, 0)
	for _, it := range items {
		if it.StartTime == nil || it.EndTime == nil {
			continue
		}
		start := time.Unix(*it.StartTime, 0)
		end := time.Unix(*it.EndTime, 0)
		if now.Before(start) || now.After(end) {
			continue
		}
		out = append(out, activityView{
			ActivityID: it.ActivityID,
			Name:       it.Name,
			GuideURL:   it.GuideURL,
			StartTime:  start.UTC().Format(time.RFC3339),
			EndTime:    end.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func (s AppService) ongoingActivities() ([]ongoingActivity, error) {
	if s.sandbox {
		return sandboxOngoingActivities(), nil
	}
	c := s.client
	if c == nil {
		public, err := client.NewAIApiClientPublic(s.remoteType)
		if err != nil {
			return nil, err
		}
		c = public
	}
	data, err := c.Get(enum.AIApiPathActivityList, nil)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			return sandboxOngoingActivities(), nil
		}
		return nil, err
	}
	return normalizeActivityListPayload(data)
}

func sandboxOngoingActivities() []ongoingActivity {
	start := int64(1792339200)
	end := int64(1795190399)
	return []ongoingActivity{{
		ActivityID: 1,
		Name:       "Muse Gadget",
		GuideURL:   "https://test-codecraft.seeed.cc/hub/application?tab=public",
		TagIDs:     []string{"1"},
		StartTime:  &start,
		EndTime:    &end,
	}}
}

func normalizeActivityListPayload(data json.RawMessage) ([]ongoingActivity, error) {
	if len(data) == 0 || string(data) == "null" {
		return []ongoingActivity{}, nil
	}
	var wrap struct {
		List []map[string]any `json:"list"`
	}
	if err := json.Unmarshal(data, &wrap); err != nil {
		return nil, err
	}
	out := make([]ongoingActivity, 0, len(wrap.List))
	for _, raw := range wrap.List {
		item, ok := normalizeOngoingActivity(raw)
		if ok {
			out = append(out, item)
		}
	}
	return out, nil
}

func normalizeOngoingActivity(raw map[string]any) (ongoingActivity, bool) {
	if raw == nil {
		return ongoingActivity{}, false
	}
	id, ok := positiveInt64(raw["activity_id"])
	if !ok {
		return ongoingActivity{}, false
	}
	name, _ := raw["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return ongoingActivity{}, false
	}
	item := ongoingActivity{
		ActivityID: id,
		Name:       name,
		TagIDs:     uniquePositiveTagIDs(raw["tag_ids"]),
	}
	if guide, ok := raw["guide_url"].(string); ok {
		item.GuideURL = strings.TrimSpace(guide)
	}
	if n, ok := flexibleInt64(raw["start_time"]); ok {
		item.StartTime = &n
	}
	if n, ok := flexibleInt64(raw["end_time"]); ok {
		item.EndTime = &n
	}
	return item, true
}

func uniquePositiveTagIDs(raw any) []string {
	list, ok := raw.([]any)
	if !ok || list == nil {
		return []string{}
	}
	out := make([]string, 0, len(list))
	seen := map[string]struct{}{}
	for _, item := range list {
		n, ok := positiveInt64(item)
		if !ok {
			continue
		}
		s := strconv.FormatInt(n, 10)
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func flexibleInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case float64:
		if t == float64(int64(t)) {
			return int64(t), true
		}
	case json.Number:
		n, err := t.Int64()
		if err == nil {
			return n, true
		}
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		if err == nil {
			return n, true
		}
	case int:
		return int64(t), true
	case int64:
		return t, true
	}
	return 0, false
}

// rejectInactiveActivity stops update_app when the activity is not in the
// ongoing list. A failed list request is ignored so the server can still
// answer with 25000; an empty successful list is a rejection.
func (s AppService) rejectInactiveActivity(id int64) error {
	items, err := s.ongoingActivities()
	if err != nil {
		return nil
	}
	for _, item := range items {
		if item.ActivityID == id {
			return nil
		}
	}
	return fmt.Errorf("activity does not exist or has expired")
}

func allowCloneIsOne(v any) bool {
	n, ok := asInt(v)
	return ok && n == 1
}

func prefixRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

type AppCloneParams struct {
	AppID string
	Name  string
	Yes   bool
}

func (s AppService) CloneApp(p AppCloneParams) (json.RawMessage, error) {
	if err := requireYes(p.Yes); err != nil {
		return nil, err
	}
	appID := strings.TrimSpace(p.AppID)
	if appID == "" {
		return nil, fmt.Errorf("--id is required")
	}
	existing, err := s.loadExistingApp(appID)
	if err != nil {
		return nil, err
	}
	if n, ok := asInt(existing["allow_clone"]); ok && n == 0 {
		return nil, fmt.Errorf("cloning is not allowed (allow_clone=0)")
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		source := stringFromMap(existing, "app_name")
		name = "Clone_" + prefixRunes(source, 54)
	}
	if err := validateAppName(name); err != nil {
		return nil, err
	}
	data, err := s.client.Post(enum.AIApiPathAppClone, map[string]any{
		"app_id":   appID,
		"app_name": name,
	})
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			data = sandboxAppClone(name)
		} else {
			return nil, err
		}
	}
	if extractAppID(data) == "" {
		data = ensureAppIdentity(data, "", name)
	}
	if extractAppString(data, "app_name") == "" {
		var m map[string]any
		if err := json.Unmarshal(data, &m); err == nil && m != nil {
			m["app_name"] = name
			if b, err := json.Marshal(m); err == nil {
				data = b
			}
		}
	}
	return s.withAppConsoleURL(data), nil
}

func sandboxAppClone(name string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"app_id":   "sandbox-cloned-app",
		"app_name": name,
	})
	return raw
}

func (s AppService) WithdrawApp(p AppDeleteParams) (json.RawMessage, error) {
	if err := requireYes(p.Yes); err != nil {
		return nil, err
	}
	appID := strings.TrimSpace(p.AppID)
	if appID == "" {
		return nil, fmt.Errorf("--id is required")
	}
	existing, err := s.loadExistingApp(appID)
	if err != nil {
		return nil, err
	}
	if err := s.rejectUnownedAppWrite(appID, existing); err != nil {
		return nil, err
	}
	switch appMapBePublic(existing) {
	case 2:
	case 1:
		return nil, fmt.Errorf("app is already public (be_public=1); use app unpublish. 已公开请用 app unpublish")
	default:
		return nil, fmt.Errorf("app is private (be_public=0); nothing to withdraw. 私有应用无需撤回")
	}
	data, err := s.client.Post(enum.AIApiPathAppCallbackSketch, map[string]any{"app_id": appID})
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			data = json.RawMessage(`{}`)
		} else {
			return nil, err
		}
	}
	return attachAppJSONMeta(data, map[string]any{
		"app_id":    appID,
		"be_public": 0,
	}), nil
}

func (s AppService) MarkApp(appID string, opType int, yes bool) (json.RawMessage, error) {
	if err := requireYes(yes); err != nil {
		return nil, err
	}
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, fmt.Errorf("--id is required")
	}
	switch opType {
	case 0, 1, 2, 3, 4:
	default:
		return nil, fmt.Errorf("op_type must be 0, 1, 2, 3, or 4")
	}
	data, err := s.client.Post(enum.AIApiPathAppMark, map[string]any{
		"app_id":  appID,
		"op_type": opType,
	})
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			data = json.RawMessage(`{}`)
		} else {
			return nil, err
		}
	}
	return ensureMarkResult(data, appID, opType), nil
}

func ensureMarkResult(data json.RawMessage, appID string, opType int) json.RawMessage {
	var m map[string]any
	if len(data) == 0 || string(data) == "null" || json.Unmarshal(data, &m) != nil || m == nil {
		m = map[string]any{}
	}
	if _, ok := m["app_id"]; !ok || strings.TrimSpace(fmt.Sprint(m["app_id"])) == "" {
		m["app_id"] = appID
	}
	if _, ok := m["op_type"]; !ok {
		m["op_type"] = opType
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return data
	}
	return raw
}
