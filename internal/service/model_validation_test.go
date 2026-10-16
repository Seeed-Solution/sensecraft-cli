package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func testModelConstants() modelConstants {
	return modelConstants{
		deviceUniformTypes: map[string]int{
			"xiao": 32, "respeaker": 52,
			"vision": 36, "watcher": 37,
			"jetson-a": 26, "jetson-b": 26,
			"recamera": 40, "pro": 53, "other": 99,
		},
		tasks:      map[string]struct{}{"1": {}, "2": {}, "4": {}, "6": {}, "7": {}},
		formats:    map[string]struct{}{"1": {}, "2": {}, "3": {}, "4": {}, "5": {}, "6": {}, "7": {}, "8": {}},
		scenarios:  map[string]struct{}{"1": {}},
		frameworks: map[string]struct{}{"1": {}, "2": {}, "8": {}, "9": {}, "10": {}, "11": {}, "12": {}, "13": {}, "14": {}, "15": {}},
		precisions: map[string]struct{}{"1": {}, "2": {}, "3": {}},
	}
}

func validModelPayload(device string, format int, file string) map[string]any {
	return map[string]any{
		"name":         "Model",
		"description":  "Description",
		"content":      "<p>Details</p>",
		"pic_url":      "https://example.com/example.png",
		"adapteds":     []any{device},
		"task":         1,
		"model_format": format,
		"labels": []any{
			map[string]any{"object_id": 0, "object_name": "person 1"},
		},
		"scenario":  "1",
		"file_url":  file,
		"checksum":  "0123456789abcdef",
		"version":   "1.0.0",
		"precision": "1",
	}
}

func TestValidateModelPayloadRequiredFields(t *testing.T) {
	body := validModelPayload("vision", 2, "https://example.com/model.tflite")
	delete(body, "description")
	err := ValidateAndNormalizeModelPayload(body, testModelConstants(), "")
	if err == nil || !strings.Contains(err.Error(), "description is required") {
		t.Fatalf("got %v", err)
	}

	body = validModelPayload("vision", 2, "https://example.com/model.tflite")
	body["name"] = ""
	err = ValidateAndNormalizeModelPayload(body, testModelConstants(), "")
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("got %v", err)
	}

	body = validModelPayload("vision", 2, "https://example.com/model.tflite")
	body["description"] = strings.Repeat("a", 301)
	err = ValidateAndNormalizeModelPayload(body, testModelConstants(), "")
	if err == nil || !strings.Contains(err.Error(), "description max length") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateModelPayloadURLAndPrecision(t *testing.T) {
	body := validModelPayload("vision", 2, "https://example.com/model.tflite")
	body["pic_url"] = "not-a-url"
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err == nil {
		t.Fatal("expected invalid pic_url rejection")
	}

	body = validModelPayload("vision", 2, "ftp://example.com/model.tflite")
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err == nil {
		t.Fatal("expected invalid file_url scheme rejection")
	}

	body = validModelPayload("vision", 2, "https://example.com/model.tflite")
	body["precision"] = "99"
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err == nil {
		t.Fatal("expected invalid precision rejection")
	}

	body = validModelPayload("vision", 2, "https://example.com/model.tflite")
	body["scenario"] = "INVALID"
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err == nil {
		t.Fatal("expected invalid scenario rejection")
	}

	body = validModelPayload("vision", 2, "https://example.com/model.tflite")
	body["task"] = nil
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err == nil {
		t.Fatal("expected null task rejection")
	}

	body = validModelPayload("vision", 2, "https://example.com/model.tflite")
	body["labels"] = []any{}
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err == nil {
		t.Fatal("expected empty labels rejection")
	}
}

func TestParseModelConstantsPrefersScenariosArray(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"app_scenarios_array":   []map[string]any{{"id": "app-only", "name": "App"}},
		"scenarios_array":       []map[string]any{{"id": "model-1", "name": "Model"}},
		"model_precision_array": []map[string]any{{"id": "1", "name": "INT8"}},
		"model_task_array":      []map[string]any{{"id": "1", "name": "Detection"}},
		"model_format_array":    []map[string]any{{"id": "2", "name": "tflite"}},
		"ai_framework_array":    []map[string]any{},
		"device_type_array":     []map[string]any{{"id": "vision", "uniform_type": 36}},
	})
	c, err := parseModelConstants(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.scenarios["model-1"]; !ok {
		t.Fatalf("expected scenarios_array id, got %#v", c.scenarios)
	}
	if _, ok := c.scenarios["app-only"]; ok {
		t.Fatal("must prefer scenarios_array over app_scenarios_array")
	}
	if _, ok := c.precisions["1"]; !ok {
		t.Fatal("expected model_precision_array")
	}
}

