package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/enum"
)

type ModelPublishParams struct {
	ModelID string
	// Status: 1 = publish to public library, 0 = privatize. Do not send other values.
	Status int
	Yes    bool
}

func (s ModelService) PublishModel(p ModelPublishParams) (json.RawMessage, error) {
	if err := requireYes(p.Yes); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.ModelID) == "" {
		return nil, fmt.Errorf("--id is required")
	}
	if p.Status != 0 && p.Status != 1 {
		return nil, fmt.Errorf("status must be 0 (unpublish) or 1 (publish)")
	}
	body := modelPublishBody(p.ModelID, p.Status)
	data, err := s.client.Post(enum.AIApiPathModelPublish, body)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			return sandboxModelPublish(p.ModelID, p.Status), nil
		}
		return nil, err
	}
	return data, nil
}

type ModelDeleteParams struct {
	ModelID string
	Yes     bool
}

func (s ModelService) DeleteModel(p ModelDeleteParams) (json.RawMessage, error) {
	if err := requireYes(p.Yes); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.ModelID) == "" {
		return nil, fmt.Errorf("--id is required")
	}
	view, err := s.ViewModel(ModelViewParams{ModelID: p.ModelID})
	if err != nil {
		return nil, err
	}
	if modelViewIsPublic(view) {
		return nil, ErrPublishedModelNoDelete
	}
	body := map[string]any{"model_id": jsonNumericID(p.ModelID)}
	data, err := s.client.Post(enum.AIApiPathModelDelete, body)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			return sandboxModelDelete(p.ModelID), nil
		}
		return nil, err
	}
	return data, nil
}

// modelPublishBody matches Console detail.vue: {model_id: <number>, status: 0|1}.
func modelPublishBody(modelID string, status int) map[string]any {
	return map[string]any{
		"model_id": jsonNumericID(modelID),
		"status":   status,
	}
}

func sandboxModelPublish(modelID string, status int) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"model_id": modelID,
		"status":   status,
		"ok":       true,
	})
	return raw
}

func sandboxModelDelete(modelID string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"model_id": modelID,
		"deleted":  true,
	})
	return raw
}

type ModelCreateParams struct {
	FromJSON string
	Target   string // optional: "recamera-pro" enforces Pro task/format/framework rules
	Yes      bool
}

func (s ModelService) CreateModel(p ModelCreateParams) (json.RawMessage, error) {
	if err := requireYes(p.Yes); err != nil {
		return nil, err
	}
	body, err := LoadJSONObject(p.FromJSON)
	if err != nil {
		return nil, err
	}
	if err := rejectForbiddenPublicFlags(body); err != nil {
		return nil, err
	}
	if err := s.rejectPublicModelClone(body); err != nil {
		return nil, err
	}
	constants, err := s.currentModelConstants()
	if err != nil {
		return nil, err
	}
	if err := ValidateAndNormalizeModelPayload(body, constants, ""); err != nil {
		return nil, err
	}
	if IsRecameraProTarget(p.Target) && !modelPayloadUsesUniformType(body, constants, RecameraProUniformType) {
		return nil, fmt.Errorf("--target recamera-pro requires adapteds containing a device with uniform_type=53")
	}
	body = canonicalModelWritePayload(body)
	delete(body, "model_id")
	delete(body, "id")

	data, err := s.client.Post(enum.AIApiPathModelCreate, body)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			data = sandboxModelCreate(body)
		} else {
			return nil, err
		}
	}
	return s.withModelConsoleURL(data), nil
}

func (s ModelService) rejectPublicModelClone(body map[string]any) error {
	for _, id := range sourceModelIDs(body) {
		view, err := s.ViewModel(ModelViewParams{ModelID: id})
		if err != nil {
			continue
		}
		if modelViewIsPublic(view) {
			return ErrPublicModelNoClone
		}
	}
	return nil
}

