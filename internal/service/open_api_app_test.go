package service

import (
	"encoding/json"
	"testing"
)

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
	raw := sandboxAppList(AppListParams{Public: true, UniformType: "40"})
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
