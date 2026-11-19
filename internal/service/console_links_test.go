package service

import (
	"encoding/json"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/enum"
)

func TestModelDetailConsoleURL(t *testing.T) {
	got := ModelDetailConsoleURL("Global", "abc123")
	want := "https://sensecraft.seeed.cc/ai/model/detail?id=abc123&tab=custom"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	dev := ModelDetailConsoleURL("China", "m1")
	if !strings.HasPrefix(dev, "https://sensecraft.seeed.cc/ai/model/detail?") {
		t.Fatalf("china host unexpected: %s", dev)
	}
	if ModelDetailConsoleURL("Global", "  ") != "" {
		t.Fatal("empty id should yield empty url")
	}
}

func TestAppConsoleURLs(t *testing.T) {
	edit := AppEditConsoleURL("Global", "app-1")
	wantEdit := "https://codecraft.seeed.cc/hub/application/edit?id=app-1"
	if edit != wantEdit {
		t.Fatalf("edit got %q want %q", edit, wantEdit)
	}
	detail := AppDetailConsoleURL("China", "app-1", "reTerminal Sticky")
	wantDetail := "https://codecraft.seeed.cc/hub/view-application/app-1-reterminal-sticky"
	if detail != wantDetail {
		t.Fatalf("detail got %q want %q", detail, wantDetail)
	}
	if AppDetailConsoleURL("Global", "app-1", "") != "https://codecraft.seeed.cc/hub/view-application/app-1" {
		t.Fatal("empty name must not invent a slug")
	}
}

func TestAttachConsoleURLBareString(t *testing.T) {
	raw := json.RawMessage(`"sandbox-created-model"`)
	out := attachConsoleURL(raw, "model_id", "sandbox-created-model",
		ModelDetailConsoleURL("Global", "sandbox-created-model"))
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["model_id"] != "sandbox-created-model" {
		t.Fatalf("model_id=%v", m["model_id"])
	}
	url, _ := m["console_url"].(string)
	if !strings.Contains(url, "/ai/model/detail?id=sandbox-created-model") {
		t.Fatalf("console_url=%v", m["console_url"])
	}
	if m["data"] != "sandbox-created-model" {
		t.Fatalf("data=%v", m["data"])
	}
}

func TestModelCreateAttachesConsoleURLInSandbox(t *testing.T) {
	s := ModelService{sandbox: true, remoteType: "Global"}
	out := s.withModelConsoleURL(sandboxModelCreate(nil))
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["model_id"] != "sandbox-created-model" {
		t.Fatalf("got %#v", m)
	}
	url, _ := m["console_url"].(string)
	if !strings.HasPrefix(url, "https://sensecraft.seeed.cc/ai/model/detail?") {
		t.Fatalf("console_url=%q", url)
	}
}

func TestAppCreateAttachesConsoleURLsInSandbox(t *testing.T) {
	s := AppService{remoteType: "Global"}
	out := s.withAppConsoleURL(sandboxAppCreate("SmokeApp"))
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["app_id"] != "sandbox-created-app" {
		t.Fatalf("got %#v", m)
	}
	edit, _ := m["console_url"].(string)
	detail, _ := m["detail_url"].(string)
	if edit != "https://codecraft.seeed.cc/hub/application/edit?id=sandbox-created-app" {
		t.Fatalf("console_url=%q", edit)
	}
	if detail != "https://codecraft.seeed.cc/hub/view-application/sandbox-created-app-smokeapp" {
		t.Fatalf("detail_url=%q", detail)
	}
}

func TestAppSiteIgnoresAIHost(t *testing.T) {
	t.Setenv(enum.EnvNameAIHost, "https://test-gateway.example")
	t.Setenv(enum.EnvNameAppSite, "")
	t.Setenv(enum.EnvNameConsoleHost, "")
	got := AppEditConsoleURL("Develop", "app-1")
	if got != "https://codecraft.seeed.cc/hub/application/edit?id=app-1" {
		t.Fatalf("got %s", got)
	}
	if model := ModelDetailConsoleURL("Develop", "abc"); !strings.HasPrefix(model, "https://sensecraft.seeed.cc/ai/model/detail?") {
		t.Fatalf("model url must stay on the AI console, got %s", model)
	}
}

func TestAppSiteOriginUsesAppSiteEnv(t *testing.T) {
	t.Setenv(enum.EnvNameAppSite, "https://test-codecraft.seeed.cc/hub/")
	got := AppEditConsoleURL("Global", "app-1")
	if got != "https://test-codecraft.seeed.cc/hub/application/edit?id=app-1" {
		t.Fatalf("got %s", got)
	}
	if detail := AppDetailConsoleURL("Global", "app-1", "Smoke"); detail != "https://test-codecraft.seeed.cc/hub/view-application/app-1-smoke" {
		t.Fatalf("detail=%s", detail)
	}
	t.Setenv(enum.EnvNameAppSite, "https://hub.example/hub")
	if edit := AppEditConsoleURL("Global", "app-1"); edit != "https://codecraft.seeed.cc/hub/application/edit?id=app-1" {
		t.Fatalf("foreign app site must stay on the public hub, got %s", edit)
	}
}

func TestConsoleURLIgnoresAIHostOverride(t *testing.T) {
	t.Setenv(enum.EnvNameAIHost, "https://ai.test.example")
	t.Setenv(enum.EnvNameConsoleHost, "")
	got := ModelDetailConsoleURL("Global", "abc")
	if !strings.HasPrefix(got, "https://sensecraft.seeed.cc/ai/model/detail?") {
		t.Fatalf("got %s", got)
	}
}

func TestConsoleURLUsesConsoleHost(t *testing.T) {
	t.Setenv(enum.EnvNameConsoleHost, "https://console.seeed.cc")
	t.Setenv(enum.EnvNameAppSite, "")
	got := ModelDetailConsoleURL("Global", "abc")
	if got != "https://console.seeed.cc/ai/model/detail?id=abc&tab=custom" {
		t.Fatalf("got %s", got)
	}
	if edit := AppEditConsoleURL("Global", "app-1"); edit != "https://codecraft.seeed.cc/hub/application/edit?id=app-1" {
		t.Fatalf("app url must stay on the hub, got %s", edit)
	}
	t.Setenv(enum.EnvNameConsoleHost, "https://console.example")
	if model := ModelDetailConsoleURL("Global", "abc"); model != "https://sensecraft.seeed.cc/ai/model/detail?id=abc&tab=custom" {
		t.Fatalf("foreign console host must stay on the public site, got %s", model)
	}
}
