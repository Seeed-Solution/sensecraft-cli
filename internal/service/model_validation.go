package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
)

var modelVersionPattern = regexp.MustCompile(`^[1-9]\d?(\.([1-9]?\d)){2}$`)

type modelConstants struct {
	deviceUniformTypes map[string]int
	tasks              map[string]struct{}
	formats            map[string]struct{}
	scenarios          map[string]struct{}
	frameworks         map[string]struct{}
	precisions         map[string]struct{}
}

func parseModelConstants(raw json.RawMessage) (modelConstants, error) {
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return modelConstants{}, fmt.Errorf("parse get_constant response: %w", err)
	}
	out := modelConstants{
		deviceUniformTypes: map[string]int{},
		tasks:              optionIDs(body["model_task_array"]),
		formats:            optionIDs(body["model_format_array"]),
		// Console model/add.vue uses scenarios_array; fall back for older payloads.
		scenarios:  optionIDsPrefer(body, "scenarios_array", "app_scenarios_array"),
		frameworks: optionIDs(body["ai_framework_array"]),
		precisions: optionIDs(body["model_precision_array"]),
	}
	devices, ok := body["device_type_array"].([]any)
	if !ok {
		return modelConstants{}, fmt.Errorf("get_constant response is missing device_type_array")
	}
	for _, item := range devices {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := strictScalarString(m["id"])
		uniformType, ok := strictInt(m["uniform_type"])
		if id != "" && ok {
			out.deviceUniformTypes[id] = uniformType
		}
	}
	if len(out.deviceUniformTypes) == 0 {
		return modelConstants{}, fmt.Errorf("get_constant device_type_array has no usable devices")
	}
	return out, nil
}

func optionIDs(raw any) map[string]struct{} {
	out := map[string]struct{}{}
	arr, _ := raw.([]any)
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id := strictScalarString(m["id"]); id != "" {
			out[id] = struct{}{}
		}
	}
	return out
}

func optionIDsPrefer(body map[string]any, keys ...string) map[string]struct{} {
	for _, key := range keys {
		if ids := optionIDs(body[key]); len(ids) > 0 {
			return ids
		}
	}
	return map[string]struct{}{}
}

func strictScalarString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int(t)) {
			return strconv.Itoa(int(t))
		}
	case json.Number:
		return strings.TrimSpace(t.String())
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	}
	return ""
}

func strictInt(v any) (int, bool) {
	s := strictScalarString(v)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

func mergeModelPayload(base, patch map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(patch))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range patch {
		out[k] = v
	}
	return out
}

func canonicalModelWritePayload(model map[string]any) map[string]any {
	allowed := []string{
		"name", "description", "content", "preparation", "adapteds", "pic_url",
		"model_format", "task", "ai_framework", "labels", "file_url", "version",
		"precision", "checksum", "scenario", "attr", "model_size", "author_name",
	}
	out := make(map[string]any, len(allowed))
	for _, key := range allowed {
		if value, ok := model[key]; ok {
			out[key] = value
		}
	}
	return out
}

