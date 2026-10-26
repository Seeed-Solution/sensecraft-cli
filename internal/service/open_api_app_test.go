package service

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/app"
)

var errActivityList = errors.New("activity list down")

func TestBuildAppListQuerySendsUniformType(t *testing.T) {
	q := buildAppListQuery(AppListParams{
		Public:      true,
		Name:        "vision",
		Scene:       "6",
		UniformType: "40",
		Page:        2,
		Length:      12,
	})
	if q.Get("be_public") != "1" {
		t.Fatalf("public plaza must send be_public=1, got %v", q)
	}
	if q.Get("app_name") != "vision" {
		t.Fatalf("want app_name=vision, got %v", q)
	}
	if q.Get("search") != "" || q.Get("name") != "" {
		t.Fatalf("must not send search/name; got %v", q)
	}
	if q.Get("uniform_type") != "40" {
		t.Fatalf("want uniform_type=40 (reCamera), got %v", q)
	}
	if q.Get("scene") != "6" || q.Get("page") != "2" || q.Get("length") != "12" {
		t.Fatalf("unexpected paging/scene: %v", q)
	}
}

func TestBuildAppListQueryOmitsEmptyUniformType(t *testing.T) {
	q := buildAppListQuery(AppListParams{Public: true})
	if q.Get("uniform_type") != "" {
		t.Fatalf("empty uniform_type must be omitted, got %v", q)
	}
	if q.Get("be_public") != "1" {
		t.Fatalf("public still sets be_public, got %v", q)
	}
}

func TestSandboxAppListIncludesUniformType(t *testing.T) {
	q := buildAppListQuery(AppListParams{Public: true, UniformType: "40"})
	raw := sandboxAppList(q)
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	list, _ := m["list"].([]any)
	if len(list) != 1 {
		t.Fatalf("want 1 sandbox app, got %d", len(list))
	}
	item, _ := list[0].(map[string]any)
	if item["uniform_type"] != "40" {
		t.Fatalf("sandbox item uniform_type=%v", item["uniform_type"])
	}
}

