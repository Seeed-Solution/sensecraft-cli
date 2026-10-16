package service

import "encoding/json"

// AppFieldsSchema returns the application info + BOM field schema for Agents.
func AppFieldsSchema() json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"app_info": []map[string]any{
			{"field": "app_name", "type": "text", "required": true, "max_length": 60, "notes": "trim"},
			{"field": "excerpt", "type": "text", "required": true, "max_length": 300, "notes": "trim"},
			{"field": "scene", "type": "option", "required": true, "notes": "string id from app_scenarios_array"},
			{"field": "allow_clone", "type": "option", "required": true, "values": []int{0, 1}},
			{"field": "allow_commerce", "type": "option", "required": true, "values": []int{0, 1}},
			{"field": "images", "type": "image_url_array", "required": true, "formats": []string{"JPG", "GIF", "PNG", "WebP"}, "max_count": 16, "max_bytes_per_image": 5242880, "notes": "first image is cover; prefer 4:3; Console infoForm maxImageCount=16 / max-size=5MB"},
			{"field": "license", "type": "text", "required": false},
			{"field": "be_public", "type": "lifecycle", "values": map[string]string{"0": "private", "1": "public(ops)", "2": "pending_review"}, "notes": "Agent submit review uses privacy=2; never send 1"},
			{"field": "app_id", "type": "id", "required_on_update": true},
			{"field": "author", "type": "text", "required": false, "notes": "recommended on save"},
		},
		"materials": map[string]any{
			"seeed_row": []map[string]any{
				{"field": "id", "required": true, "notes": "SKU"},
				{"field": "name", "required": true},
				{"field": "quantity", "required": true, "type": "string"},
				{"field": "note", "required": true},
				{"field": "buy_url", "required": true},
				{"field": "price", "stored": false, "notes": "catalog value from app bom search; resolved by SKU, not persisted"},
				{"field": "pic_url", "stored": false, "notes": "catalog value from app bom search; resolved by SKU, not persisted"},
			},
			"custom_row": []map[string]any{
				{"field": "name", "required": true},
				{"field": "quantity", "required": true, "type": "string"},
				{"field": "note", "required": false, "notes": "optional in Console"},
				{"field": "buy_url", "required": false, "notes": "optional purchase link in Console"},
				{"field": "id", "required": false, "notes": "omit for custom rows"},
				{"field": "image", "supported": false, "notes": "no image input for custom BOM rows in Console"},
				{"field": "price", "supported": false, "notes": "no price input; preview counts custom rows as 0"},
			},
		},
		"reference": "skills/sensecraft-apps/reference-app-fields.md",
	})
	return raw
}

// AppBlocksCatalog returns deploy doc structure + tool-block catalog for Agents.
func AppBlocksCatalog() json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"layout": map[string]any{
			"field": "deployment.blocks",
			"rules": []string{
				"Flat ordered array",
				"Blocks before first type=step are Introduction",
				"Each step runs until the next step; text and tool-blocks may mix",
			},
			"text_block_types": []string{"step", "paragraph", "h1", "h2", "h3", "h4", "h5", "h6", "bullet", "number", "todo", "quote", "code", "divider"},
		},
		"tool_blocks": []map[string]any{
			{
				"type":        "flash-firmware",
				"priority":    "high",
				"when":        "Burn firmware image to a supported board",
				"execute":     "L3",
				"config_keys": []string{"device_id", "device_type", "firmware_url", "firmware_size"},
			},
			{
				"type":     "deploy-ai-model",
				"priority": "high",
				"when":     "Vision / audio AI model deploy to SenseCraft devices",
				"execute":  "L3",
				"config_keys": []string{
					"model", "model.type", "model.model", "device_type", "powerMode", "conf", "iou", "threshold", "triggerAction",
				},
			},
			{
				"type":        "ai-vibration",
				"priority":    "high",
				"when":        "Vibration detection flows",
				"execute":     "L3",
				"config_keys": []string{"detectionType", "deviceType", "windowSize", "isPower", "gpio", "dStatus"},
			},
			{
				"type":        "deploy-container-app",
				"priority":    "high",
				"when":        "Edge container app deploy UX in docs (not App Plaza clone)",
				"execute":     "L3",
				"config_keys": []string{},
				"notes":       "Container install is not App Plaza clone; use Console when the CLI has no container control",
			},
			{
				"type":        "node-red",
				"priority":    "lower",
				"when":        "Node-RED flow + optional model binding",
				"execute":     "L3",
				"config_keys": []string{"flow_data_url", "flow_data", "model_id", "model_data"},
			},
			{
				"type":        "aily-blockly",
				"priority":    "lower",
				"when":        "Aily Blockly public project",
				"execute":     "L3",
				"config_keys": []string{"project_id", "device_id", "config"},
			},
		},
		"reference": "skills/sensecraft-apps/reference-tool-blocks.md",
	})
	return raw
}