func TestValidateModelPayloadUsesConsoleUTF16Length(t *testing.T) {
	body := validModelPayload("vision", 2, "https://example.com/model.tflite")
	body["name"] = strings.Repeat("😀", 33) // 66 UTF-16 code units
	err := ValidateAndNormalizeModelPayload(body, testModelConstants(), "")
	if err == nil || !strings.Contains(err.Error(), "64") {
		t.Fatalf("expected Console-compatible UTF-16 length rejection, got %v", err)
	}
}

func TestValidateModelPayloadVersionAndLabels(t *testing.T) {
	body := validModelPayload("vision", 2, "https://example.com/model.tflite")
	body["version"] = "1.01.0"
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err == nil {
		t.Fatal("expected invalid version")
	}

	body = validModelPayload("vision", 2, "https://example.com/model.tflite")
	body["labels"] = []any{map[string]any{"object_id": 0, "object_name": "1person"}}
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err == nil {
		t.Fatal("expected invalid class name")
	}

	body = validModelPayload("vision", 2, "https://example.com/model.tflite")
	body["version"] = "1.2.2"
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), "1.2.3"); err == nil {
		t.Fatal("expected version downgrade rejection")
	}
	body["version"] = "1.2.3"
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), "1.2.3"); err != nil {
		t.Fatalf("equal version is allowed by Console: %v", err)
	}
}

func TestValidateModelPayloadDeviceGroupsAndFormats(t *testing.T) {
	body := validModelPayload("xiao", 2, "https://example.com/model.tflite")
	body["adapteds"] = []any{"xiao", "respeaker"}
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err != nil {
		t.Fatal(err)
	}

	body["adapteds"] = []any{"xiao", "vision"}
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err == nil {
		t.Fatal("expected mutually exclusive family rejection")
	}

	body = validModelPayload("jetson-a", 4, "https://example.com/model.pt")
	body["adapteds"] = []any{"jetson-a", "jetson-b"}
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err != nil {
		t.Fatal(err)
	}

	body = validModelPayload("recamera", 2, "https://example.com/model.tflite")
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err == nil {
		t.Fatal("classic reCamera must require cvimodel")
	}
}

func TestValidateModelPayloadAutoDetectsRecameraPro(t *testing.T) {
	body := validModelPayload("pro", 6, "https://example.com/model.rknn")
	body["task"] = 2
	body["ai_framework"] = "2"
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err != nil {
		t.Fatal(err)
	}
	if _, exists := body["ai_framework"]; exists {
		t.Fatal("Pro classification must strip ai_framework automatically")
	}

	body = validModelPayload("pro", 6, "https://example.com/model.rknn")
	body["task"] = 4
	body["ai_framework"] = "1"
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err == nil {
		t.Fatal("Pro pose must reject non-YOLOV8 framework")
	}
}

func TestValidateModelPayloadStripsNonProFramework(t *testing.T) {
	body := validModelPayload("vision", 2, "https://example.com/model.tflite")
	body["ai_framework"] = "2"
	if err := ValidateAndNormalizeModelPayload(body, testModelConstants(), ""); err != nil {
		t.Fatal(err)
	}
	if _, exists := body["ai_framework"]; exists {
		t.Fatal("non-Pro payload must not submit ai_framework")
	}
}

func TestNormalizeModelUpdateVersion(t *testing.T) {
	existing := validModelPayload("vision", 2, "https://example.com/old.tflite")
	merged := mergeModelPayload(existing, map[string]any{
		"file_url": "https://example.com/new.tflite",
		"checksum": "new-md5",
	})
	patch := map[string]any{
		"file_url": "https://example.com/new.tflite",
		"checksum": "new-md5",
	}
	if err := normalizeModelUpdateVersion(existing, merged, patch); err != nil {
		t.Fatal(err)
	}
	if patch["version"] != "1.0.1" || merged["version"] != "1.0.1" {
		t.Fatalf("version was not incremented: patch=%#v merged=%#v", patch, merged)
	}

	existing["version"] = "1.0.99"
	merged = mergeModelPayload(existing, map[string]any{"file_url": "https://example.com/newer.tflite"})
	if err := normalizeModelUpdateVersion(existing, merged, map[string]any{}); err == nil {
		t.Fatal("expected patch overflow error")
	}
}

func TestCanonicalModelWritePayloadOmitsConditionalAndServerFields(t *testing.T) {
	model := validModelPayload("pro", 6, "https://example.com/model.rknn")
	model["server_only"] = "drop"
	model["be_public"] = 1
	delete(model, "ai_framework")
	out := canonicalModelWritePayload(model)
	if _, ok := out["server_only"]; ok {
		t.Fatal("server-only fields must not be written back")
	}
	if _, ok := out["be_public"]; ok {
		t.Fatal("visibility must stay on model publish/unpublish path")
	}
	if _, ok := out["ai_framework"]; ok {
		t.Fatal("omitted conditional framework must remain omitted")
	}
}
