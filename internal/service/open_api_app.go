package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/enum"
)

type AppService struct {
	client     *client.AIApiClient
	sandbox    bool
	remoteType string
	userID     string
}

func GetAppService(ctx app.AICmdContext) AppService {
	return AppService{
		client:     newAIHTTPClient(ctx),
		sandbox:    ctx.SandBox,
		remoteType: ctx.RemoteType(),
	}
}

type AppListParams struct {
	Public             bool
	Mine               bool
	Name               string
	Scene              string
	UniformType        string
	Page               int
	Length             int
	MarkType           string
	Sort               string
	PriorityActivityID string
	TagID              string
	Likes              bool
	Collections        bool
}

func (s AppService) ListApps(p AppListParams) (json.RawMessage, error) {
	var activities []ongoingActivity
	var actErr error
	if p.Sort == "popular" && strings.TrimSpace(p.PriorityActivityID) == "" {
		activities, actErr = s.ongoingActivities()
	}
	q, err := prepareAppListQuery(p, activities, actErr)
	if err != nil {
		return nil, err
	}
	data, err := s.client.Get(enum.AIApiPathAppList, q)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			return sandboxAppList(q), nil
		}
		return nil, err
	}
	return data, nil
}

type AppViewParams struct {
	AppID      string
	ViewSketch bool // true => view_sketch=1
}

func (s AppService) ViewApp(p AppViewParams) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("app_id", p.AppID)
	q.Set("access_type", "look")
	if p.ViewSketch {
		q.Set("view_sketch", "1")
	} else {
		q.Set("view_sketch", "0")
	}
	data, err := s.client.Get(enum.AIApiPathAppView, q)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			return sandboxAppView(p), nil
		}
		return nil, err
	}
	return data, nil
}

