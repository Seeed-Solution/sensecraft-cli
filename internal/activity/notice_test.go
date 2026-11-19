package activity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDueFetchesAndThenStaysQuiet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SENSECRAFT_CLI_PATH", dir)
	t.Setenv("CI", "")
	t.Setenv("BUILD_NUMBER", "")
	t.Setenv("RUN_ID", "")
	t.Setenv(envNoNotify, "")

	var calls int
	stubFetch(t, func(context.Context) ([]Item, error) {
		calls++
		return []Item{{
			ID:        "1",
			Name:      "测试活动",
			GuideURL:  "https://example.com/activity/guide",
			StartTime: "1792339200",
			EndTime:   "1795190399",
		}}, nil
	})

	now := time.Now()
	got := Due(now)
	if len(got) != 1 || got[0].Name != "测试活动" {
		t.Fatalf("due=%+v", got)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	text := Format(got)
	want := "Notice: SenseCraft AI is hosting the 测试活动 activity from October 19 to November 20. You can take part by creating and publishing apps, or draw inspiration from other developers. Visit https://example.com/activity/guide to learn more and join in."
	if text != want {
		t.Fatalf("format=%q", text)
	}
	if err := MarkShown(now); err != nil {
		t.Fatal(err)
	}
	if again := Due(now.Add(time.Hour)); again != nil {
		t.Fatalf("second due=%+v", again)
	}
	if calls != 1 {
		t.Fatalf("calls after quiet window=%d", calls)
	}
	next := Due(now.Add(25 * time.Hour))
	if len(next) != 1 {
		t.Fatalf("next day=%+v", next)
	}
	if calls != 2 {
		t.Fatalf("calls next day=%d", calls)
	}
}

func TestDueSkipsDismissedAndEmptyList(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SENSECRAFT_CLI_PATH", dir)
	t.Setenv("CI", "")
	t.Setenv("BUILD_NUMBER", "")
	t.Setenv("RUN_ID", "")
	t.Setenv(envNoNotify, "")

	stubFetch(t, func(context.Context) ([]Item, error) {
		return []Item{
			{ID: "1", Name: "已拒绝", GuideURL: "https://example.com/a"},
			{ID: "2", Name: "新活动", DescriptionURL: "https://example.com/b"},
		}, nil
	})

	now := time.Now()
	first := Due(now)
	if err := Acknowledge(IDs(first[:1])); err != nil {
		t.Fatal(err)
	}
	// ShownAt is still empty, so the cached list is filtered on the next read.
	got := Due(now.Add(time.Minute))
	if len(got) != 1 || got[0].ID != "2" || got[0].Link() != "https://example.com/b" {
		t.Fatalf("got=%+v", got)
	}
}

func TestDueFetchErrorStaysQuiet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SENSECRAFT_CLI_PATH", dir)
	t.Setenv("CI", "")
	t.Setenv("BUILD_NUMBER", "")
	t.Setenv("RUN_ID", "")
	t.Setenv(envNoNotify, "")
	var calls int
	stubFetch(t, func(context.Context) ([]Item, error) {
		calls++
		return nil, context.DeadlineExceeded
	})
	now := time.Now()
	if got := Due(now); got != nil {
		t.Fatalf("got=%+v", got)
	}
	if got := Due(now.Add(time.Hour)); got != nil {
		t.Fatalf("retry got=%+v", got)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestShouldSkip(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("BUILD_NUMBER", "")
	t.Setenv("RUN_ID", "")
	t.Setenv(envNoNotify, "")
	if ShouldSkip() {
		t.Fatal("should remind")
	}
	t.Setenv(envNoNotify, "1")
	if !ShouldSkip() {
		t.Fatal("opt-out should skip")
	}
	t.Setenv(envNoNotify, "")
	t.Setenv("CI", "true")
	if !ShouldSkip() {
		t.Fatal("CI should skip")
	}
}

func TestParseListStringCode(t *testing.T) {
	items, err := parseList([]byte(`{"code":"0","data":{"list":[{"activity_id":"1","name":"测试活动","guide_url":"https://example.com/activity/guide","description_url":"https://example.com/activity/description","tag_ids":["1"],"device_ids":[],"start_time":"1792339200","end_time":"1795190399"}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].GuideURL == "" || items[0].StartTime != "1792339200" || items[0].EndTime != "1795190399" {
		t.Fatalf("items=%+v", items)
	}
	if _, err := parseList([]byte(`{"code":"11201","msg":"no"}`)); err == nil {
		t.Fatal("expected business error")
	}
}

func TestEndpointUsesOverrideAndDefault(t *testing.T) {
	t.Setenv("SENSECRAFT_AI_HOST", "")
	got, err := Endpoint()
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://api-gateway.seeed.cc/openapi/activity/list" {
		t.Fatalf("default=%s", got)
	}
	t.Setenv("SENSECRAFT_AI_HOST", "https://test-api-gateway.seeed.cc/openapi/")
	got, err = Endpoint()
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://test-api-gateway.seeed.cc/openapi/activity/list" {
		t.Fatalf("override=%s", got)
	}
	t.Setenv("SENSECRAFT_AI_HOST", "http://127.0.0.1:9")
	if _, err := Endpoint(); err == nil {
		t.Fatal("expected unofficial host rejection")
	}
}

func TestFetchUsesEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openapi/activity/list" {
			t.Errorf("path=%s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"code":"0","data":{"list":[{"activity_id":"9","name":"线上"}]}}`))
	}))
	defer srv.Close()
	orig := activityEndpoint
	activityEndpoint = func() (string, error) {
		return srv.URL + "/openapi/activity/list", nil
	}
	t.Cleanup(func() { activityEndpoint = orig })
	items, err := fetchFromNetwork(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "9" {
		t.Fatalf("items=%+v", items)
	}
}

func stubFetch(t *testing.T, fn func(context.Context) ([]Item, error)) {
	t.Helper()
	orig := fetchActivities
	fetchActivities = fn
	t.Cleanup(func() { fetchActivities = orig })
}
