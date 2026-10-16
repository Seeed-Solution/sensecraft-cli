package service

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"sensecraft-cloud-terminal/internal/utils"
)

// LoadJSONObject reads a JSON object from a file path, or stdin when path is "-".
func LoadJSONObject(path string) (map[string]any, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("--from-json is required")
	}

	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
	} else {
		data, err = utils.ReadBytes(path)
		if err != nil {
			return nil, err
		}
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("JSON payload is empty")
	}

	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}
	if obj == nil {
		return nil, fmt.Errorf("JSON root must be an object")
	}
	return obj, nil
}

func stringFromMap(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%.0f", t))
	case json.Number:
		return strings.TrimSpace(t.String())
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func rejectForbiddenPublicFlags(body map[string]any) error {
	if v, ok := body["privacy"]; ok {
		n, ok := asInt(v)
		if ok && n == 1 {
			return fmt.Errorf("privacy=1 is forbidden; use app publish (privacy=2) after user confirmation")
		}
	}
	if v, ok := body["be_public"]; ok {
		n, ok := asInt(v)
		if ok && n == 1 {
			return fmt.Errorf("be_public=1 is forbidden from CLI; use app publish / model publish after user confirmation")
		}
	}
	return nil
}

// jsonNumericID encodes a CLI --id as a JSON number when it is all digits
// (aiserver bind of model_id as int rejects string "63758" with 11202).
// Non-numeric ids (sandbox fixtures) stay strings.
func jsonNumericID(id string) any {
	s := strings.TrimSpace(id)
	if s == "" {
		return id
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return s
	}
	return n
}

func asInt(v any) (int, bool) {
	switch t := v.(type) {
	case float64:
		return int(t), true
	case int:
		return t, true
	case int64:
		return int(t), true
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	case string:
		var n int
		_, err := fmt.Sscanf(strings.TrimSpace(t), "%d", &n)
		return n, err == nil
	default:
		return 0, false
	}
}

func validateAppName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("app_name is required")
	}
	if utf8.RuneCountInString(name) > maxAppNameRunes {
		return fmt.Errorf("app_name max length is %d", maxAppNameRunes)
	}
	return nil
}

func validateModelName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if utf8.RuneCountInString(name) > 64 {
		return fmt.Errorf("name max length is 64")
	}
	return nil
}
