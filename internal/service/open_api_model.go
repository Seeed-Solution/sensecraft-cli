package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/enum"
	"sensecraft-cloud-terminal/internal/utils"
)

type ModelService struct {
	client     *client.AIApiClient
	sandbox    bool
	remoteType string
	userID     string
}

func GetModelService(ctx app.AICmdContext) ModelService {
	return ModelService{
		client:     newAIHTTPClient(ctx),
		sandbox:    ctx.SandBox,
		remoteType: ctx.RemoteType(),
	}
}

type ModelListParams struct {
	Public      bool
	Mine        bool
	Name        string
	Task        string
	UniformType string
	Adapteds    string
	Page        int
	Length      int
}

func (s ModelService) ListModels(p ModelListParams) (json.RawMessage, error) {
	if s.sandbox {
		return sandboxModelList(p), nil
	}
	q := buildModelListQuery(p)

	path := enum.AIApiPathModelList
	if p.Mine && !p.Public {
		path = enum.AIApiPathModelMine
	}
	if p.Public {
		path = enum.AIApiPathModelList
	}

	data, err := s.client.Get(path, q)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			return sandboxModelList(p), nil
		}
		return nil, err
	}
	return data, nil
}

// buildModelListQuery mirrors Console cardList.vue getModellist params.
// Name search must use query key "search" (not model_name / name).
func buildModelListQuery(p ModelListParams) url.Values {
	q := url.Values{}
	page := p.Page
	if page <= 0 {
		page = 1
	}
	length := p.Length
	if length <= 0 {
		length = 20
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("length", strconv.Itoa(length))
	if name := strings.TrimSpace(p.Name); name != "" {
		q.Set("search", name)
	}
	if task := strings.TrimSpace(p.Task); task != "" {
		// Console sends task as an array when a sidebar task is selected;
		// a single id as a plain query value is accepted by list_model.
		q.Set("task", task)
	}
	if ut := strings.TrimSpace(p.UniformType); ut != "" {
		q.Set("uniform_type", ut)
	}
	if adapteds := strings.TrimSpace(p.Adapteds); adapteds != "" {
		// Console cardList.vue sends adapteds as an array of device ids;
		// a single id as a plain query value matches list_model.
		q.Set("adapteds", adapteds)
	}
	return q
}

type ModelViewParams struct {
	ModelID string
}

func (s ModelService) ViewModel(p ModelViewParams) (json.RawMessage, error) {
	if s.sandbox {
		return sandboxModelView(p), nil
	}
	q := url.Values{}
	q.Set("model_id", p.ModelID)
	data, err := s.client.Get(enum.AIApiPathModelView, q)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			return sandboxModelView(p), nil
		}
		return nil, err
	}
	return data, nil
}

// ErrPublicModelNoDownload is returned when a caller tries to fetch a public
// library artifact. Public models are deploy-only: the device pulls firmware
// at apply time. Do not expose file_url / apply_model URLs.
var ErrPublicModelNoDownload = errors.New("public models do not expose a download URL; deploy to a device so the artifact is pulled as firmware at apply time (cloud deploy or Console workspace)")

// ErrPublicModelNoClone is returned when create JSON points at a public-library
// model. The platform has no model clone; others' public models are deploy-only.
var ErrPublicModelNoClone = errors.New("public models cannot be cloned; there is no model clone on this platform — deploy instead (cloud deploy or Console workspace)")

// ErrForeignPublicModelNoEdit is returned when writing to someone else's
// public-library model. Owners may still edit their own published models
// (Console re-submits them for review).
var ErrForeignPublicModelNoEdit = errors.New("cannot edit another user's public library model; there is no clone — deploy instead, or update your own My Models id")

// ErrPublishedModelNoDelete matches Console: delete is only offered when be_public==0.
var ErrPublishedModelNoDelete = errors.New("published or under-review models cannot be deleted; unpublish first")

// ModelReReviewHint is returned when an owner updates a published model.
// Console add.vue: "模型信息更新后，将重新进入审核流程。"
const ModelReReviewHint = "published model was updated; it will re-enter review (Console edit_model sends be_public=1)"

