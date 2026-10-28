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

// appSiteOrigin is the CodeCraft Hub app site. API calls stay on the AI host.
// A test AI host (name containing "test-", including remote_type Develop) maps to the test hub.
func appSiteOrigin(remoteType string) string {
	host, err := enum.EffectiveAIHost(remoteType)
	if err == nil && strings.Contains(strings.ToLower(host), "test-") {
		return "https://test-codecraft.seeed.cc/hub"
	}
	return "https://codecraft.seeed.cc/hub"
}

// appTitleSlug matches the app site formatToTitleCase used in /view-application/{id}-{slug}.
func appTitleSlug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || (r >= 0x4e00 && r <= 0x9fff) {
			b.WriteRune(r)
			dash = false
			continue
		}
		if b.Len() > 0 && !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func appViewPath(appID, appName string) string {
	id := url.PathEscape(strings.TrimSpace(appID))
	slug := appTitleSlug(appName)
	if slug == "" {
		return "/view-application/" + id
	}
	return "/view-application/" + id + "-" + slug
}

// AppEditConsoleURL is the app-site editor: /application/edit?id=<app_id>
func AppEditConsoleURL(remoteType, appID string) string {
	id := strings.TrimSpace(appID)
	if id == "" {
		return ""
	}
	q := url.Values{}
	q.Set("id", id)
	return appSiteOrigin(remoteType) + "/application/edit?" + q.Encode()
}

// AppDetailConsoleURL is the owner detail route. from=user-center loads the sketch
// and the detail back button returns to the creator center.
// from=my-apps is the older plaza "my apps" list; do not use it for new links.
func AppDetailConsoleURL(remoteType, appID, appName string) string {
	id := strings.TrimSpace(appID)
	if id == "" {
		return ""
	}
	q := url.Values{}
	q.Set("tab", "custom")
	q.Set("from", "user-center")
	return appSiteOrigin(remoteType) + appViewPath(id, appName) + "?" + q.Encode()
}

// AppPlazaDetailConsoleURL is the public plaza snapshot (no from=my-apps).
func AppPlazaDetailConsoleURL(remoteType, appID, appName string) string {
	id := strings.TrimSpace(appID)
	if id == "" {
		return ""
	}
	q := url.Values{}
	q.Set("tab", "public")
	q.Set("from", "application-store")
	return appSiteOrigin(remoteType) + appViewPath(id, appName) + "?" + q.Encode()
}

// RecameraModelDeployURL is the AI deploy page for classic reCamera (uniform_type 40).
// Console model/detail.vue opens RecameraWorkspaceConnectPanel (IP/USB) on Deploy;
// there is no /ai/device/local/40, and /ai/recamera 301s to Grove Vision (36).
func RecameraModelDeployURL(remoteType, modelID string) string {
	return ModelDetailConsoleURL(remoteType, modelID)
}

// RecameraAppDeployURL is the app detail page where deploy-ai-model can open the
// same RecameraWorkspaceConnectPanel. Public plaza apps omit from=my-apps.
func RecameraAppDeployURL(remoteType, appID, appName string, mine bool) string {
	if mine {
		return AppDetailConsoleURL(remoteType, appID, appName)
	}
	return AppPlazaDetailConsoleURL(remoteType, appID, appName)
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
