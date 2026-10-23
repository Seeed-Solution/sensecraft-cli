package service

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"sensecraft-cloud-terminal/internal/app"
)

// RecameraClassicUniformType is Console classic reCamera (not Pro 53).
const RecameraClassicUniformType = 40

const recameraListProbePages = 3
const recameraListProbeLength = 20

type RecameraService struct {
	apps       AppService
	models     ModelService
	sandbox    bool
	remoteType string
}

func GetRecameraService(ctx app.AICmdContext) RecameraService {
	apps := GetAppService(ctx)
	return RecameraService{
		apps:       apps,
		models:     GetModelService(ctx),
		sandbox:    ctx.SandBox,
		remoteType: ctx.RemoteType(),
	}
}

type RecameraCandidate struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Source string `json:"source"`
}

type RecameraCheckResult struct {
	OK         bool                `json:"ok"`
	Device     int                 `json:"device"`
	Kind       string              `json:"kind"`
	ID         string              `json:"id"`
	Name       string              `json:"name,omitempty"`
	Task       any                 `json:"task,omitempty"`
	BePublic   any                 `json:"be_public,omitempty"`
	Flash      string              `json:"flash"`
	ConsoleURL string              `json:"console_url,omitempty"`
	AskPick    bool                `json:"ask_pick"`
	Candidates []RecameraCandidate `json:"candidates,omitempty"`
	Reason     string              `json:"reason,omitempty"`
	Next       string              `json:"next,omitempty"`
}

type RecameraOpenResult struct {
	OK         bool   `json:"ok"`
	Opened     bool   `json:"opened"`
	PrintOnly  bool   `json:"print_only"`
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	ConsoleURL string `json:"console_url"`
	Flash      string `json:"flash"`
	Next       string `json:"next"`
}

func recameraFlashNext(kind string) string {
	return fmt.Sprintf("Open console_url and click Deploy on the %s page. Connect classic reCamera (uniform_type 40) over IP or USB. Do not use cloud deploy. Do not download or clone public models.", kind)
}

func (s RecameraService) Check(kind, id string) (RecameraCheckResult, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	id = strings.TrimSpace(id)
	switch kind {
	case "model":
		return s.checkModel(id)
	case "app":
		return s.checkApp(id)
	default:
		return RecameraCheckResult{}, fmt.Errorf("kind must be model or app")
	}
}

func (s RecameraService) checkModel(modelID string) (RecameraCheckResult, error) {
	if modelID == "" {
		return RecameraCheckResult{}, fmt.Errorf("model_id is required")
	}
	raw, err := s.models.ViewModel(ModelViewParams{ModelID: modelID})
	if err != nil {
		return RecameraCheckResult{}, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return RecameraCheckResult{}, err
	}
	name, _ := m["name"].(string)
	ok := payloadContainsUniformType(m["uniform_types"], RecameraClassicUniformType)
	res := RecameraCheckResult{
		OK:         ok,
		Device:     RecameraClassicUniformType,
		Kind:       "model",
		ID:         modelID,
		Name:       name,
		Task:       m["task"],
		BePublic:   m["be_public"],
		Flash:      "local",
		ConsoleURL: RecameraModelDeployURL(s.remoteType, modelID),
		Candidates: []RecameraCandidate{{
			Kind:   "model",
			ID:     modelID,
			Name:   name,
			Source: "view",
		}},
	}
	if ok {
		res.Next = recameraFlashNext("model")
	} else {
		res.Reason = "uniform_types does not contain 40 (classic reCamera)"
	}
	return res, nil
}

func (s RecameraService) checkApp(appID string) (RecameraCheckResult, error) {
	if appID == "" {
		return RecameraCheckResult{}, fmt.Errorf("app_id is required")
	}
	raw, err := s.apps.ViewApp(AppViewParams{AppID: appID, ViewSketch: true})
	if err != nil {
		return RecameraCheckResult{}, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return RecameraCheckResult{}, err
	}
	name, _ := m["app_name"].(string)
	bePublic := m["be_public"]
	mine := true
	if n, ok := asInt(bePublic); ok && n != 0 {
		mine = false
	}
	candidates := extractRecameraModelCandidates(m)
	listed, err := s.appListedForRecamera(appID)
	if err != nil {
		return RecameraCheckResult{}, err
	}
	ok := listed || len(candidates) > 0
	res := RecameraCheckResult{
		OK:         ok,
		Device:     RecameraClassicUniformType,
		Kind:       "app",
		ID:         appID,
		Name:       name,
		BePublic:   bePublic,
		Flash:      "local",
		ConsoleURL: RecameraAppDeployURL(s.remoteType, appID, name, mine),
		Candidates: candidates,
		AskPick:    len(candidates) > 1,
	}
	if !ok {
		res.Reason = "app is not listed for uniform_type 40 and has no deploy-ai-model block with device_type 40"
		return res, nil
	}
	if res.AskPick {
		res.Next = "Multiple reCamera models in this app. Ask the user which firmware/model to flash, then open that model page or the app page."
	} else {
		res.Next = recameraFlashNext("app")
	}
	return res, nil
}