// ValidateAndNormalizeModelPayload mirrors the constraints enforced by Console views/model/add.vue.
// The input must be a complete model. Update callers should merge the existing detail first.
func ValidateAndNormalizeModelPayload(body map[string]any, constants modelConstants, oldVersion string) error {
	if body == nil {
		return fmt.Errorf("model JSON is empty")
	}

	name, err := requiredModelString(body, "name")
	if err != nil {
		return err
	}
	if jsStringLength(name) > 64 {
		return fmt.Errorf("name max length is 64")
	}
	body["name"] = name

	description, err := requiredModelString(body, "description")
	if err != nil {
		return err
	}
	if jsStringLength(description) > 300 {
		return fmt.Errorf("description max length is 300")
	}
	body["description"] = description

	if _, err := requiredModelString(body, "content"); err != nil {
		return err
	}
	picURL, err := requiredModelString(body, "pic_url")
	if err != nil {
		return err
	}
	if err := validateHTTPOrHTTPSURL(picURL, "pic_url"); err != nil {
		return err
	}
	fileURL, err := requiredModelString(body, "file_url")
	if err != nil {
		return err
	}
	if err := validateHTTPOrHTTPSURL(fileURL, "file_url"); err != nil {
		return err
	}
	if _, err := requiredModelString(body, "checksum"); err != nil {
		return fmt.Errorf("checksum is required (use model upload and keep it paired with file_url)")
	}
	if _, err := requiredOption(body, "precision", constants.precisions); err != nil {
		return err
	}

	task, err := requiredOption(body, "task", constants.tasks)
	if err != nil {
		return err
	}
	format, err := requiredOption(body, "model_format", nil)
	if err != nil {
		return err
	}
	if _, err := requiredOption(body, "scenario", constants.scenarios); err != nil {
		return err
	}
	if err := validateModelVersion(strictScalarString(body["version"]), oldVersion); err != nil {
		return err
	}
	if err := validateModelLabels(body["labels"]); err != nil {
		return err
	}

	uniformTypes, err := resolveModelUniformTypes(body["adapteds"], constants.deviceUniformTypes)
	if err != nil {
		return err
	}
	if err := validateModelDeviceGroup(uniformTypes); err != nil {
		return err
	}

	isPro := containsInt(uniformTypes, RecameraProUniformType)
	if _, exists := constants.formats[strconv.Itoa(format)]; !exists &&
		!(isPro && task == RecameraProTaskAudio &&
			(format == RecameraProModelFormatALPKG || format == RecameraProModelFormatZIP)) {
		return fmt.Errorf("model_format=%d is not present in current get_constant options", format)
	}
	if isPro {
		if fw := strictScalarString(body["ai_framework"]); fw != "" && len(constants.frameworks) > 0 {
			if _, ok := constants.frameworks[fw]; !ok {
				return fmt.Errorf("ai_framework=%s is not present in current get_constant options", fw)
			}
		}
		if err := NormalizeRecameraProModelPayload(body); err != nil {
			return err
		}
	} else {
		delete(body, "ai_framework")
		if err := validateNonProModelFormat(uniformTypes, format); err != nil {
			return err
		}
		if err := validateModelFileExtension(strictScalarString(body["file_url"]), format); err != nil {
			return err
		}
	}

	// Pro validates its own framework/task combination. Non-Pro Console never submits ai_framework.
	return nil
}

func requiredModelString(body map[string]any, key string) (string, error) {
	raw, ok := body[key]
	if !ok || raw == nil {
		return "", fmt.Errorf("%s is required", key)
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func requiredOption(body map[string]any, key string, allowed map[string]struct{}) (int, error) {
	value := strictScalarString(body[key])
	if value == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	if len(allowed) > 0 {
		if _, ok := allowed[value]; !ok {
			return 0, fmt.Errorf("%s=%s is not present in current get_constant options", key, value)
		}
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a numeric option id", key)
	}
	return n, nil
}

func validateModelVersion(version, oldVersion string) error {
	if !modelVersionPattern.MatchString(version) {
		return fmt.Errorf("version must match major.minor.patch (1-99.0-99.0-99, no leading zeros)")
	}
	if strings.TrimSpace(oldVersion) == "" {
		return nil
	}
	if !modelVersionPattern.MatchString(strings.TrimSpace(oldVersion)) {
		return nil
	}
	if compareModelVersions(version, oldVersion) < 0 {
		return fmt.Errorf("version %s cannot be lower than existing version %s", version, oldVersion)
	}
	return nil
}

func compareModelVersions(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		an, _ := strconv.Atoi(ap[i])
		bn, _ := strconv.Atoi(bp[i])
		if an < bn {
			return -1
		}
		if an > bn {
			return 1
		}
	}
	return 0
}

func normalizeModelUpdateVersion(existing, merged, patch map[string]any) error {
	oldVersion := strictScalarString(existing["version"])
	newVersion := strictScalarString(merged["version"])
	if oldVersion == "" || newVersion != oldVersion {
		return nil
	}
	labelsChanged := !reflect.DeepEqual(existing["labels"], merged["labels"])
	fileChanged := strictScalarString(existing["file_url"]) != strictScalarString(merged["file_url"])
	if !labelsChanged && !fileChanged {
		return nil
	}
	parts := strings.Split(oldVersion, ".")
	if len(parts) != 3 {
		return fmt.Errorf("existing version %q cannot be incremented", oldVersion)
	}
	patchNumber, err := strconv.Atoi(parts[2])
	if err != nil || patchNumber >= 99 {
		return fmt.Errorf("labels/file changed but version %s cannot auto-increment within Console version constraints; provide a higher valid version", oldVersion)
	}
	next := fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patchNumber+1)
	merged["version"] = next
	patch["version"] = next
	return nil
}

