package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/config"
)

func TestValidateAppInfoPatchEmptyName(t *testing.T) {
	err := validateAppInfoPatch(map[string]any{"app_name": ""})
	if err == nil || !strings.Contains(err.Error(), "app_name is required") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateAppInfoPatchExcerpt(t *testing.T) {
	if err := validateAppInfoPatch(map[string]any{"excerpt": ""}); err == nil {
		t.Fatal("expected empty excerpt rejection")
	}
	long := strings.Repeat("摘", 301)
	err := validateAppInfoPatch(map[string]any{"excerpt": long})
	if err == nil || !strings.Contains(err.Error(), "300") {
		t.Fatalf("got %v", err)
	}
	if err := validateAppInfoPatch(map[string]any{"excerpt": strings.Repeat("摘", 300)}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAppInfoPatchSceneAllowImages(t *testing.T) {
	if err := validateAppInfoPatch(map[string]any{"scene": ""}); err == nil {
		t.Fatal("expected empty scene rejection")
	}
	if err := validateAppInfoPatch(map[string]any{"scene": nil}); err == nil {
		t.Fatal("expected null scene rejection")
	}
	if err := validateAppInfoPatch(map[string]any{"allow_clone": nil}); err == nil {
		t.Fatal("expected null allow_clone rejection")
	}
	if err := validateAppInfoPatch(map[string]any{"allow_clone": -1}); err == nil {
		t.Fatal("expected allow_clone=-1 rejection")
	}
	if err := validateAppInfoPatch(map[string]any{"allow_clone": 2}); err == nil {
		t.Fatal("expected allow_clone=2 rejection")
	}
	if err := validateAppInfoPatch(map[string]any{"allow_commerce": 1}); err != nil {
		t.Fatal(err)
	}
	if err := validateAppInfoPatch(map[string]any{"images": nil}); err == nil {
		t.Fatal("expected null images rejection")
	}
	if err := validateAppInfoPatch(map[string]any{"images": []any{}}); err == nil {
		t.Fatal("expected empty images rejection")
	}
	if err := validateAppInfoPatch(map[string]any{"images": []any{"not-a-url"}}); err == nil {
		t.Fatal("expected invalid URL rejection")
	}
	if err := validateAppInfoPatch(map[string]any{
		"images": []any{"https://sensecraft-statics.seeed.cc/refer/pic/x.png"},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateAppRejectsEmptyName(t *testing.T) {
	s := sandboxAppSvc()
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	payload := `{"app_name":"","excerpt":"ok","scene":"1","allow_clone":1,"allow_commerce":0,"images":["https://example.invalid/a.png"]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := s.UpdateApp(AppUpdateParams{AppID: "sandbox-app-1", FromJSON: path, Yes: true})
	if err == nil || !strings.Contains(err.Error(), "app_name is required") {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateAppRejectsLongExcerpt(t *testing.T) {
	s := sandboxAppSvc()
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	body := map[string]any{
		"app_name":       "边界限制测试应用",
		"excerpt":        strings.Repeat("摘", 301),
		"scene":          "1",
		"allow_clone":    1,
		"allow_commerce": 0,
		"images":         []string{"https://example.invalid/a.png"},
	}
	raw, _ := json.Marshal(body)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := s.UpdateApp(AppUpdateParams{AppID: "sandbox-app-1", FromJSON: path, Yes: true})
	if err == nil || !strings.Contains(err.Error(), "excerpt max length") {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateAppRejectsInvalidScene(t *testing.T) {
	s := sandboxAppSvc()
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	payload := `{"scene":"INVALID_SCENE"}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := s.UpdateApp(AppUpdateParams{AppID: "sandbox-app-1", FromJSON: path, Yes: true})
	if err == nil || !strings.Contains(err.Error(), "app_scenarios_array") {
		t.Fatalf("got %v", err)
	}
}

func TestCreateAppRejectsBadExtraBeforeCreate(t *testing.T) {
	s := GetAppService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	payload := `{"app_name":"Ok Name","excerpt":"","scene":"1","allow_clone":1,"allow_commerce":0,"images":["https://example.invalid/a.png"]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateApp(AppCreateParams{FromJSON: path})
	if err == nil || !strings.Contains(err.Error(), "excerpt is required") {
		t.Fatalf("got %v", err)
	}
}