func (s RecameraService) appListedForRecamera(appID string) (bool, error) {
	for _, public := range []bool{true, false} {
		for page := 1; page <= recameraListProbePages; page++ {
			raw, err := s.apps.ListApps(AppListParams{
				Public:      public,
				Page:        page,
				Length:      recameraListProbeLength,
				UniformType: fmt.Sprintf("%d", RecameraClassicUniformType),
			})
			if err != nil {
				return false, err
			}
			if listContainsAppID(raw, appID) {
				return true, nil
			}
			if listPageCount(raw) <= page {
				break
			}
		}
	}
	return false, nil
}

func (s RecameraService) Open(kind, id string, printOnly bool) (RecameraOpenResult, RecameraCheckResult, error) {
	check, err := s.Check(kind, id)
	if err != nil {
		return RecameraOpenResult{}, check, err
	}
	if !check.OK {
		return RecameraOpenResult{
			OK:         false,
			Kind:       check.Kind,
			ID:         check.ID,
			ConsoleURL: check.ConsoleURL,
			Flash:      "local",
			Next:       check.Reason,
		}, check, fmt.Errorf("%s", check.Reason)
	}
	url := check.ConsoleURL
	opened := false
	if !printOnly && url != "" && !s.sandbox {
		opened = openConsoleInBrowser(url) == nil
	}
	return RecameraOpenResult{
		OK:         true,
		Opened:     opened,
		PrintOnly:  printOnly || s.sandbox,
		Kind:       check.Kind,
		ID:         check.ID,
		ConsoleURL: url,
		Flash:      "local",
		Next:       check.Next,
	}, check, nil
}

func payloadContainsUniformType(v any, want int) bool {
	switch t := v.(type) {
	case []any:
		for _, item := range t {
			if n, ok := asInt(item); ok && n == want {
				return true
			}
		}
	case []int:
		for _, n := range t {
			if n == want {
				return true
			}
		}
	default:
		if n, ok := asInt(v); ok {
			return n == want
		}
	}
	return false
}

func extractRecameraModelCandidates(app map[string]any) []RecameraCandidate {
	dep, _ := app["deployment"].(map[string]any)
	if dep == nil {
		return nil
	}
	blocks, _ := dep["blocks"].([]any)
	var out []RecameraCandidate
	seen := map[string]bool{}
	for _, raw := range blocks {
		block, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := block["type"].(string)
		if typ != "deploy-ai-model" {
			continue
		}
		cfg, _ := block["config"].(map[string]any)
		if cfg == nil {
			continue
		}
		if n, ok := asInt(cfg["device_type"]); !ok || n != RecameraClassicUniformType {
			continue
		}
		id, name := recameraModelFromBlockConfig(cfg)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, RecameraCandidate{Kind: "model", ID: id, Name: name, Source: "block"})
	}
	return out
}

func recameraModelFromBlockConfig(cfg map[string]any) (id, name string) {
	modelWrap, _ := cfg["model"].(map[string]any)
	if modelWrap == nil {
		return "", ""
	}
	inner, _ := modelWrap["model"].(map[string]any)
	if inner == nil {
		inner = modelWrap
	}
	id, _ = inner["id"].(string)
	name, _ = inner["name"].(string)
	if id == "" {
		if n, ok := asInt(inner["id"]); ok {
			id = fmt.Sprintf("%d", n)
		}
	}
	return strings.TrimSpace(id), strings.TrimSpace(name)
}

func listContainsAppID(raw json.RawMessage, appID string) bool {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	list, _ := m["list"].([]any)
	want := strings.TrimSpace(appID)
	for _, item := range list {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := row["app_id"].(string)
		if strings.TrimSpace(id) == want {
			return true
		}
	}
	return false
}

func listPageCount(raw json.RawMessage) int {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return 1
	}
	page, _ := m["page"].(map[string]any)
	if n, ok := asInt(page["count"]); ok && n > 0 {
		return n
	}
	return 1
}

func openConsoleInBrowser(u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	return cmd.Start()
}