type ModelDownloadParams struct {
	ModelID string
	Out     string
	// Apply uses GET /model/apply_model → model_snapshot.arguments.url (deploy path).
	// Default false uses view_model.file_url (private owner library download).
	Apply   bool
	URLOnly bool
}

type ModelDownloadResult struct {
	ModelID  string `json:"model_id"`
	Source   string `json:"source"`
	URL      string `json:"url,omitempty"`
	Path     string `json:"path,omitempty"`
	Bytes    int64  `json:"bytes,omitempty"`
	Filename string `json:"filename,omitempty"`
}

func (s ModelService) DownloadModel(p ModelDownloadParams) (ModelDownloadResult, error) {
	if strings.TrimSpace(p.ModelID) == "" {
		return ModelDownloadResult{}, fmt.Errorf("--id is required")
	}

	view, err := s.ViewModel(ModelViewParams{ModelID: p.ModelID})
	if err != nil {
		return ModelDownloadResult{}, err
	}
	if modelViewIsPublic(view) {
		return ModelDownloadResult{}, ErrPublicModelNoDownload
	}

	source := "file_url"
	if p.Apply {
		source = "apply_model"
	}
	if s.sandbox {
		result, err := sandboxModelDownload(p, source)
		if err != nil {
			return result, err
		}
		return omitDownloadURLUnlessRequested(result, p.URLOnly), nil
	}

	var downloadURL string
	if p.Apply {
		downloadURL, err = s.resolveApplyModelURL(p.ModelID)
	} else {
		downloadURL, err = extractFileURLFromView(view)
	}
	if err != nil {
		return ModelDownloadResult{}, err
	}
	if downloadURL == "" {
		return ModelDownloadResult{}, fmt.Errorf("no downloadable URL found via %s for model_id=%s", source, p.ModelID)
	}

	filename := filenameFromURL(downloadURL, "model.bin")
	result := ModelDownloadResult{
		ModelID:  p.ModelID,
		Source:   source,
		URL:      downloadURL,
		Filename: filename,
	}
	if p.URLOnly {
		return result, nil
	}

	outPath, err := resolveDownloadOutPath(p.Out, filename)
	if err != nil {
		return ModelDownloadResult{}, err
	}
	n, err := downloadURLToFile(downloadURL, outPath)
	if err != nil {
		return ModelDownloadResult{}, err
	}
	result.Path = outPath
	result.Bytes = n
	return omitDownloadURLUnlessRequested(result, false), nil
}

func omitDownloadURLUnlessRequested(result ModelDownloadResult, urlOnly bool) ModelDownloadResult {
	if !urlOnly {
		result.URL = ""
	}
	return result
}

// RedactPublicModelArtifactURLs strips file_url from public library payloads
// before stdout. Internal ViewModel (create/update merge, download check) stays
// unredacted. force=true redacts every object (public list); force=false uses
// each object's be_public.
func RedactPublicModelArtifactURLs(raw json.RawMessage, force bool) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	redactModelArtifactURLs(v, force)
	out, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return out
}

func redactModelArtifactURLs(v any, force bool) {
	switch t := v.(type) {
	case map[string]any:
		if force || modelMapIsPublic(t) {
			delete(t, "file_url")
		}
		for _, child := range t {
			redactModelArtifactURLs(child, force)
		}
	case []any:
		for _, item := range t {
			redactModelArtifactURLs(item, force)
		}
	}
}

func modelViewBePublic(data json.RawMessage) int {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return 0
	}
	n, ok := asInt(m["be_public"])
	if !ok {
		return 0
	}
	return n
}

func modelViewIsPublic(data json.RawMessage) bool {
	return modelViewBePublic(data) != 0
}

func modelViewAuthor(data json.RawMessage) string {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	return strictScalarString(m["author"])
}

func modelMapIsPublic(m map[string]any) bool {
	n, ok := asInt(m["be_public"])
	return ok && n != 0
}

