package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/enum"
)

const (
	maxAppNameRunes    = 60
	maxAppExcerptRunes = 300
)

// validateAppInfoPatch checks info fields present in a create/update JSON patch.
// Aligns with Console infoForm + publish validateForm + schema (app fields).
// Only keys present in body are validated (sparse update stays sparse).
func validateAppInfoPatch(body map[string]any) error {
	if body == nil {
		return nil
	}
	if _, ok := body["app_name"]; ok {
		if err := validateAppName(stringFromMap(body, "app_name")); err != nil {
			return err
		}
	}
	if _, ok := body["excerpt"]; ok {
		if err := validateAppExcerpt(stringFromMap(body, "excerpt")); err != nil {
			return err
		}
	}
	if _, ok := body["scene"]; ok {
		scene := stringFromMap(body, "scene")
		if scene == "" {
			return fmt.Errorf("scene is required (non-empty id from model constants app_scenarios_array)")
		}
	}
	if _, ok := body["allow_clone"]; ok {
		if err := validateAppAllowFlag(body, "allow_clone"); err != nil {
			return err
		}
	}
	if _, ok := body["allow_commerce"]; ok {
		if err := validateAppAllowFlag(body, "allow_commerce"); err != nil {
			return err
		}
	}
	if _, ok := body["images"]; ok {
		if body["images"] == nil {
			return fmt.Errorf("images is required (non-empty URL array; at least 1, max %d)", MaxAppImageCount)
		}
		imgs := normalizeAppImages(body["images"])
		if len(imgs) == 0 {
			return fmt.Errorf("images must contain at least 1 URL (Console: at least one image; max %d)", MaxAppImageCount)
		}
		if err := validateAppImageCount(imgs); err != nil {
			return err
		}
		for _, u := range imgs {
			if err := validateAppImageURL(u); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateAppExcerpt(excerpt string) error {
	excerpt = strings.TrimSpace(excerpt)
	if excerpt == "" {
		return fmt.Errorf("excerpt is required")
	}
	if utf8.RuneCountInString(excerpt) > maxAppExcerptRunes {
		return fmt.Errorf("excerpt max length is %d", maxAppExcerptRunes)
	}
	return nil
}

func validateAppAllowFlag(body map[string]any, key string) error {
	v := body[key]
	if v == nil {
		return fmt.Errorf("%s is required (0 or 1)", key)
	}
	if b, ok := v.(bool); ok {
		_ = b
		return nil
	}
	n, ok := asInt(v)
	if !ok || (n != 0 && n != 1) {
		return fmt.Errorf("%s must be 0 or 1 (got %v)", key, v)
	}
	return nil
}

func validateAppImageURL(raw string) error {
	return validateHTTPOrHTTPSURL(raw, "images")
}

// validateHTTPOrHTTPSURL rejects empty / non-http(s) URLs (Console uploads always produce https CDN URLs).
func validateHTTPOrHTTPSURL(raw, field string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("%s contains an empty URL", field)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%s contains invalid URL %q (expect http(s) URL)", field, raw)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s contains invalid URL %q (expect http(s) URL)", field, raw)
	}
	return nil
}

// validateAppSceneID checks scene against get_constant app_scenarios_array when available.
func (s AppService) validateAppSceneID(scene string) error {
	scene = strings.TrimSpace(scene)
	if scene == "" {
		return fmt.Errorf("scene is required (non-empty id from model constants app_scenarios_array)")
	}
	ids, err := s.appScenarioIDs()
	if err != nil {
		// Network/constants failure: emptiness already checked; do not block with opaque API errors.
		return nil
	}
	if len(ids) == 0 {
		return nil
	}
	if _, ok := ids[scene]; !ok {
		return fmt.Errorf(
			"scene %q is not a valid app_scenarios_array id; run: sensecraft-cli model constants",
			scene,
		)
	}
	return nil
}

func (s AppService) appScenarioIDs() (map[string]struct{}, error) {
	data, err := s.client.Get(enum.AIApiPathGetConstant, nil)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			data, _ = json.Marshal(map[string]any{
				"app_scenarios_array": []map[string]any{
					{"id": "1", "name": "Smart Camera"},
					{"id": "5", "name": "Other"},
				},
			})
		} else {
			return nil, err
		}
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	arr, _ := root["app_scenarios_array"].([]any)
	out := make(map[string]struct{}, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := stringFromMap(m, "id")
		if id != "" {
			out[id] = struct{}{}
		}
	}
	return out, nil
}