func TestPrepareAppListQueryPopularFillsFirstActivity(t *testing.T) {
	q, err := prepareAppListQuery(AppListParams{Public: true, Sort: "popular"}, sandboxOngoingActivities(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if q.Get("sort_by") != "popular" || q.Get("priority_activity_id") != "1" || q.Get("be_public") != "1" {
		t.Fatalf("query=%v", q)
	}
}

func TestPrepareAppListQueryPopularOmitsWhenNoActivity(t *testing.T) {
	q, err := prepareAppListQuery(AppListParams{Sort: "popular"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if q.Get("sort_by") != "popular" || q.Get("priority_activity_id") != "" {
		t.Fatalf("query=%v", q)
	}
}

func TestPrepareAppListQueryPopularOmitsWhenActivityListFails(t *testing.T) {
	q, err := prepareAppListQuery(AppListParams{Sort: "popular"}, nil, errActivityList)
	if err != nil {
		t.Fatal(err)
	}
	if q.Get("priority_activity_id") != "" {
		t.Fatalf("failed activity list must not pin an id: %v", q)
	}
}

func TestPrepareAppListQueryNewestAndLikesSkipPriority(t *testing.T) {
	for _, sort := range []string{"newest", "likes"} {
		q, err := prepareAppListQuery(AppListParams{Sort: sort}, sandboxOngoingActivities(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if q.Get("sort_by") != sort || q.Get("priority_activity_id") != "" {
			t.Fatalf("sort %s query=%v", sort, q)
		}
		if _, err := prepareAppListQuery(AppListParams{Sort: sort, PriorityActivityID: "1"}, nil, nil); err == nil {
			t.Fatalf("sort %s must reject an explicit priority activity", sort)
		}
	}
}

func TestPrepareAppListQueryTagAndMarks(t *testing.T) {
	q, err := prepareAppListQuery(AppListParams{Public: true, Sort: "popular", TagID: "1", PriorityActivityID: "4"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if q.Get("tag_id") != "1" || q.Get("priority_activity_id") != "4" {
		t.Fatalf("query=%v", q)
	}
	likes, err := prepareAppListQuery(AppListParams{Likes: true}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if likes.Get("mark_type") != "1" || likes.Get("be_public") != "1" {
		t.Fatalf("likes=%v", likes)
	}
	cols, err := prepareAppListQuery(AppListParams{Collections: true}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cols.Get("mark_type") != "0" || cols.Get("be_public") != "1" {
		t.Fatalf("collections=%v", cols)
	}
	if _, err := prepareAppListQuery(AppListParams{Likes: true, Mine: true}, nil, nil); err == nil {
		t.Fatal("likes must not combine with mine")
	}
	if _, err := prepareAppListQuery(AppListParams{MarkType: "like"}, nil, nil); err == nil {
		t.Fatal("like/star strings must be rejected")
	}
}

func TestListAppsSandboxShowsQueryIntent(t *testing.T) {
	s := GetAppService(app.AICmdContext{SandBox: true})
	raw, err := s.ListApps(AppListParams{Public: true, Sort: "popular", TagID: "1", Length: 12})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	query, _ := m["sandbox_query"].(map[string]any)
	priority, _ := query["priority_activity_id"].([]any)
	sortBy, _ := query["sort_by"].([]any)
	tag, _ := query["tag_id"].([]any)
	if len(priority) != 1 || priority[0] != "1" || len(sortBy) != 1 || sortBy[0] != "popular" || len(tag) != 1 || tag[0] != "1" {
		t.Fatalf("sandbox_query=%#v", query)
	}
}

func TestNormalizeOngoingActivityDropsIncompleteRows(t *testing.T) {
	raw := json.RawMessage(`{"list":[
		{"activity_id":"2","name":"Keep","tag_ids":["1","1","x",3]},
		{"activity_id":"0","name":"Zero"},
		{"name":"No id"},
		{"activity_id":"4"}
	]}`)
	items, err := normalizeActivityListPayload(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ActivityID != 2 || items[0].Name != "Keep" {
		t.Fatalf("items=%#v", items)
	}
	if strings.Join(items[0].TagIDs, ",") != "1,3" {
		t.Fatalf("tags=%v", items[0].TagIDs)
	}
}

func TestCloneNameAndRefuse(t *testing.T) {
	s := GetAppService(app.AICmdContext{SandBox: true})
	if _, err := s.CloneApp(AppCloneParams{AppID: "noclone-app", Yes: true}); err == nil || !strings.Contains(err.Error(), "allow_clone=0") {
		t.Fatalf("got %v", err)
	}
	raw, err := s.CloneApp(AppCloneParams{AppID: "sandbox-app-1", Yes: true})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["app_name"] != "Clone_Sandbox Vision App" {
		t.Fatalf("name=%v", m["app_name"])
	}
	if m["app_id"] != "sandbox-cloned-app" {
		t.Fatalf("id=%v", m["app_id"])
	}
	detail, _ := m["detail_url"].(string)
	if !strings.Contains(detail, "/view-application/sandbox-cloned-app-clone_sandbox-vision-app") || !strings.Contains(detail, "from=user-center") {
		t.Fatalf("detail=%s", detail)
	}
	if _, ok := m["be_public"]; ok {
		t.Fatalf("clone must not copy be_public: %#v", m)
	}
}

func TestWithdrawRequiresReview(t *testing.T) {
	s := GetAppService(app.AICmdContext{SandBox: true})
	if _, err := s.WithdrawApp(AppDeleteParams{AppID: "published-app", Yes: true}); err == nil || !strings.Contains(err.Error(), "unpublish") {
		t.Fatalf("public got %v", err)
	}
	if _, err := s.WithdrawApp(AppDeleteParams{AppID: "sandbox-app-1", Yes: true}); err == nil || !strings.Contains(err.Error(), "无需撤回") {
		t.Fatalf("private got %v", err)
	}
	raw, err := s.WithdrawApp(AppDeleteParams{AppID: "review-app", Yes: true})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["app_id"] != "review-app" || m["be_public"] != float64(0) {
		t.Fatalf("got %#v", m)
	}
}

func TestMarkAppFillsEmptyBody(t *testing.T) {
	s := GetAppService(app.AICmdContext{SandBox: true})
	raw, err := s.MarkApp("sandbox-app-1", 4, true)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["app_id"] != "sandbox-app-1" || m["op_type"] != float64(4) {
		t.Fatalf("got %#v", m)
	}
}

func TestPrefixRunes(t *testing.T) {
	if got := prefixRunes("abcdef", 4); got != "abcd" {
		t.Fatalf("got %q", got)
	}
	if got := prefixRunes("中文名称", 2); got != "中文" {
		t.Fatalf("got %q", got)
	}
}
