package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/enum"
)

func TestRequireYes(t *testing.T) {
	t.Setenv("SENSECRAFT_NO_CONFIRM_PROMPT", "1")
	if err := requireYes(false); !errors.Is(err, ErrConfirmRequired) {
		t.Fatalf("want ErrConfirmRequired, got %v", err)
	}
	if !strings.Contains(ErrConfirmRequired.Error(), "intent, not confirmation") {
		t.Fatalf("error should tell agents the first request is not confirmation: %v", ErrConfirmRequired)
	}
	if err := requireYes(true); err != nil {
		t.Fatal(err)
	}
}

func TestRequireAgreeDeclaration(t *testing.T) {
	if err := requireAgreeDeclaration(false); !errors.Is(err, ErrDeclarationRequired) {
		t.Fatalf("want ErrDeclarationRequired, got %v", err)
	}
	if err := requireAgreeDeclaration(true); err != nil {
		t.Fatal(err)
	}
}

func TestPublishRequiresAgreeDeclaration(t *testing.T) {
	s := sandboxAppSvc()
	_, err := s.SetAppPrivacy(AppPrivacyParams{AppID: "sandbox-app-1", Privacy: 2, Yes: true})
	if !errors.Is(err, ErrDeclarationRequired) {
		t.Fatalf("want ErrDeclarationRequired, got %v", err)
	}
	_, err = s.SetAppPrivacy(AppPrivacyParams{
		AppID: "sandbox-app-1", Privacy: 2, Yes: true, AgreeDeclaration: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// unpublish does not require declaration agreement
	_, err = s.SetAppPrivacy(AppPrivacyParams{AppID: "sandbox-app-1", Privacy: 0, Yes: true})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAppPrivacyRejectsOne(t *testing.T) {
	s := AppService{}
	_, err := s.SetAppPrivacy(AppPrivacyParams{AppID: "a1", Privacy: 1, Yes: true})
	if err == nil {
		t.Fatal("expected error for privacy=1")
	}
}

func TestFinalizeAppCreateResultKeepsIDWhenUpdateEmpty(t *testing.T) {
	created, _ := json.Marshal(map[string]any{"app_id": "11683", "app_name": "My App"})
	updated := json.RawMessage(`{}`)
	got := finalizeAppCreateResult(created, updated, "11683", "My App")
	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatal(err)
	}
	if m["app_id"] != "11683" {
		t.Fatalf("lost app_id: %#v", m)
	}
	if m["app_name"] != "My App" {
		t.Fatalf("lost app_name: %#v", m)
	}
}

func TestFinalizeAppCreateResultPrefersUpdateWithID(t *testing.T) {
	created, _ := json.Marshal(map[string]any{"app_id": "1"})
	updated, _ := json.Marshal(map[string]any{"app_id": "1", "ok": true})
	got := finalizeAppCreateResult(created, updated, "1", "n")
	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatal(err)
	}
	if m["ok"] != true {
		t.Fatalf("expected update body kept: %#v", m)
	}
}

func TestCreateAppBlankMarkdown(t *testing.T) {
	_, err := AppService{}.CreateApp(AppCreateParams{Markdown: "  \n", MarkdownSet: true})
	if err == nil || !strings.Contains(err.Error(), "markdown is empty") {
		t.Fatal(err)
	}
}

func TestCreateAppFromMarkdownSandbox(t *testing.T) {
	s := GetAppService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	raw, err := s.CreateApp(AppCreateParams{
		Markdown:    "# QA Name\n\n简介\n\nhttps://www.youtube.com/watch?v=dQw4w9WgXcQ\n\n![图](./a.png)\n",
		MarkdownSet: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["excerpt"] != "简介" {
		t.Fatalf("excerpt %#v", m["excerpt"])
	}
	blocks, _, err := deploymentBlockList(m)
	if err != nil || len(blocks) == 0 {
		t.Fatalf("blocks %v %#v", err, m["deployment"])
	}
	content := asString(blocks[0].(map[string]any)["content"])
	if !strings.Contains(content, "video-embed-wrapper") {
		t.Fatalf("create did not fill video html: %s", content)
	}
	pending, _ := m["pending_uploads"].([]any)
	if len(pending) != 1 {
		t.Fatalf("pending %#v", m["pending_uploads"])
	}
	if _, ok := m["warnings"]; !ok {
		t.Fatal("missing warnings")
	}
}

func TestCreateAppFromJSONReturnsIdentityInSandbox(t *testing.T) {
	s := GetAppService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	dir := t.TempDir()
	path := filepath.Join(dir, "app.json")
	payload := `{"app_name":"Sandbox Full","excerpt":"hi","scene":"5","allow_clone":1,"allow_commerce":0,"images":["https://example.invalid/a.jpg"]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := s.CreateApp(AppCreateParams{FromJSON: path})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["app_id"] != "sandbox-created-app" {
		t.Fatalf("app_id: %#v", m["app_id"])
	}
	if m["console_url"] == nil || m["detail_url"] == nil {
		t.Fatalf("missing console links: %#v", m)
	}
}

func TestMergeAppInfoFromExisting(t *testing.T) {
	body := map[string]any{"app_name": "Renamed"}
	existing := map[string]any{
		"app_name":       "Old",
		"excerpt":        "Keep me",
		"scene":          float64(1),
		"allow_clone":    float64(0),
		"allow_commerce": float64(1),
		"images": []any{
			map[string]any{"url": "https://example.com/a.jpg"},
		},
		"author":    "alice",
		"be_public": float64(1),
	}
	mergeAppInfoFromExisting(body, existing)
	if body["app_name"] != "Renamed" {
		t.Fatalf("name overwritten: %#v", body["app_name"])
	}
	if body["excerpt"] != "Keep me" {
		t.Fatalf("excerpt not merged: %#v", body["excerpt"])
	}
	if body["scene"] != "1" {
		t.Fatalf("scene not stringified: %#v", body["scene"])
	}
	imgs, ok := body["images"].([]string)
	if !ok || len(imgs) != 1 || imgs[0] != "https://example.com/a.jpg" {
		t.Fatalf("images not normalized: %#v", body["images"])
	}
	if _, ok := body["be_public"]; ok {
		t.Fatal("be_public must not be merged into update body")
	}
}

func TestSyncAppCoverFieldsFromImages(t *testing.T) {
	body := map[string]any{"images": []any{"https://cdn.example/cover.png", "https://cdn.example/2.png"}}
	syncAppCoverFields(body)
	if body["pic_url"] != "https://cdn.example/cover.png" {
		t.Fatalf("pic_url: %#v", body["pic_url"])
	}
}

func TestSyncAppCoverFieldsFromPicURL(t *testing.T) {
	body := map[string]any{"pic_url": "https://cdn.example/from-list.png"}
	syncAppCoverFields(body)
	imgs, ok := body["images"].([]string)
	if !ok || len(imgs) != 1 || imgs[0] != "https://cdn.example/from-list.png" {
		t.Fatalf("images: %#v", body["images"])
	}
}

func TestCreateAppFromPicURLFillsImages(t *testing.T) {
	s := GetAppService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	dir := t.TempDir()
	path := filepath.Join(dir, "app.json")
	payload := `{"app_name":"Cover Restore","pic_url":"https://example.invalid/cover.png","excerpt":"hi","scene":"5","allow_clone":0,"allow_commerce":0}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := s.CreateApp(AppCreateParams{FromJSON: path})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["pic_url"] != "https://example.invalid/cover.png" {
		t.Fatalf("pic_url: %#v", m)
	}
	imgs, ok := m["images"].([]any)
	if !ok || len(imgs) < 1 || imgs[0] != "https://example.invalid/cover.png" {
		t.Fatalf("cover not first in images: %#v", m["images"])
	}
}

func TestUpdateAppMergesInfoWhenOnlyNameProvided(t *testing.T) {
	s := GetAppService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	dir := t.TempDir()
	path := filepath.Join(dir, "patch.json")
	if err := os.WriteFile(path, []byte(`{"app_name":"Renamed App"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := s.UpdateApp(AppUpdateParams{
		AppID:    "sandbox-app-1",
		FromJSON: path,
		Yes:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result["app_name"] != "Renamed App" {
		t.Fatalf("app_name: %#v", result["app_name"])
	}
	if result["excerpt"] != "Sandbox application detail" {
		t.Fatalf("expected merged excerpt, got %#v", result["excerpt"])
	}
	if result["scene"] != "1" {
		t.Fatalf("expected scene string 1, got %#v", result["scene"])
	}
	imgs, ok := result["images"].([]any)
	if !ok || len(imgs) != 1 || imgs[0] != "https://example.invalid/sandbox-cover.jpg" {
		t.Fatalf("expected merged images, got %#v", result["images"])
	}
	if _, ok := result["be_public"]; ok {
		t.Fatal("be_public must not appear on info rename update")
	}
}

func TestUpdateAppMaterialsOnlyDoesNotForceInfoMerge(t *testing.T) {
	s := GetAppService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	dir := t.TempDir()
	path := filepath.Join(dir, "bom.json")
	payload := `{"materials":[{"name":"Custom","quantity":"1","note":"","buy_url":""}]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := s.UpdateApp(AppUpdateParams{
		AppID:    "sandbox-app-1",
		FromJSON: path,
		Yes:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if _, ok := result["app_name"]; ok {
		t.Fatalf("materials-only update should not inject app_name: %#v", result)
	}
	if _, ok := result["materials"]; !ok {
		t.Fatalf("materials missing: %#v", result)
	}
}

func TestUpdateAppRejectsQuantityAbove999(t *testing.T) {
	s := sandboxAppSvc()
	dir := t.TempDir()
	path := filepath.Join(dir, "bom.json")
	payload := `{"materials":[{"name":"Custom","quantity":"1000","note":"","buy_url":""}]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := s.UpdateApp(AppUpdateParams{
		AppID:    "sandbox-app-1",
		FromJSON: path,
		Yes:      true,
	})
	if err == nil || !strings.Contains(err.Error(), "1 to 999") {
		t.Fatalf("got %v", err)
	}
}

func TestAttachCoverImagePrepends(t *testing.T) {
	s := GetAppService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	_, err := s.AttachCoverImage("sandbox-app-1", "https://example.invalid/new-cover.png", false)
	if !errors.Is(err, ErrConfirmRequired) {
		t.Fatalf("want confirm required, got %v", err)
	}
	raw, err := s.AttachCoverImage("sandbox-app-1", "https://example.invalid/new-cover.png", true)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	imgs, ok := result["images"].([]any)
	if !ok || len(imgs) < 1 {
		t.Fatalf("images: %#v", result["images"])
	}
	if imgs[0] != "https://example.invalid/new-cover.png" {
		t.Fatalf("cover not prepended: %#v", imgs)
	}
}

func TestUpdateAppRejectsTooManyImages(t *testing.T) {
	s := sandboxAppSvc()
	dir := t.TempDir()
	path := filepath.Join(dir, "images.json")
	imgs := make([]string, MaxAppImageCount+1)
	for i := range imgs {
		imgs[i] = fmt.Sprintf("https://example.invalid/%d.png", i)
	}
	body, _ := json.Marshal(map[string]any{"images": imgs})
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := s.UpdateApp(AppUpdateParams{AppID: "sandbox-app-1", FromJSON: path, Yes: true})
	if err == nil || !strings.Contains(err.Error(), "maxImageCount") {
		t.Fatalf("want maxImageCount rejection, got %v", err)
	}
}

func sandboxAppSvc() AppService {
	return GetAppService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
}

func TestUpdateOwnPublishedAppSavesSketch(t *testing.T) {
	s := sandboxAppSvc()
	dir := t.TempDir()
	path := filepath.Join(dir, "patch.json")
	if err := os.WriteFile(path, []byte(`{"app_name":"Plaza Draft"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := s.UpdateApp(AppUpdateParams{
		AppID:    "sandbox-published-1",
		FromJSON: path,
		Yes:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result["sketch_saved"] != true {
		t.Fatalf("expected sketch_saved: %#v", result)
	}
	if result["be_public"] != float64(1) {
		t.Fatalf("expected be_public=1 on published info save: %#v", result["be_public"])
	}
}

func TestUpdateAppLockedUnderReview(t *testing.T) {
	s := sandboxAppSvc()
	dir := t.TempDir()
	path := filepath.Join(dir, "patch.json")
	if err := os.WriteFile(path, []byte(`{"app_name":"No"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := s.UpdateApp(AppUpdateParams{
		AppID:    "sandbox-review-1",
		FromJSON: path,
		Yes:      true,
	})
	if !errors.Is(err, ErrAppLockedUnderReview) {
		t.Fatalf("want locked, got %v", err)
	}
}

func TestDeletePublishedAppRefused(t *testing.T) {
	s := sandboxAppSvc()
	_, err := s.DeleteApp(AppDeleteParams{AppID: "sandbox-published-1", Yes: true})
	if !errors.Is(err, ErrPublishedAppNoDelete) {
		t.Fatalf("want no-delete, got %v", err)
	}
	_, err = s.DeleteApp(AppDeleteParams{AppID: "sandbox-app-1", Yes: true})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPublishAlreadyPublicIsDraftPublish(t *testing.T) {
	s := sandboxAppSvc()
	raw, err := s.SetAppPrivacy(AppPrivacyParams{
		AppID: "sandbox-published-1", Privacy: 2, Yes: true, AgreeDeclaration: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result["draft_publish"] != true {
		t.Fatalf("expected draft_publish: %#v", result)
	}
	if _, ok := result["console_url"]; ok {
		t.Fatalf("under review must not include an edit link: %#v", result["console_url"])
	}
}

func TestPublishDetailURLIsCreatorCenter(t *testing.T) {
	s := sandboxAppSvc()
	raw, err := s.SetAppPrivacy(AppPrivacyParams{
		AppID: "sandbox-app-1", Privacy: 2, Yes: true, AgreeDeclaration: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	detail, _ := result["detail_url"].(string)
	if !strings.Contains(detail, "/view-application/sandbox-app-1-sandbox-vision-app") ||
		strings.Contains(detail, "?") ||
		strings.Contains(detail, "/application/detail/") {
		t.Fatalf("detail_url=%q", detail)
	}
	if _, ok := result["console_url"]; ok {
		t.Fatalf("under review must not include an edit link: %#v", result["console_url"])
	}
}

func TestViewUnderReviewDropsEditURL(t *testing.T) {
	s := sandboxAppSvc()
	raw := s.withViewDetailURL(json.RawMessage(`{
		"app_id": "sandbox-review-1",
		"app_name": "Sandbox Vision App",
		"be_public": 2,
		"console_url": "https://codecraft.seeed.cc/hub/application/edit?id=sandbox-review-1"
	}`), false)
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if _, ok := result["console_url"]; ok {
		t.Fatalf("view of an app under review must drop console_url: %#v", result["console_url"])
	}
	detail, _ := result["detail_url"].(string)
	if !strings.Contains(detail, "/view-application/sandbox-review-1") {
		t.Fatalf("detail_url=%q", detail)
	}
	if result["view_kind"] != "published" || result["view_sketch"] != float64(0) {
		t.Fatalf("want published view markers, got view_kind=%#v view_sketch=%#v", result["view_kind"], result["view_sketch"])
	}
}

func TestViewAppMarksSketchAndPublished(t *testing.T) {
	s := sandboxAppSvc()
	sketchRaw, err := s.ViewApp(AppViewParams{AppID: "sandbox-app-1", ViewSketch: true})
	if err != nil {
		t.Fatal(err)
	}
	var sketch map[string]any
	if err := json.Unmarshal(sketchRaw, &sketch); err != nil {
		t.Fatal(err)
	}
	if sketch["view_kind"] != "sketch" || sketch["view_sketch"] != float64(1) {
		t.Fatalf("sketch view markers: %#v", sketch)
	}

	pubRaw, err := s.ViewApp(AppViewParams{AppID: "sandbox-app-1", ViewSketch: false})
	if err != nil {
		t.Fatal(err)
	}
	var published map[string]any
	if err := json.Unmarshal(pubRaw, &published); err != nil {
		t.Fatal(err)
	}
	if published["view_kind"] != "published" || published["view_sketch"] != float64(0) {
		t.Fatalf("published view markers: %#v", published)
	}
}

func TestListUnderReviewDropsEditURLOnly(t *testing.T) {
	s := sandboxAppSvc()
	raw := s.withListDetailURLs(json.RawMessage(`{
		"list": [
			{
				"app_id": "sandbox-review-1",
				"app_name": "Locked",
				"be_public": 2,
				"console_url": "https://codecraft.seeed.cc/hub/application/edit?id=sandbox-review-1"
			},
			{
				"app_id": "sandbox-app-1",
				"app_name": "Draft",
				"be_public": 0,
				"console_url": "https://codecraft.seeed.cc/hub/application/edit?id=sandbox-app-1"
			}
		]
	}`), false)
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	list, _ := result["list"].([]any)
	if len(list) != 2 {
		t.Fatalf("list=%#v", result["list"])
	}
	locked, _ := list[0].(map[string]any)
	draft, _ := list[1].(map[string]any)
	if _, ok := locked["console_url"]; ok {
		t.Fatalf("review row kept an edit link: %#v", locked["console_url"])
	}
	if _, ok := locked["detail_url"]; !ok {
		t.Fatal("review row lost detail_url")
	}
	edit, _ := draft["console_url"].(string)
	if !strings.Contains(edit, "/application/edit?id=sandbox-app-1") {
		t.Fatalf("private row edit link=%q", edit)
	}
}

func TestPublishLockedUnderReview(t *testing.T) {
	s := sandboxAppSvc()
	_, err := s.SetAppPrivacy(AppPrivacyParams{
		AppID: "sandbox-review-1", Privacy: 2, Yes: true, AgreeDeclaration: true,
	})
	if !errors.Is(err, ErrAppLockedUnderReview) {
		t.Fatalf("want locked, got %v", err)
	}
}

func TestSetBOMLockedUnderReview(t *testing.T) {
	s := sandboxAppSvc()
	dir := t.TempDir()
	path := filepath.Join(dir, "bom.json")
	payload := `{"materials":[{"name":"Custom","quantity":"1","note":"","buy_url":""}]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := s.SetBOM(AppBOMSetParams{AppID: "sandbox-review-1", FromJSON: path, Yes: true})
	if !errors.Is(err, ErrAppLockedUnderReview) {
		t.Fatalf("want locked, got %v", err)
	}
}

func TestRejectForeignAppWrite(t *testing.T) {
	s := sandboxAppSvc()
	dir := t.TempDir()
	path := filepath.Join(dir, "patch.json")
	if err := os.WriteFile(path, []byte(`{"app_name":"No"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := s.UpdateApp(AppUpdateParams{AppID: "public-1", FromJSON: path, Yes: true})
	if !errors.Is(err, ErrForeignAppNoEdit) {
		t.Fatalf("want foreign refuse, got %v", err)
	}
	_, err = s.DeleteApp(AppDeleteParams{AppID: "public-1", Yes: true})
	if !errors.Is(err, ErrForeignAppNoEdit) {
		t.Fatalf("want foreign refuse on delete, got %v", err)
	}
	_, err = s.SetAppPrivacy(AppPrivacyParams{
		AppID: "public-1", Privacy: 2, Yes: true, AgreeDeclaration: true,
	})
	if !errors.Is(err, ErrForeignAppNoEdit) {
		t.Fatalf("want foreign refuse on publish, got %v", err)
	}
}

func TestRejectUnownedAppWriteLiveAuthor(t *testing.T) {
	s := AppService{userID: "me"}
	if err := s.rejectUnownedAppWrite("a1", map[string]any{"user_id": "other"}); !errors.Is(err, ErrForeignAppNoEdit) {
		t.Fatalf("want foreign refuse, got %v", err)
	}
	if err := s.rejectUnownedAppWrite("a1", map[string]any{"user_id": "me"}); err != nil {
		t.Fatal(err)
	}
	if err := s.rejectUnownedAppWrite("a1", map[string]any{}); err != nil {
		t.Fatalf("missing user_id should not block: %v", err)
	}
}

func TestCreateAppKeepsNameWhenAPIOmitsIt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != enum.AIApiPathAppCreate {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"app_id": "11641"},
		})
	}))
	t.Cleanup(srv.Close)
	s := appServiceForTest(t, srv)
	raw, err := s.CreateApp(AppCreateParams{Name: "reTerminal Sticky"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["app_name"] != "reTerminal Sticky" {
		t.Fatalf("app_name: %#v", got["app_name"])
	}
	detail, _ := got["detail_url"].(string)
	if !strings.Contains(detail, "/view-application/11641-reterminal-sticky") {
		t.Fatalf("detail_url: %s", detail)
	}
}

func TestCreateAppWritesPayloadWhenSketchIsMissing(t *testing.T) {
	var updateBody map[string]any
	var updateCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case enum.AIApiPathAppCreate:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"app_id": "11567", "app_name": "Snake"},
			})
		case enum.AIApiPathAppView:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 11306,
				"msg":  "Not Found the resources",
			})
		case enum.AIApiPathAppUpdate:
			updateCalls++
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &updateBody)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{}})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	s := appServiceForTest(t, srv)

	dir := t.TempDir()
	path := filepath.Join(dir, "app.json")
	payload := `{"app_name":"Snake","excerpt":"a board game"}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := s.CreateApp(AppCreateParams{FromJSON: path})
	if err != nil {
		t.Fatal(err)
	}
	if updateCalls != 1 {
		t.Fatalf("update calls=%d", updateCalls)
	}
	if updateBody["app_id"] != "11567" || updateBody["excerpt"] != "a board game" {
		t.Fatalf("update body: %#v", updateBody)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["app_id"] != "11567" {
		t.Fatalf("app_id: %#v", got)
	}
}

func TestUpdateAppStillFailsWhenSnapshotMissing(t *testing.T) {
	var updateCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case enum.AIApiPathAppView:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 11306,
				"msg":  "Not Found the resources",
			})
		case enum.AIApiPathAppUpdate:
			updateCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	s := appServiceForTest(t, srv)
	dir := t.TempDir()
	path := filepath.Join(dir, "patch.json")
	if err := os.WriteFile(path, []byte(`{"excerpt":"keep"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := s.UpdateApp(AppUpdateParams{AppID: "11567", FromJSON: path, Yes: true})
	if err == nil || !strings.Contains(err.Error(), "11306") {
		t.Fatalf("want 11306, got %v", err)
	}
	if updateCalls != 0 {
		t.Fatalf("update must not run when the snapshot is missing, calls=%d", updateCalls)
	}
}

func TestUpdateAppSendsNumericActivityIDAndDropsTags(t *testing.T) {
	var rawBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case enum.AIApiPathAppView:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"app_id": "11619", "be_public": 0, "excerpt": "keep", "tags": []any{"old"}},
			})
		case enum.AIApiPathAppUpdate:
			rawBody, _ = io.ReadAll(r.Body)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	s := appServiceForTest(t, srv)
	path := filepath.Join(t.TempDir(), "patch.json")
	payload := `{"excerpt":"new","activity_id":"1","tags":["ignored"]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := s.UpdateApp(AppUpdateParams{AppID: "11619", FromJSON: path, Yes: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rawBody), `"activity_id":1`) {
		t.Fatalf("activity_id must be a JSON number, body=%s", rawBody)
	}
	var body map[string]any
	if err := json.Unmarshal(rawBody, &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["tags"]; ok {
		t.Fatalf("tags must not be sent: %#v", body)
	}
	if body["excerpt"] != "new" {
		t.Fatalf("excerpt: %#v", body["excerpt"])
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out["tags_ignored"] != true {
		t.Fatalf("stdout should say tags were ignored: %#v", out)
	}
}

func TestUpdateAppOmitsBlankActivityID(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case enum.AIApiPathAppView:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"app_id": "11619", "be_public": 0, "excerpt": "keep"},
			})
		case enum.AIApiPathAppUpdate:
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	s := appServiceForTest(t, srv)
	path := filepath.Join(t.TempDir(), "patch.json")
	if err := os.WriteFile(path, []byte(`{"excerpt":"new","activity_id":""}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateApp(AppUpdateParams{AppID: "11619", FromJSON: path, Yes: true}); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["activity_id"]; ok {
		t.Fatalf("blank activity_id must be omitted: %#v", body)
	}
}

func TestUpdateAppRejectsBadActivityIDBeforePost(t *testing.T) {
	var updateCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case enum.AIApiPathAppView:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"app_id": "11619", "be_public": 0},
			})
		case enum.AIApiPathAppUpdate:
			updateCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	s := appServiceForTest(t, srv)
	for _, payload := range []string{
		`{"excerpt":"new","activity_id":0}`,
		`{"excerpt":"new","activity_id":-1}`,
		`{"excerpt":"new","activity_id":1.5}`,
		`{"excerpt":"new","activity_id":"abc"}`,
	} {
		path := filepath.Join(t.TempDir(), "patch.json")
		if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := s.UpdateApp(AppUpdateParams{AppID: "11619", FromJSON: path, Yes: true})
		if err == nil || !strings.Contains(err.Error(), "positive integer") {
			t.Fatalf("payload %s: got %v", payload, err)
		}
	}
	if updateCalls != 0 {
		t.Fatalf("invalid activity_id must not POST, calls=%d", updateCalls)
	}
}

func TestUpdateAppActivityFlagOnly(t *testing.T) {
	var rawBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case enum.AIApiPathAppView:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"app_id": "11619", "be_public": 0, "excerpt": "keep", "allow_clone": 1},
			})
		case enum.AIApiPathActivityList:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"list": []any{map[string]any{"activity_id": "12", "name": "Open"}}},
			})
		case enum.AIApiPathAppUpdate:
			rawBody, _ = io.ReadAll(r.Body)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	s := appServiceForTest(t, srv)
	if _, err := s.UpdateApp(AppUpdateParams{AppID: "11619", Yes: true, ActivitySet: true, ActivityID: "12"}); err != nil {
		t.Fatal(err)
	}
	if string(rawBody) != `{"activity_id":12,"app_id":"11619"}` && !strings.Contains(string(rawBody), `"activity_id":12`) {
		t.Fatalf("body=%s", rawBody)
	}
	var body map[string]any
	if err := json.Unmarshal(rawBody, &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["excerpt"]; ok {
		t.Fatalf("activity-only update must not merge info fields: %#v", body)
	}
}

func TestUpdateAppSurfacesExpiredActivity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case enum.AIApiPathAppView:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"app_id": "11619", "be_public": 0},
			})
		case enum.AIApiPathAppUpdate:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 25000,
				"msg":  "Activity does not exist or has expired",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	s := appServiceForTest(t, srv)
	_, err := s.UpdateApp(AppUpdateParams{AppID: "11619", Yes: true, ActivitySet: true, ActivityID: "9"})
	if err == nil || !strings.Contains(err.Error(), "25000") || !strings.Contains(err.Error(), "Activity does not exist or has expired") {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateAppContestSetsAllowCloneAndRejectsMissingActivity(t *testing.T) {
	var updateCalls int
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case enum.AIApiPathAppView:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"app_id": "11619", "be_public": 0, "excerpt": "keep",
					"allow_clone": 0, "app_name": "Demo", "scene": "5",
					"allow_commerce": 0, "images": []any{"https://example.invalid/a.png"},
					"license": "", "author": "me",
				},
			})
		case enum.AIApiPathActivityList:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"list": []any{map[string]any{"activity_id": "3", "name": "Open", "tag_ids": []any{"1"}}}},
			})
		case enum.AIApiPathAppUpdate:
			updateCalls++
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	s := appServiceForTest(t, srv)
	if _, err := s.UpdateApp(AppUpdateParams{AppID: "11619", Yes: true, ActivitySet: true, ActivityID: "9"}); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("missing activity got %v", err)
	}
	if updateCalls != 0 {
		t.Fatalf("inactive activity must not POST, calls=%d", updateCalls)
	}
	if _, err := s.UpdateApp(AppUpdateParams{AppID: "11619", Yes: true, ActivitySet: true, ActivityID: "3"}); err != nil {
		t.Fatal(err)
	}
	if body["allow_clone"] != float64(1) || body["activity_id"] != float64(3) {
		t.Fatalf("body=%#v", body)
	}
}

func TestUpdateAppCancelActivityFlagOnly(t *testing.T) {
	var rawBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case enum.AIApiPathAppView:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"app_id": "11619", "be_public": 0, "excerpt": "keep",
					"allow_clone": 0, "app_name": "Demo",
				},
			})
		case enum.AIApiPathActivityList:
			t.Errorf("cancel must not consult the ongoing activity list")
			http.NotFound(w, r)
		case enum.AIApiPathAppUpdate:
			rawBody, _ = io.ReadAll(r.Body)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	s := appServiceForTest(t, srv)
	if _, err := s.UpdateApp(AppUpdateParams{AppID: "11619", Yes: true, CancelActivitySet: true, CancelActivityID: "1"}); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(rawBody, &body); err != nil {
		t.Fatal(err)
	}
	if body["cancel_activity_id"] != float64(1) || body["app_id"] != "11619" {
		t.Fatalf("body=%#v", body)
	}
	if _, ok := body["allow_clone"]; ok {
		t.Fatalf("cancel must not force allow_clone: %#v", body)
	}
	if _, ok := body["excerpt"]; ok {
		t.Fatalf("cancel-only update must not merge info fields: %#v", body)
	}
}

func TestUpdateAppCancelActivityFromJSON(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case enum.AIApiPathAppView:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"app_id": "11619", "be_public": 0, "excerpt": "keep"},
			})
		case enum.AIApiPathAppUpdate:
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	s := appServiceForTest(t, srv)
	path := filepath.Join(t.TempDir(), "patch.json")
	if err := os.WriteFile(path, []byte(`{"excerpt":"new","cancel_activity_id":"1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateApp(AppUpdateParams{AppID: "11619", FromJSON: path, Yes: true}); err != nil {
		t.Fatal(err)
	}
	if body["cancel_activity_id"] != float64(1) {
		t.Fatalf("cancel_activity_id must be a JSON number: %#v", body)
	}
}

func TestUpdateAppRejectsBadCancelActivityID(t *testing.T) {
	var updateCalls int
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case enum.AIApiPathAppView:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"app_id": "11619", "be_public": 0, "excerpt": "keep"},
			})
		case enum.AIApiPathAppUpdate:
			updateCalls++
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	s := appServiceForTest(t, srv)
	for _, payload := range []string{
		`{"excerpt":"new","cancel_activity_id":0}`,
		`{"excerpt":"new","cancel_activity_id":-1}`,
		`{"excerpt":"new","cancel_activity_id":"abc"}`,
		`{"excerpt":"new","activity_id":1,"cancel_activity_id":1}`,
	} {
		path := filepath.Join(t.TempDir(), "patch.json")
		if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.UpdateApp(AppUpdateParams{AppID: "11619", FromJSON: path, Yes: true}); err == nil {
			t.Fatalf("payload %s: expected error", payload)
		}
	}
	blank := filepath.Join(t.TempDir(), "blank.json")
	if err := os.WriteFile(blank, []byte(`{"excerpt":"new","cancel_activity_id":""}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateApp(AppUpdateParams{AppID: "11619", FromJSON: blank, Yes: true}); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["cancel_activity_id"]; ok {
		t.Fatalf("blank cancel_activity_id must be omitted: %#v", body)
	}
	if updateCalls != 1 {
		t.Fatalf("only the blank cancel id may POST, calls=%d", updateCalls)
	}
}

func appServiceForTest(t *testing.T, srv *httptest.Server) AppService {
	t.Helper()
	return AppService{
		client:     client.NewAIApiClientAt(srv.URL, srv.Client()),
		remoteType: "Global",
	}
}