func sourceModelIDs(body map[string]any) []string {
	if body == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var ids []string
	for _, key := range []string{"id", "model_id", "from_id"} {
		id := strings.TrimSpace(strictScalarString(body[key]))
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func (s ModelService) withModelConsoleURL(data json.RawMessage) json.RawMessage {
	id := extractModelID(data)
	return attachConsoleURL(data, "model_id", id, ModelDetailConsoleURL(s.remoteType, id))
}

type ModelUpdateParams struct {
	ModelID  string
	FromJSON string
	// Path is an optional local model binary (Console edit: replace file_url).
	Path   string
	Target string
	Yes    bool
}

func (s ModelService) UpdateModel(p ModelUpdateParams) (json.RawMessage, error) {
	if err := requireYes(p.Yes); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.ModelID) == "" {
		return nil, fmt.Errorf("--id is required")
	}
	fromJSON := strings.TrimSpace(p.FromJSON)
	localPath := strings.TrimSpace(p.Path)
	if fromJSON == "" && localPath == "" {
		return nil, fmt.Errorf("provide --from-json and/or --path")
	}

	var body map[string]any
	if fromJSON != "" {
		loaded, err := LoadJSONObject(fromJSON)
		if err != nil {
			return nil, err
		}
		body = loaded
	} else {
		body = map[string]any{}
	}

	if localPath != "" {
		existingRaw, err := s.ViewModel(ModelViewParams{ModelID: p.ModelID})
		if err != nil {
			return nil, fmt.Errorf("load model %s before attach: %w", p.ModelID, err)
		}
		if err := s.rejectUnownedPublicModelEdit(p.ModelID, existingRaw); err != nil {
			return nil, err
		}
		files := FileService{client: s.client, sandbox: s.sandbox}
		up, err := files.UploadFile(FileUploadParams{
			Path:     localPath,
			FileType: FileTypeFile,
			Checksum: true,
		})
		if err != nil {
			return nil, err
		}
		body["file_url"] = up.FileURL
		body["checksum"] = up.Checksum
		body["model_size"] = up.Size
	}

	result, err := s.applyModelPatch(p.ModelID, body, p.Target)
	if err != nil {
		return nil, err
	}
	return withModelUpdateMeta(result), nil
}

type ModelUploadParams struct {
	Path    string
	Name    string
	ModelID string
	Yes     bool
	// Unlinked explicitly allows an orphan OSS object for brand-new model create.
	// Without ModelID or Unlinked, UploadAndAttach refuses before any PUT.
	Unlinked bool
}

// UploadAndAttach uploads a model binary (file_type=2 + checksum).
// With --id --yes it writes file_url/checksum/model_size onto that private
// model and auto-increments version the same way Console add.vue does
// (patch +1 when file_url changes). Without --id, callers must pass
// Unlinked=true (--unlinked) or the call fails with ErrAttachRequired
// before any OSS write.
func (s ModelService) UploadAndAttach(p ModelUploadParams) (FileAttachResult, error) {
	modelID := strings.TrimSpace(p.ModelID)
	if modelID != "" && p.Unlinked {
		return FileAttachResult{}, fmt.Errorf("use only one of --id or --unlinked")
	}
	if modelID == "" && !p.Unlinked {
		return FileAttachResult{}, ErrAttachRequired
	}
	var prev string
	if modelID != "" {
		if err := requireYes(p.Yes); err != nil {
			return FileAttachResult{}, fmt.Errorf("%w; --id attaches the file and increments version", err)
		}
		existingRaw, err := s.ViewModel(ModelViewParams{ModelID: modelID})
		if err != nil {
			return FileAttachResult{}, fmt.Errorf("load model %s before attach: %w", modelID, err)
		}
		if err := s.rejectUnownedPublicModelEdit(modelID, existingRaw); err != nil {
			return FileAttachResult{}, err
		}
		var existing map[string]any
		if err := json.Unmarshal(existingRaw, &existing); err != nil || existing == nil {
			return FileAttachResult{}, fmt.Errorf("parse model %s before attach", modelID)
		}
		prev = strictScalarString(existing["version"])
	}

	files := FileService{client: s.client, sandbox: s.sandbox}
	up, err := files.UploadFile(FileUploadParams{
		Path:     p.Path,
		Name:     p.Name,
		FileType: FileTypeFile,
		Checksum: true,
	})
	if err != nil {
		return FileAttachResult{}, err
	}
	out := FileAttachFromUpload(up)
	if modelID == "" {
		out.Hint = UnlinkedUploadHint
		return out, nil
	}

	patch := map[string]any{
		"file_url":   up.FileURL,
		"checksum":   up.Checksum,
		"model_size": up.Size,
	}
	result, err := s.applyModelPatch(modelID, patch, "")
	if err != nil {
		return FileAttachResult{}, err
	}
	out.Attached = true
	out.ModelID = modelID
	out.PreviousVersion = prev
	if result.PreviousVersion != "" {
		out.PreviousVersion = result.PreviousVersion
	}
	out.Version = result.Version
	out.BePublic = result.BePublic
	out.ReReview = result.ReReview
	if result.ReReview {
		out.Hint = ModelReReviewHint
	}
	return out, nil
}

type modelPatchResult struct {
	Raw             json.RawMessage
	PreviousVersion string
	Version         string
	BePublic        int
	ReReview        bool
}

func (s ModelService) applyModelPatch(modelID string, body map[string]any, target string) (modelPatchResult, error) {
	if err := rejectForbiddenPublicFlags(body); err != nil {
		return modelPatchResult{}, err
	}
	existingRaw, err := s.ViewModel(ModelViewParams{ModelID: modelID})
	if err != nil {
		return modelPatchResult{}, fmt.Errorf("load existing model before validation: %w", err)
	}
	if err := s.rejectUnownedPublicModelEdit(modelID, existingRaw); err != nil {
		return modelPatchResult{}, err
	}
	var existing map[string]any
	if err := json.Unmarshal(existingRaw, &existing); err != nil || existing == nil {
		return modelPatchResult{}, fmt.Errorf("parse existing model detail before validation")
	}
	prev := strictScalarString(existing["version"])
	currentPublic := modelViewBePublic(existingRaw)
	merged := mergeModelPayload(existing, body)
	if err := normalizeModelUpdateVersion(existing, merged, body); err != nil {
		return modelPatchResult{}, err
	}
	constants, err := s.currentModelConstants()
	if err != nil {
		return modelPatchResult{}, err
	}
	if err := ValidateAndNormalizeModelPayload(merged, constants, prev); err != nil {
		return modelPatchResult{}, err
	}
	if IsRecameraProTarget(target) && !modelPayloadUsesUniformType(merged, constants, RecameraProUniformType) {
		return modelPatchResult{}, fmt.Errorf("--target recamera-pro requires adapteds containing a device with uniform_type=53")
	}
	body = canonicalModelWritePayload(merged)
	body["model_id"] = modelID
	// Console edit_model requires both id and model_id (aiserver 11202 if id missing).
	body["id"] = modelID
	// Console requestConfirmFn: keep the publish checkbox on → send be_public=1 so
	// a published model re-enters review after any edit.
	reReview := currentPublic == 1
	if reReview {
		body["be_public"] = 1
	}

	next := strictScalarString(merged["version"])
	data, err := s.client.Post(enum.AIApiPathModelEdit, body)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			data = sandboxModelUpdate(modelID, body)
		} else {
			return modelPatchResult{}, err
		}
	}
	outPublic := currentPublic
	if reReview {
		outPublic = 1
	}
	return modelPatchResult{
		Raw:             data,
		PreviousVersion: prev,
		Version:         next,
		BePublic:        outPublic,
		ReReview:        reReview,
	}, nil
}