func jsStringLength(value string) int {
	return len(utf16.Encode([]rune(value)))
}

func validateModelLabels(raw any) error {
	labels, ok := raw.([]any)
	if !ok || len(labels) == 0 {
		return fmt.Errorf("labels must be a non-empty array")
	}
	for i, item := range labels {
		label, ok := item.(map[string]any)
		if !ok {
			return fmt.Errorf("labels[%d] must be an object", i)
		}
		id, exists := label["object_id"]
		if !exists || id == nil || (strictScalarString(id) == "" && id != float64(0) && id != 0) {
			return fmt.Errorf("labels[%d].object_id is required", i)
		}
		name, ok := label["object_name"].(string)
		if !ok || !validateModelClassName(name) {
			return fmt.Errorf("labels[%d].object_name must start with a non-digit, contain a letter, and use only ASCII letters, digits, or spaces", i)
		}
	}
	return nil
}

func validateModelClassName(value string) bool {
	if value == "" {
		return false
	}
	hasLetter := false
	for i, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			hasLetter = true
			continue
		}
		if r >= '0' && r <= '9' {
			if i == 0 {
				return false
			}
			continue
		}
		if r != ' ' {
			return false
		}
	}
	return hasLetter
}

func resolveModelUniformTypes(raw any, deviceMap map[string]int) ([]int, error) {
	adapteds, ok := raw.([]any)
	if !ok || len(adapteds) == 0 {
		return nil, fmt.Errorf("adapteds must be a non-empty array")
	}
	out := make([]int, 0, len(adapteds))
	for i, adapted := range adapteds {
		id := strictScalarString(adapted)
		if id == "" {
			return nil, fmt.Errorf("adapteds[%d] must be a device option id", i)
		}
		uniformType, ok := deviceMap[id]
		if !ok {
			return nil, fmt.Errorf("adapteds[%d]=%s is not present in current device_type_array", i, id)
		}
		out = append(out, uniformType)
	}
	return out, nil
}

func validateModelDeviceGroup(uniformTypes []int) error {
	if len(uniformTypes) == 0 {
		return fmt.Errorf("adapteds must be non-empty")
	}
	group := modelDeviceGroup(uniformTypes[0])
	for _, uniformType := range uniformTypes[1:] {
		if modelDeviceGroup(uniformType) != group {
			return fmt.Errorf("adapteds contains mutually exclusive device families")
		}
	}
	if group == "single" && len(uniformTypes) > 1 {
		return fmt.Errorf("selected device type only allows one adapted device")
	}
	return nil
}

func modelDeviceGroup(uniformType int) string {
	switch uniformType {
	case 32, 52:
		return "xiao-respeaker"
	case 36, 37, 41:
		return "vision-family"
	case 26:
		return "jetson"
	default:
		return "single"
	}
}

func validateNonProModelFormat(uniformTypes []int, format int) error {
	switch modelDeviceGroup(uniformTypes[0]) {
	case "xiao-respeaker", "vision-family":
		if format != 2 {
			return fmt.Errorf("selected XIAO/reSpeaker/Vision family devices require model_format=2 (tflite)")
		}
	case "jetson":
		if format != 3 && format != 4 {
			return fmt.Errorf("Jetson/reComputer devices require model_format=3 (engine) or 4 (pt)")
		}
	default:
		if uniformTypes[0] == 40 {
			if format != 5 {
				return fmt.Errorf("classic reCamera requires model_format=5 (cvimodel)")
			}
		} else if format == 2 {
			return fmt.Errorf("selected device does not allow model_format=2 (tflite)")
		}
	}
	return nil
}

func validateModelFileExtension(fileURL string, format int) error {
	parsed, err := url.Parse(fileURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("file_url is invalid (expect http(s) URL)")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("file_url is invalid (expect http(s) URL)")
	}
	ext := strings.ToLower(filepath.Ext(parsed.Path))
	expected := map[int]string{
		1: ".onnx",
		2: ".tflite",
		3: ".engine",
		4: ".pt",
		5: ".cvimodel",
		6: ".rknn",
		7: ".alpkg",
		8: ".zip",
	}[format]
	if expected != "" && ext != expected {
		return fmt.Errorf("model_format=%d requires %s file_url", format, expected)
	}
	return nil
}

func containsInt(values []int, want int) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