func (s ModelService) resolveApplyModelURL(modelID string) (string, error) {
	q := url.Values{}
	q.Set("model_id", modelID)
	data, err := s.client.Get(enum.AIApiPathModelApply, q)
	if err != nil {
		return "", err
	}
	return extractURLFromApplyModel(data)
}

func extractFileURLFromView(data json.RawMessage) (string, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("parse view_model: %w", err)
	}
	if u, ok := stringField(m, "file_url"); ok {
		return u, nil
	}
	return "", fmt.Errorf("view_model has no file_url (owner download may be unavailable for this model)")
}

func extractURLFromApplyModel(data json.RawMessage) (string, error) {
	var wrap struct {
		ModelSnapshot json.RawMessage `json:"model_snapshot"`
	}
	if err := json.Unmarshal(data, &wrap); err != nil {
		return "", fmt.Errorf("parse apply_model: %w", err)
	}
	if len(wrap.ModelSnapshot) == 0 || string(wrap.ModelSnapshot) == "null" {
		return "", fmt.Errorf("apply_model returned empty model_snapshot")
	}

	// model_snapshot may be a JSON object or a JSON-encoded string.
	snapshot := wrap.ModelSnapshot
	if snapshot[0] == '"' {
		var asString string
		if err := json.Unmarshal(snapshot, &asString); err != nil {
			return "", fmt.Errorf("parse model_snapshot string: %w", err)
		}
		snapshot = json.RawMessage(asString)
	}

	var snap map[string]any
	if err := json.Unmarshal(snapshot, &snap); err != nil {
		return "", fmt.Errorf("parse model_snapshot: %w", err)
	}
	if args, ok := snap["arguments"].(map[string]any); ok {
		if u, ok := stringField(args, "url"); ok {
			return stripTimestampQuery(u), nil
		}
	}
	if u, ok := stringField(snap, "url"); ok {
		return stripTimestampQuery(u), nil
	}
	return "", fmt.Errorf("apply_model model_snapshot has no arguments.url")
}

func stringField(m map[string]any, key string) (string, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

func stripTimestampQuery(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return strings.Split(raw, "?timestamp=")[0]
	}
	q := u.Query()
	q.Del("timestamp")
	u.RawQuery = q.Encode()
	return u.String()
}

func filenameFromURL(raw, fallback string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return fallback
	}
	base := filepath.Base(u.Path)
	if base == "" || base == "." || base == "/" {
		return fallback
	}
	if decoded, err := url.PathUnescape(base); err == nil && decoded != "" {
		return decoded
	}
	return base
}

func resolveDownloadOutPath(out, filename string) (string, error) {
	out = strings.TrimSpace(out)
	if out == "" {
		return filepath.Clean(filename), nil
	}
	info, err := os.Stat(out)
	if err == nil && info.IsDir() {
		return filepath.Join(out, filename), nil
	}
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	// Treat as file path when it does not exist or is an existing file.
	if strings.HasSuffix(out, string(os.PathSeparator)) || strings.HasSuffix(out, "/") {
		if err := utils.EnsureDir(out); err != nil {
			return "", err
		}
		return filepath.Join(out, filename), nil
	}
	return filepath.Clean(out), nil
}

