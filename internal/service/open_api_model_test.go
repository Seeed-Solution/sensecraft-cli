package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildModelListQueryUsesSearchLikeConsole(t *testing.T) {
	q := buildModelListQuery(ModelListParams{Name: "face", Task: "1", Page: 1, Length: 10})
	if got := q.Get("search"); got != "face" {
		t.Fatalf("want search=face (Console cardList.vue), got %q; raw=%v", got, q)
	}
	if q.Get("model_name") != "" || q.Get("name") != "" {
		t.Fatalf("must not send model_name/name; got %v", q)
	}
	if q.Get("task") != "1" || q.Get("page") != "1" || q.Get("length") != "10" {
		t.Fatalf("unexpected paging/task: %v", q)
	}
}

func TestBuildModelListQuerySendsDeviceFilters(t *testing.T) {
	q := buildModelListQuery(ModelListParams{
		Name:        "face",
		Task:        "1",
		UniformType: "40",
		Adapteds:    "recamera-sku",
		Page:        1,
		Length:      10,
	})
	if q.Get("uniform_type") != "40" {
		t.Fatalf("want uniform_type=40, got %v", q)
	}
	if q.Get("adapteds") != "recamera-sku" {
		t.Fatalf("want adapteds=recamera-sku, got %v", q)
	}
	if q.Get("search") != "face" {
		t.Fatalf("search must still be set, got %v", q)
	}
}

func TestBuildModelListQueryOmitsEmptyDeviceFilters(t *testing.T) {
	q := buildModelListQuery(ModelListParams{Page: 1, Length: 20})
	if q.Get("uniform_type") != "" || q.Get("adapteds") != "" {
		t.Fatalf("empty device filters must be omitted, got %v", q)
	}
}

func TestSandboxModelListFiltersByName(t *testing.T) {
	raw := sandboxModelList(ModelListParams{Public: true, Name: "person"})
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	list, _ := m["list"].([]any)
	if len(list) != 1 {
		t.Fatalf("want 1 match for person, got %d", len(list))
	}
	raw = sandboxModelList(ModelListParams{Public: true, Name: "zzzz-no-match"})
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	list, _ = m["list"].([]any)
	if len(list) != 0 {
		t.Fatalf("want 0 matches, got %d", len(list))
	}
}

func TestExtractFileURLFromView(t *testing.T) {
	raw := json.RawMessage(`{"id":"1","file_url":"https://cdn.example/a/model.rknn"}`)
	u, err := extractFileURLFromView(raw)
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://cdn.example/a/model.rknn" {
		t.Fatalf("got %q", u)
	}
}

func TestExtractURLFromApplyModelObjectAndString(t *testing.T) {
	objectSnap := json.RawMessage(`{
		"model_snapshot": {
			"arguments": {"url": "https://cdn.example/x.bin?timestamp=1"}
		}
	}`)
	u, err := extractURLFromApplyModel(objectSnap)
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://cdn.example/x.bin" {
		t.Fatalf("object snap got %q", u)
	}

	stringSnap := json.RawMessage(`{
		"model_snapshot": "{\"arguments\":{\"url\":\"https://cdn.example/y.bin\"}}"
	}`)
	u, err = extractURLFromApplyModel(stringSnap)
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://cdn.example/y.bin" {
		t.Fatalf("string snap got %q", u)
	}
}

func TestFilenameFromURLAndOutPath(t *testing.T) {
	if got := filenameFromURL("https://cdn.example/path/foo%20bar.rknn", "x"); got != "foo bar.rknn" {
		t.Fatalf("filename got %q", got)
	}
	dir := t.TempDir()
	p, err := resolveDownloadOutPath(dir, "model.bin")
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join(dir, "model.bin") {
		t.Fatalf("dir out got %q", p)
	}
	fileOut := filepath.Join(dir, "custom.bin")
	p, err = resolveDownloadOutPath(fileOut, "model.bin")
	if err != nil {
		t.Fatal(err)
	}
	if p != fileOut {
		t.Fatalf("file out got %q", p)
	}
}

func TestRedactPublicModelArtifactURLs(t *testing.T) {
	publicView := json.RawMessage(`{"id":"1","be_public":1,"file_url":"https://cdn.example/secret.bin","pic_url":"https://cdn.example/cover.png"}`)
	got := RedactPublicModelArtifactURLs(publicView, false)
	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["file_url"]; ok {
		t.Fatal("public view must omit file_url")
	}
	if m["pic_url"] != "https://cdn.example/cover.png" {
		t.Fatalf("pic_url should remain, got %v", m["pic_url"])
	}

	privateView := json.RawMessage(`{"id":"2","be_public":0,"file_url":"https://cdn.example/mine.bin"}`)
	got = RedactPublicModelArtifactURLs(privateView, false)
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatal(err)
	}
	if m["file_url"] != "https://cdn.example/mine.bin" {
		t.Fatalf("private view must keep file_url, got %v", m["file_url"])
	}

	publicList := json.RawMessage(`{"list":[{"id":"3","file_url":"https://cdn.example/lib.bin"}]}`)
	got = RedactPublicModelArtifactURLs(publicList, true)
	var wrap struct {
		List []map[string]any `json:"list"`
	}
	if err := json.Unmarshal(got, &wrap); err != nil {
		t.Fatal(err)
	}
	if len(wrap.List) != 1 {
		t.Fatalf("list len %d", len(wrap.List))
	}
	if _, ok := wrap.List[0]["file_url"]; ok {
		t.Fatal("public list must omit file_url even when be_public is missing")
	}
}

func TestSandboxDownloadRejectsPublicModel(t *testing.T) {
	s := ModelService{sandbox: true}
	_, err := s.DownloadModel(ModelDownloadParams{ModelID: "public-1"})
	if !errors.Is(err, ErrPublicModelNoDownload) {
		t.Fatalf("expected ErrPublicModelNoDownload, got %v", err)
	}
	_, err = s.DownloadModel(ModelDownloadParams{ModelID: "public-1", Apply: true, URLOnly: true})
	if !errors.Is(err, ErrPublicModelNoDownload) {
		t.Fatalf("apply/url-only must also refuse public models, got %v", err)
	}
}

func TestSandboxDownloadPrivateOmitsURLUnlessRequested(t *testing.T) {
	s := ModelService{sandbox: true}
	dir := t.TempDir()
	out := filepath.Join(dir, "out.bin")
	res, err := s.DownloadModel(ModelDownloadParams{ModelID: "mine-1", Out: out})
	if err != nil {
		t.Fatal(err)
	}
	if res.URL != "" {
		t.Fatalf("file download must omit url, got %q", res.URL)
	}
	if res.Path != out || res.Bytes == 0 {
		t.Fatalf("unexpected result: %+v", res)
	}

	res, err = s.DownloadModel(ModelDownloadParams{ModelID: "mine-1", URLOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.URL == "" {
		t.Fatal("url-only private download must include url")
	}
	if res.Path != "" {
		t.Fatalf("url-only must not write a file, path=%q", res.Path)
	}
}

func TestSandboxModelDownloadWritesFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.bin")
	res, err := sandboxModelDownload(ModelDownloadParams{
		ModelID: "m1",
		Out:     out,
	}, "file_url")
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != out || res.Bytes == 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "sandbox-model-bytes" {
		t.Fatalf("file content %q", data)
	}
}
