package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"sensecraft-cloud-terminal/internal/enum"
)

// Console AI SPA is served under /ai/ in production (routerBase).
const consoleAIPathPrefix = "/ai"

func consoleAIOrigin(string) string {
	return strings.TrimSuffix(enum.DomainNameAIConsole, "/")
}

// ModelDetailConsoleURL matches Console post-create navigation:
// /model/detail?id=<id>&tab=custom → https://…/ai/model/detail?id=…&tab=custom
func ModelDetailConsoleURL(remoteType, modelID string) string {
	id := strings.TrimSpace(modelID)
	if id == "" {
		return ""
	}
	q := url.Values{}
	q.Set("id", id)
	q.Set("tab", "custom")
	return fmt.Sprintf("%s%s/model/detail?%s", consoleAIOrigin(remoteType), consoleAIPathPrefix, q.Encode())
}

// AppEditConsoleURL matches Console post-create navigation:
// /application/edit?id=<app_id>
func AppEditConsoleURL(remoteType, appID string) string {
	id := strings.TrimSpace(appID)
	if id == "" {
		return ""
	}
	q := url.Values{}
	q.Set("id", id)
	return fmt.Sprintf("%s%s/application/edit?%s", consoleAIOrigin(remoteType), consoleAIPathPrefix, q.Encode())
}

// AppDetailConsoleURL is the owner detail route. Console detail.vue uses
// view_sketch=0 unless query from=my-apps (then sketch). CLI create/restore
// writes the sketch; without from=my-apps the page fetches the empty published
// snapshot and the cover is missing.
func AppDetailConsoleURL(remoteType, appID string) string {
	id := strings.TrimSpace(appID)
	if id == "" {
		return ""
	}
	q := url.Values{}
	q.Set("tab", "custom")
	q.Set("from", "my-apps")
	return fmt.Sprintf("%s%s/application/detail/%s?%s", consoleAIOrigin(remoteType), consoleAIPathPrefix, url.PathEscape(id), q.Encode())
}

// AppPlazaDetailConsoleURL is the public plaza snapshot (no from=my-apps).
func AppPlazaDetailConsoleURL(remoteType, appID string) string {
	id := strings.TrimSpace(appID)
	if id == "" {
		return ""
	}
	return fmt.Sprintf("%s%s/application/detail/%s", consoleAIOrigin(remoteType), consoleAIPathPrefix, url.PathEscape(id))
}

// RecameraModelDeployURL is the AI deploy page for classic reCamera (uniform_type 40).
// Console model/detail.vue opens RecameraWorkspaceConnectPanel (IP/USB) on Deploy;
// there is no /ai/device/local/40, and /ai/recamera 301s to Grove Vision (36).
func RecameraModelDeployURL(remoteType, modelID string) string {
	return ModelDetailConsoleURL(remoteType, modelID)
}

// RecameraAppDeployURL is the app detail page where deploy-ai-model can open the
// same RecameraWorkspaceConnectPanel. Public plaza apps omit from=my-apps.
func RecameraAppDeployURL(remoteType, appID string, mine bool) string {
	if mine {
		return AppDetailConsoleURL(remoteType, appID)
	}
	return AppPlazaDetailConsoleURL(remoteType, appID)
}

func extractModelID(raw json.RawMessage) string {
	if id := extractAppString(raw, "model_id"); id != "" {
		return id
	}
	if id := extractAppString(raw, "id"); id != "" {
		return id
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	return ""
}

// attachConsoleURL merges console_url (and normalized id) into an API create payload.
// Bare string ids become {"model_id"|"app_id": id, "console_url": …, "data": id}.
func attachConsoleURL(raw json.RawMessage, idKey, id, consoleURL string) json.RawMessage {
	if id == "" || consoleURL == "" {
		return raw
	}
	out := map[string]any{}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err == nil && obj != nil {
		for k, v := range obj {
			out[k] = v
		}
	} else {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil && strings.TrimSpace(s) != "" {
			out["data"] = strings.TrimSpace(s)
		}
	}
	out[idKey] = id
	out["console_url"] = consoleURL
	encoded, err := json.Marshal(out)
	if err != nil {
		return raw
	}
	return encoded
}