func downloadURLToFile(rawURL, outPath string) (int64, error) {
	if err := utils.EnsureDir(filepath.Dir(outPath)); err != nil {
		return 0, err
	}
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(rawURL)
	if err != nil {
		return 0, fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return 0, fmt.Errorf("download HTTP %d: %s", resp.StatusCode, string(body))
	}

	tmp, err := os.CreateTemp(filepath.Dir(outPath), filepath.Base(outPath)+".*.tmp")
	if err != nil {
		return 0, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	n, err := io.Copy(tmp, resp.Body)
	if err != nil {
		cleanup()
		return 0, fmt.Errorf("write download: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return 0, err
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Remove(outPath)
		if err2 := os.Rename(tmpPath, outPath); err2 != nil {
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("save file %s: %w", outPath, err2)
		}
	}
	return n, nil
}

func sandboxModelDownload(p ModelDownloadParams, source string) (ModelDownloadResult, error) {
	filename := "sandbox-model.bin"
	url := "https://example.invalid/sandbox/" + filename
	result := ModelDownloadResult{
		ModelID:  p.ModelID,
		Source:   source,
		URL:      url,
		Filename: filename,
	}
	if p.URLOnly {
		return result, nil
	}
	outPath, err := resolveDownloadOutPath(p.Out, filename)
	if err != nil {
		return ModelDownloadResult{}, err
	}
	payload := []byte("sandbox-model-bytes")
	if err := utils.WriteBytes(outPath, payload, 0o600); err != nil {
		return ModelDownloadResult{}, err
	}
	result.Path = outPath
	result.Bytes = int64(len(payload))
	return result, nil
}

func sandboxModelList(p ModelListParams) json.RawMessage {
	bePublic := 0
	if p.Public {
		bePublic = 1
	}
	item := map[string]any{
		"id":          "sandbox-model-1",
		"name":        "Sandbox Person Detection",
		"description": "Sandbox public model",
		"task":        p.Task,
		"be_public":   bePublic,
		"classes":     []string{"person"},
	}
	if ut := strings.TrimSpace(p.UniformType); ut != "" {
		item["uniform_types"] = []string{ut}
	}
	if bePublic != 0 {
		item["file_url"] = "https://example.invalid/sandbox/should-redact.tflite"
	}
	list := []map[string]any{item}
	if name := strings.TrimSpace(p.Name); name != "" {
		kw := strings.ToLower(name)
		filtered := make([]map[string]any, 0, 1)
		for _, it := range list {
			n, _ := it["name"].(string)
			if strings.Contains(strings.ToLower(n), kw) {
				filtered = append(filtered, it)
			}
		}
		list = filtered
	}
	raw, _ := json.Marshal(map[string]any{
		"list": list,
		"page": map[string]any{"count": len(list), "page": 1, "length": 20},
	})
	return raw
}

func sandboxModelView(p ModelViewParams) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"id":            p.ModelID,
		"name":          "Sandbox Person Detection",
		"description":   "Sandbox model detail",
		"content":       "<p>Sandbox model detail</p>",
		"pic_url":       "https://example.invalid/sandbox/model.png",
		"adapteds":      []string{"sandbox-vision"},
		"task":          "1",
		"model_format":  "2",
		"be_public":     sandboxModelBePublic(p.ModelID),
		"author":        sandboxModelAuthor(p.ModelID),
		"version":       "1.0.0",
		"file_url":      "https://example.invalid/sandbox/sandbox-model.tflite",
		"checksum":      "sandbox-md5",
		"precision":     "1",
		"scenario":      "1",
		"labels":        []map[string]any{{"object_id": 0, "object_name": "person"}},
		"classes":       []string{"person"},
		"uniform_types": sandboxModelUniformTypes(p.ModelID),
	})
	return raw
}

func sandboxModelUniformTypes(modelID string) []int {
	id := strings.ToLower(strings.TrimSpace(modelID))
	switch {
	case strings.Contains(id, "recamera") || strings.Contains(id, "type-40"):
		return []int{RecameraClassicUniformType}
	case strings.Contains(id, "xiao") || strings.Contains(id, "type-32"):
		return []int{32}
	default:
		return []int{36}
	}
}

// sandboxModelBePublic mirrors Console: 0 private, 1 public, 2 under review.
// Foreign public-library fixtures use a public- prefix (clone/download tests).
// Owner published fixtures use "published" in the id without that prefix.
func sandboxModelBePublic(modelID string) int {
	id := strings.ToLower(strings.TrimSpace(modelID))
	if sandboxIsForeignPublicID(id) {
		return 1
	}
	if strings.Contains(id, "review") {
		return 2
	}
	if strings.Contains(id, "published") {
		return 1
	}
	return 0
}

func sandboxIsForeignPublicID(modelID string) bool {
	id := strings.ToLower(strings.TrimSpace(modelID))
	return id == "public" || strings.HasPrefix(id, "public-")
}

func sandboxModelAuthor(modelID string) string {
	if sandboxIsForeignPublicID(modelID) {
		return "other-user"
	}
	return "sandbox-owner"
}
