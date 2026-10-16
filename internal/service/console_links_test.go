package service

import (
	"encoding/json"
	"strings"
	"testing"
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
	wantEdit := "https://sensecraft.seeed.cc/ai/application/edit?id=app-1"
	if edit != wantEdit {
		t.Fatalf("edit got %q want %q", edit, wantEdit)
	}
	detail := AppDetailConsoleURL("China", "app-1")
	wantDetail := "https://sensecraft.seeed.cc/ai/application/detail/app-1?from=my-apps&tab=custom"
	if detail != wantDetail {
		t.Fatalf("detail got %q want %q", detail, wantDetail)
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
	if edit != "https://sensecraft.seeed.cc/ai/application/edit?id=sandbox-created-app" {
		t.Fatalf("console_url=%q", edit)
	}
	if detail != "https://sensecraft.seeed.cc/ai/application/detail/sandbox-created-app?from=my-apps&tab=custom" {
		t.Fatalf("detail_url=%q", detail)
	}
}