func (s ModelService) rejectUnownedPublicModelEdit(modelID string, view json.RawMessage) error {
	if !modelViewIsPublic(view) {
		return nil
	}
	if s.sandbox {
		if sandboxIsForeignPublicID(modelID) {
			return fmt.Errorf("%w (%s)", ErrForeignPublicModelNoEdit, modelID)
		}
		return nil
	}
	if uid := strings.TrimSpace(s.userID); uid != "" {
		if author := modelViewAuthor(view); author != "" && author != uid {
			return fmt.Errorf("%w (%s)", ErrForeignPublicModelNoEdit, modelID)
		}
	}
	return nil
}

func withModelUpdateMeta(result modelPatchResult) json.RawMessage {
	data := result.Raw
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil || m == nil {
		return data
	}
	if result.PreviousVersion != "" {
		m["previous_version"] = result.PreviousVersion
	}
	if result.Version != "" {
		m["version"] = result.Version
	}
	if result.BePublic != 0 {
		m["be_public"] = result.BePublic
	}
	if result.ReReview {
		m["re_review"] = true
		m["hint"] = ModelReReviewHint
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return data
	}
	return raw
}

func (s ModelService) currentModelConstants() (modelConstants, error) {
	raw, err := s.Constants()
	if err != nil {
		return modelConstants{}, fmt.Errorf("load model constants: %w", err)
	}
	return parseModelConstants(raw)
}

func modelPayloadUsesUniformType(body map[string]any, constants modelConstants, want int) bool {
	uniformTypes, err := resolveModelUniformTypes(body["adapteds"], constants.deviceUniformTypes)
	return err == nil && containsInt(uniformTypes, want)
}

func sandboxModelCreate(body map[string]any) json.RawMessage {
	// Console create often returns bare id in data.
	raw, _ := json.Marshal("sandbox-created-model")
	_ = body
	return raw
}

func sandboxModelUpdate(modelID string, body map[string]any) json.RawMessage {
	out := map[string]any{"model_id": modelID, "updated": true}
	for k, v := range body {
		if k == "model_id" {
			continue
		}
		out[k] = v
	}
	raw, _ := json.Marshal(out)
	return raw
}
