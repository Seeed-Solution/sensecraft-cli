package service

import (
	"encoding/json"
	"errors"
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
	Public      bool
	Mine        bool
	Name        string
	Scene       string
	UniformType string
	Page        int
	Length      int
	MarkType    string
}

func (s AppService) ListApps(p AppListParams) (json.RawMessage, error) {
	q := buildAppListQuery(p)

	data, err := s.client.Get(enum.AIApiPathAppList, q)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			return sandboxAppList(p), nil
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
	// Mine: omit be_public (Console custom tab). Public and Mine both false → same as mine/all-owned depending on backend.
	_ = p.Mine
	return q
}

func sandboxAppList(p AppListParams) json.RawMessage {
	bePublic := 0
	if p.Public {
		bePublic = 1
	}
	item := map[string]any{
		"app_id":    "sandbox-app-1",
		"app_name":  "Sandbox Vision App",
		"excerpt":   "Sandbox public application",
		"be_public": bePublic,
		"scene":     p.Scene,
	}
	if ut := strings.TrimSpace(p.UniformType); ut != "" {
		item["uniform_type"] = ut
	}
	raw, _ := json.Marshal(map[string]any{
		"list": []map[string]any{item},
		"page": map[string]any{"count": 1, "page": 1, "length": 20},
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
		"allow_clone":    0,
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