// buildAppListQuery mirrors GET /flow/app/v2/list_app.
// Device filter is query uniform_type (reCamera=40, XIAO ESP32S3=32).
func buildAppListQuery(p AppListParams) url.Values {
	q := url.Values{}
	page := p.Page
	if page <= 0 {
		page = 1
	}
	length := p.Length
	if length <= 0 {
		length = 20
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("length", strconv.Itoa(length))
	if p.Public {
		q.Set("be_public", "1")
	}
	// Console list.vue getAppList: params.app_name (not "search" / "name").
	if name := strings.TrimSpace(p.Name); name != "" {
		q.Set("app_name", name)
	}
	if scene := strings.TrimSpace(p.Scene); scene != "" {
		q.Set("scene", scene)
	}
	if ut := strings.TrimSpace(p.UniformType); ut != "" {
		q.Set("uniform_type", ut)
	}
	if p.MarkType != "" {
		q.Set("mark_type", p.MarkType)
	}
	if sort := strings.TrimSpace(p.Sort); sort != "" {
		q.Set("sort_by", sort)
	}
	if id := strings.TrimSpace(p.PriorityActivityID); id != "" {
		q.Set("priority_activity_id", id)
	}
	if tag := strings.TrimSpace(p.TagID); tag != "" {
		q.Set("tag_id", tag)
	}
	// Mine: omit be_public (Console custom tab). Public and Mine both false → same as mine/all-owned depending on backend.
	_ = p.Mine
	return q
}

func prepareAppListQuery(p AppListParams, activities []ongoingActivity, activitiesErr error) (url.Values, error) {
	switch p.Sort {
	case "", "popular", "newest", "likes":
	default:
		return nil, fmt.Errorf("--sort must be popular, newest, or likes")
	}
	mark, forcePublic, err := resolveListMark(p)
	if err != nil {
		return nil, err
	}
	if forcePublic {
		if p.Mine {
			return nil, fmt.Errorf("--likes and --collections cannot be combined with --mine")
		}
		p.Public = true
	}
	p.MarkType = mark

	priority := strings.TrimSpace(p.PriorityActivityID)
	switch p.Sort {
	case "newest", "likes":
		if priority != "" {
			return nil, fmt.Errorf("--priority-activity-id cannot be used with --sort %s", p.Sort)
		}
	case "popular":
		if priority != "" {
			n, ok := positiveInt64(priority)
			if !ok {
				return nil, fmt.Errorf("--priority-activity-id must be a positive integer")
			}
			priority = strconv.FormatInt(n, 10)
		} else if activitiesErr == nil && len(activities) > 0 {
			priority = strconv.FormatInt(activities[0].ActivityID, 10)
		} else {
			priority = ""
		}
	default:
		if priority != "" {
			return nil, fmt.Errorf("--priority-activity-id requires --sort popular")
		}
	}
	p.PriorityActivityID = priority
	p.TagID = strings.TrimSpace(p.TagID)
	return buildAppListQuery(p), nil
}

func resolveListMark(p AppListParams) (mark string, forcePublic bool, err error) {
	if p.Likes && p.Collections {
		return "", false, fmt.Errorf("use only one of --likes or --collections")
	}
	explicit := strings.TrimSpace(p.MarkType)
	switch {
	case p.Likes:
		mark = "1"
		forcePublic = true
	case p.Collections:
		mark = "0"
		forcePublic = true
	}
	if explicit != "" {
		if explicit != "0" && explicit != "1" {
			return "", false, fmt.Errorf("--mark-type must be 1 (my likes) or 0 (my collections)")
		}
		if mark != "" && mark != explicit {
			return "", false, fmt.Errorf("--mark-type %s does not match --likes/--collections", explicit)
		}
		mark = explicit
	}
	return mark, forcePublic, nil
}

func sandboxAppList(q url.Values) json.RawMessage {
	bePublic := 0
	if q.Get("be_public") == "1" {
		bePublic = 1
	}
	item := map[string]any{
		"app_id":    "sandbox-app-1",
		"app_name":  "Sandbox Vision App",
		"excerpt":   "Sandbox public application",
		"be_public": bePublic,
		"scene":     q.Get("scene"),
	}
	if ut := strings.TrimSpace(q.Get("uniform_type")); ut != "" {
		item["uniform_type"] = ut
	}
	raw, _ := json.Marshal(map[string]any{
		"list":          []map[string]any{item},
		"page":          map[string]any{"count": 1, "page": 1, "length": 20},
		"sandbox_query": q,
	})
	return raw
}

func sandboxAppView(p AppViewParams) json.RawMessage {
	sketch := 0
	if p.ViewSketch {
		sketch = 1
	}
	raw, _ := json.Marshal(map[string]any{
		"app_id":         p.AppID,
		"app_name":       "Sandbox Vision App",
		"excerpt":        "Sandbox application detail",
		"scene":          1,
		"allow_clone":    sandboxAllowClone(p.AppID),
		"allow_commerce": 0,
		"images": []any{
			map[string]any{"url": "https://example.invalid/sandbox-cover.jpg"},
		},
		"author":     sandboxAppAuthor(p.AppID),
		"user_id":    sandboxAppOwnerID(p.AppID),
		"be_public":  sandboxAppBePublic(p.AppID),
		"is_sketch":  sketch,
		"deployment": map[string]any{"blocks": sandboxRecameraAppBlocks(p.AppID)},
		"materials":  []any{},
	})
	return raw
}

func sandboxRecameraAppBlocks(appID string) []any {
	id := strings.ToLower(strings.TrimSpace(appID))
	if strings.Contains(id, "recamera-multi") {
		return []any{
			sandboxDeployAIModelBlock("recamera-model-a", RecameraClassicUniformType),
			sandboxDeployAIModelBlock("recamera-model-b", RecameraClassicUniformType),
		}
	}
	if strings.Contains(id, "recamera") {
		return []any{sandboxDeployAIModelBlock("recamera-model-a", RecameraClassicUniformType)}
	}
	return []any{}
}

func sandboxDeployAIModelBlock(modelID string, deviceType int) map[string]any {
	return map[string]any{
		"type": "deploy-ai-model",
		"config": map[string]any{
			"device_type": deviceType,
			"model": map[string]any{
				"model": map[string]any{"id": modelID, "name": modelID},
			},
		},
	}
}

func sandboxAllowClone(appID string) int {
	if strings.Contains(strings.ToLower(appID), "noclone") {
		return 0
	}
	return 1
}

func sandboxAppBePublic(appID string) int {
	id := strings.ToLower(strings.TrimSpace(appID))
	if sandboxIsForeignPublicID(id) {
		return 1
	}
	if strings.Contains(id, "review") {
		return 2
	}
	if strings.Contains(id, "published") {
		return 1
	}
	return 0
}

func sandboxAppOwnerID(appID string) string {
	if sandboxIsForeignPublicID(appID) {
		return "other-user"
	}
	return "user-sandbox"
}

func sandboxAppAuthor(appID string) string {
	if sandboxIsForeignPublicID(appID) {
		return "other-user"
	}
	return "sandbox-author"
}
