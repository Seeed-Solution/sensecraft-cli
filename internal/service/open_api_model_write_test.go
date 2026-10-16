package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/config"
)

func TestJSONNumericID(t *testing.T) {
	if got, ok := jsonNumericID("63758").(int64); !ok || got != 63758 {
		t.Fatalf("numeric id: got %#v", jsonNumericID("63758"))
	}
	if got, ok := jsonNumericID(" sandbox-model ").(string); !ok || got != "sandbox-model" {
		t.Fatalf("non-numeric id: got %#v", jsonNumericID(" sandbox-model "))
	}
	raw, err := json.Marshal(jsonNumericID("63758"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "63758" {
		t.Fatalf("want JSON number 63758, got %s", raw)
	}
	raw, err = json.Marshal(jsonNumericID("sandbox-model"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `"sandbox-model"` {
		t.Fatalf("want JSON string, got %s", raw)
	}
}

func TestModelPublishBodyUsesJSONNumberID(t *testing.T) {
	raw, err := json.Marshal(modelPublishBody("63758", 1))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"model_id":63758,"status":1}` {
		t.Fatalf("publish body: %s", raw)
	}
	raw, err = json.Marshal(modelPublishBody("63758", 0))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"model_id":63758,"status":0}` {
		t.Fatalf("unpublish body: %s", raw)
	}
}

func TestModelPublishRejectsBadStatus(t *testing.T) {
	s := ModelService{}
	_, err := s.PublishModel(ModelPublishParams{ModelID: "m1", Status: 2, Yes: true})
	if err == nil {
		t.Fatal("expected error for status=2")
	}
	_, err = s.PublishModel(ModelPublishParams{ModelID: "m1", Status: 1, Yes: false})
	if err == nil {
		t.Fatal("expected confirmation required")
	}
}

func TestCreateModelRunsConsoleCompatibleValidation(t *testing.T) {
	s := GetModelService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	body := validModelPayload("sandbox-pro", 6, "https://example.com/model.rknn")
	body["task"] = 2
	path := writeModelPayloadFile(t, body)

	raw, err := s.CreateModel(ModelCreateParams{FromJSON: path, Yes: true})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result["model_id"] != "sandbox-created-model" {
		t.Fatalf("unexpected create result: %#v", result)
	}
}

func TestCreateModelRejectsPublicLibraryClone(t *testing.T) {
	s := GetModelService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	body := validModelPayload("sandbox-pro", 6, "https://example.com/model.rknn")
	body["task"] = 2
	body["id"] = "public-60768"
	path := writeModelPayloadFile(t, body)

	_, err := s.CreateModel(ModelCreateParams{FromJSON: path, Yes: true})
	if !errors.Is(err, ErrPublicModelNoClone) {
		t.Fatalf("want ErrPublicModelNoClone, got %v", err)
	}

	body["id"] = "mine-1"
	body["from_id"] = "public-60768"
	path = writeModelPayloadFile(t, body)
	_, err = s.CreateModel(ModelCreateParams{FromJSON: path, Yes: true})
	if !errors.Is(err, ErrPublicModelNoClone) {
		t.Fatalf("from_id want ErrPublicModelNoClone, got %v", err)
	}

	delete(body, "from_id")
	body["id"] = "mine-1"
	path = writeModelPayloadFile(t, body)
	if _, err := s.CreateModel(ModelCreateParams{FromJSON: path, Yes: true}); err != nil {
		t.Fatalf("private source id should still create, got %v", err)
	}
}

func TestUploadAndAttachBumpsVersionLikeConsole(t *testing.T) {
	s := GetModelService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	path := filepath.Join(t.TempDir(), "model.tflite")
	if err := os.WriteFile(path, []byte("tflite-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := s.UploadAndAttach(ModelUploadParams{Path: path, ModelID: "mine-1"})
	if !errors.Is(err, ErrConfirmRequired) {
		t.Fatalf("want confirm required before upload, got %v", err)
	}

	_, err = s.UploadAndAttach(ModelUploadParams{Path: path})
	if !errors.Is(err, ErrAttachRequired) {
		t.Fatalf("want attach required without --id/--unlinked, got %v", err)
	}

	unlinked, err := s.UploadAndAttach(ModelUploadParams{Path: path, Unlinked: true})
	if err != nil {
		t.Fatal(err)
	}
	if unlinked.Attached || unlinked.Hint == "" {
		t.Fatalf("unlinked upload should hint, got %+v", unlinked)
	}

	_, err = s.UploadAndAttach(ModelUploadParams{Path: path, ModelID: "public-60768", Yes: true})
	if !errors.Is(err, ErrForeignPublicModelNoEdit) {
		t.Fatalf("want foreign public refuse, got %v", err)
	}

	got, err := s.UploadAndAttach(ModelUploadParams{Path: path, ModelID: "mine-1", Yes: true})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Attached || got.ModelID != "mine-1" {
		t.Fatalf("expected attach, got %+v", got)
	}
	if got.PreviousVersion != "1.0.0" || got.Version != "1.0.1" {
		t.Fatalf("want Console patch increment 1.0.0 → 1.0.1, got %+v", got)
	}
	if got.FileURL == "" || got.Checksum == "" {
		t.Fatalf("upload fields missing: %+v", got)
	}
}

func TestUpdateModelMergesExistingDetailBeforeValidation(t *testing.T) {
	s := GetModelService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	path := writeModelPayloadFile(t, map[string]any{
		"description": "Updated description",
	})
	raw, err := s.UpdateModel(ModelUpdateParams{
		ModelID:  "sandbox-model",
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
	if result["description"] != "Updated description" {
		t.Fatalf("unexpected update result: %#v", result)
	}
	if result["previous_version"] != "1.0.0" {
		t.Fatalf("want previous_version 1.0.0, got %#v", result["previous_version"])
	}
	if result["version"] != "1.0.0" {
		t.Fatalf("description-only update must not bump version, got %#v", result["version"])
	}
}

func TestUpdateModelPathBumpsVersionLikeConsole(t *testing.T) {
	s := GetModelService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	bin := filepath.Join(t.TempDir(), "model.tflite")
	if err := os.WriteFile(bin, []byte("tflite-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := s.UpdateModel(ModelUpdateParams{ModelID: "mine-1", Path: bin})
	if !errors.Is(err, ErrConfirmRequired) {
		t.Fatalf("want confirm required, got %v", err)
	}
	_, err = s.UpdateModel(ModelUpdateParams{ModelID: "mine-1"})
	if err == nil {
		t.Fatal("want error when neither --from-json nor --path")
	}
	_, err = s.UpdateModel(ModelUpdateParams{ModelID: "public-60768", Path: bin, Yes: true})
	if !errors.Is(err, ErrForeignPublicModelNoEdit) {
		t.Fatalf("want foreign public refuse, got %v", err)
	}

	raw, err := s.UpdateModel(ModelUpdateParams{ModelID: "mine-1", Path: bin, Yes: true})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result["previous_version"] != "1.0.0" || result["version"] != "1.0.1" {
		t.Fatalf("want 1.0.0 → 1.0.1, got %#v", result)
	}
	fileURL, _ := result["file_url"].(string)
	if !strings.Contains(fileURL, "model.tflite") {
		t.Fatalf("expected uploaded file_url, got %#v", result["file_url"])
	}

	meta := writeModelPayloadFile(t, map[string]any{"description": "new file and copy"})
	raw, err = s.UpdateModel(ModelUpdateParams{
		ModelID:  "mine-1",
		FromJSON: meta,
		Path:     bin,
		Yes:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result["description"] != "new file and copy" {
		t.Fatalf("json+path should merge description: %#v", result)
	}
	if result["version"] != "1.0.1" {
		t.Fatalf("file change should still bump version, got %#v", result["version"])
	}
}

func TestUpdateModelLabelsBumpVersionLikeConsole(t *testing.T) {
	s := GetModelService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	path := writeModelPayloadFile(t, map[string]any{
		"labels": []any{map[string]any{"object_id": 0, "object_name": "cat"}},
	})
	raw, err := s.UpdateModel(ModelUpdateParams{
		ModelID:  "mine-1",
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
	if result["previous_version"] != "1.0.0" || result["version"] != "1.0.1" {
		t.Fatalf("labels change should incrementVersion, got %#v", result)
	}
}

func TestUpdateOwnPublishedModelReentersReview(t *testing.T) {
	s := GetModelService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	bin := filepath.Join(t.TempDir(), "model.tflite")
	if err := os.WriteFile(bin, []byte("tflite-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	raw, err := s.UpdateModel(ModelUpdateParams{ModelID: "mine-published-1", Path: bin, Yes: true})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result["re_review"] != true {
		t.Fatalf("own published model should re-enter review, got %#v", result)
	}
	if result["be_public"] != float64(1) {
		t.Fatalf("Console edit sends be_public=1, got %#v", result["be_public"])
	}
	if result["previous_version"] != "1.0.0" || result["version"] != "1.0.1" {
		t.Fatalf("file change should still bump version, got %#v", result)
	}

	got, err := s.UploadAndAttach(ModelUploadParams{Path: bin, ModelID: "mine-published-1", Yes: true})
	if err != nil {
		t.Fatal(err)
	}
	if !got.ReReview || got.BePublic != 1 {
		t.Fatalf("upload attach on own published should re-review, got %+v", got)
	}
}

func TestDeleteRefusesPublishedAndAllowsPrivate(t *testing.T) {
	s := GetModelService(app.AICmdContext{
		Ctx:     context.Background(),
		Access:  config.AccessApiConfig{RemoteType: "Global"},
		SandBox: true,
	})
	_, err := s.DeleteModel(ModelDeleteParams{ModelID: "mine-published-1", Yes: true})
	if !errors.Is(err, ErrPublishedModelNoDelete) {
		t.Fatalf("want published delete refuse, got %v", err)
	}
	_, err = s.DeleteModel(ModelDeleteParams{ModelID: "mine-review-1", Yes: true})
	if !errors.Is(err, ErrPublishedModelNoDelete) {
		t.Fatalf("want under-review delete refuse, got %v", err)
	}
	raw, err := s.DeleteModel(ModelDeleteParams{ModelID: "mine-1", Yes: true})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result["deleted"] != true {
		t.Fatalf("private delete should succeed, got %#v", result)
	}
}

func TestRejectUnownedPublicWhenAuthorMismatch(t *testing.T) {
	s := ModelService{userID: "me"}
	err := s.rejectUnownedPublicModelEdit("99", json.RawMessage(`{"be_public":1,"author":"other"}`))
	if !errors.Is(err, ErrForeignPublicModelNoEdit) {
		t.Fatalf("want foreign refuse, got %v", err)
	}
	if err := s.rejectUnownedPublicModelEdit("99", json.RawMessage(`{"be_public":1,"author":"me"}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.rejectUnownedPublicModelEdit("99", json.RawMessage(`{"be_public":0,"author":"other"}`)); err != nil {
		t.Fatal(err)
	}
}

func writeModelPayloadFile(t *testing.T, body map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "model.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
