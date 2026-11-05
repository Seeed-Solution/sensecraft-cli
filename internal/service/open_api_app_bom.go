package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/enum"
)

type AppBOMSearchParams struct {
	Name string
}

func (s AppService) SearchBOMProducts(p AppBOMSearchParams) (json.RawMessage, error) {
	data, err := s.client.Get(enum.AIApiPathGetProductionOptions, nil)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			return sandboxBOMCatalog(p.Name), nil
		}
		return nil, err
	}
	filtered, err := filterMaterialOptions(data, p.Name)
	if err != nil {
		return nil, err
	}
	return filtered, nil
}

func (s AppService) ListBOM(appID string, sketch bool) (json.RawMessage, error) {
	view, err := s.ViewApp(AppViewParams{AppID: appID, ViewSketch: sketch})
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(view, &m); err != nil {
		return nil, err
	}
	materials, ok := m["materials"]
	if !ok || materials == nil {
		materials = []any{}
	}
	raw, _ := json.Marshal(map[string]any{
		"app_id":    appID,
		"materials": materials,
	})
	return raw, nil
}

type AppBOMSetParams struct {
	AppID    string
	FromJSON string
	Yes      bool
}

func (s AppService) SetBOM(p AppBOMSetParams) (json.RawMessage, error) {
	if err := requireYes(p.Yes); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.AppID) == "" {
		return nil, fmt.Errorf("--id is required")
	}
	view, err := s.ViewApp(AppViewParams{AppID: p.AppID, ViewSketch: true})
	if err != nil {
		return nil, err
	}
	existing, err := rawJSONObject(view)
	if err != nil {
		return nil, err
	}
	if err := s.rejectUnownedAppWrite(p.AppID, existing); err != nil {
		return nil, err
	}
	if appMapBePublic(existing) == 2 {
		return nil, ErrAppLockedUnderReview
	}
	currentPublic := appMapBePublic(existing)
	body, err := LoadJSONObject(p.FromJSON)
	if err != nil {
		return nil, err
	}
	materials, ok := body["materials"]
	if !ok {
		return nil, fmt.Errorf("--from-json must be an object with materials[] (Seeed rows need id=SKU; custom omit id)")
	}
	updateBody := map[string]any{
		"app_id":    p.AppID,
		"materials": materials,
	}
	data, err := s.client.Post(enum.AIApiPathAppUpdate, updateBody)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			data, _ = json.Marshal(map[string]any{
				"app_id":    p.AppID,
				"materials": body["materials"],
				"updated":   true,
			})
		} else {
			return nil, err
		}
	}
	if currentPublic == 1 {
		data = attachAppJSONMeta(data, map[string]any{
			"be_public":    1,
			"sketch_saved": true,
			"hint":         AppSketchSaveHint,
		})
	}
	return data, nil
}

func filterMaterialOptions(raw json.RawMessage, keyword string) (json.RawMessage, error) {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	opts, _ := root["material_options"].([]any)
	tokens := normalizedProductTokens(keyword)
	out := make([]any, 0)
	for _, item := range opts {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		fields := make([]string, 0, 5)
		for _, key := range []string{"name", "sku", "classify", "note", "description"} {
			fields = append(fields, strings.Join(normalizedProductTokens(stringFromMap(m, key)), ""))
		}
		matches := true
		for _, token := range tokens {
			found := false
			for _, field := range fields {
				if strings.Contains(field, token) {
					found = true
					break
				}
			}
			if !found {
				matches = false
				break
			}
		}
		if matches {
			if support, exists := m["block_support"]; exists {
				m["block_supported"] = support == "\u662f"
			}
			out = append(out, m)
		}
	}
	return json.Marshal(map[string]any{
		"list":  out,
		"count": len(out),
	})
}

// normalizedProductTokens splits on whitespace and removes punctuation from each token.
// normalizedProductTokens 按空白分词并移除各词中的标点。
func normalizedProductTokens(value string) []string {
	tokens := make([]string, 0)
	for _, word := range strings.Fields(strings.ToLower(value)) {
		word = strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		}, word)
		if word != "" {
			tokens = append(tokens, word)
		}
	}
	return tokens
}

func sandboxBOMCatalog(keyword string) json.RawMessage {
	items := []map[string]any{
		{
			"sku":      "113991115",
			"name":     "Seeed Studio XIAO ESP32S3 Sense",
			"classify": "XIAO",
			"price":    "15.99",
			"pic_url":  "https://media-cdn.seeedstudio.com/sandbox/xiao-esp32s3-sense.png",
			"buy_url":  "https://www.seeedstudio.com/",
			"note":     "Sandbox catalog item",
		},
		{
			"sku":      "110061361",
			"name":     "reCamera 2002",
			"classify": "reCamera",
			"price":    "99.00",
			"pic_url":  "https://media-cdn.seeedstudio.com/sandbox/recamera-2002.png",
			"buy_url":  "https://www.seeedstudio.com/",
		},
	}
	raw, _ := json.Marshal(map[string]any{"material_options": items})
	filtered, _ := filterMaterialOptions(raw, keyword)
	return filtered
}
