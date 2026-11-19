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
			{"field": "activity_id", "type": "id", "required": false, "notes": "positive integer; omit or blank leaves existing activity tags unchanged. tags is ignored and is not sent"},
			{"field": "cancel_activity_id", "type": "id", "required": false, "notes": "positive integer; leave this activity. omit or blank leaves participation unchanged. does not change allow_clone. must not equal activity_id"},
			{"field": "author", "type": "text", "required": false, "notes": "recommended on save"},
		},
		"materials": map[string]any{
			"seeed_row": []map[string]any{
				{"field": "id", "required": true, "notes": "SKU"},
				{"field": "name", "required": true},
				{"field": "quantity", "required": true, "type": "string", "notes": "integer string from 1 to 999"},
				{"field": "note", "required": true},
				{"field": "buy_url", "required": true},
				{"field": "price", "stored": false, "notes": "catalog value from app bom search; resolved by SKU, not persisted"},
				{"field": "pic_url", "stored": false, "notes": "catalog value from app bom search; resolved by SKU, not persisted"},
			},
			"custom_row": []map[string]any{
				{"field": "name", "required": true},
				{"field": "quantity", "required": true, "type": "string", "notes": "integer string from 1 to 999"},
				{"field": "note", "required": false, "notes": "optional in Console"},
				{"field": "buy_url", "required": false, "notes": "optional purchase link in Console"},
				{"field": "id", "required": false, "notes": "omit for custom rows"},
				{"field": "image", "supported": false, "notes": "no image input for custom BOM rows in Console"},
				{"field": "price", "supported": false, "notes": "no price input; preview counts custom rows as 0"},
			},
		},
		"reference": "skills/sensecraft-cli/references/app-fields.md",
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
				"Compile Markdown with app doc compile --from-md, or pass the file to app create --from-md / app update --from-md. JSON text blocks are filled to editor video, link, and image HTML. Raw Markdown markers in JSON are rejected",
			},
			"text_block_types": []string{"step", "paragraph", "h1", "h2", "h3", "h4", "h5", "h6", "bullet", "number", "todo", "quote", "code", "divider"},
		},
		"tool_blocks": []map[string]any{
			{
				"type":        "flash-firmware",
				"priority":    "high",
				"when":        "Burn firmware image to a supported board",
				"execute":     "L3",
				"config_path": "config",
				"config_keys": appToolConfigKeys["flash-firmware"],
				"example": map[string]any{
					"id": "flash_example", "type": "flash-firmware",
					"config": map[string]any{
						"device_id": "102991299", "device_type": "", "flash_mode": "bin",
						"firmware_url": "https://example.invalid/firmware.bin", "firmware_name": "firmware.bin",
						"firmware_size": "1234", "flash_address": "0x4000", "muse_gadget": false, "full_flash": false,
					},
				},
				"device_defaults": flashFirmwareDeviceDefaults(),
				"notes": "Write flash settings only in block.config. The app site reads config and does not use a second copy on the block. app create and app update copy flat fields into config when config is missing or blank, then remove those fields from the block. A filled config wins when the two copies differ. Other tool blocks also keep settings only in config. Legacy deployment[].tools[] is a different wire format. device_id is a material SKU string, not a production type id. device_type is uniform_type or empty. flash_mode is bin or zip; zip only for ordinary XIAO ESP32 (C3/C6/S3 Sense) — fixed-address boards (E100x/Wio/Watcher/Sticky/Indicator/e-paper/IPS/Grove) must use bin. device_id is required once the block has firmware or flash settings (empty config {} scaffold is still allowed). firmware_size and project_zip_size are strings. flash_address is 0x0, 0x4000, 0x10000, or 0x90000; omit it and CLI fills the device default, or see device_defaults. Each device_defaults row includes build_artifact and build_recipe (artifact/chip/forbid/steps/muse_alt). Agents should run `sensecraft-cli firmware advise --device-id <SKU>` before compiling — 0x0 needs a merged image (bootloader+partitions+app), 0x90000/0x4000/0x10000 need an application-region .bin, Muse needs muse-gadget.bin only. When firmware_size is set, CLI rejects sizes that are too small for a 0x0 merge (<200KiB) or too large for an app-region/Muse bin (>4MiB). Fixed-address SKUs are coerced to the only safe option: E100x 0x90000 (no 0x0), Wio 0x4000 (no 0x0), Watcher/Sticky/Indicator/1.47 0x0 (no 0x10000). Indicator aliases 114993069/114993071 and Watcher 114993275 are Muse-capable. When create/update JSON includes materials[], name rules also force XIAO e-paper/EE DIY to 0x90000 and IPS/Touch ESP to 0x0. Ordinary XIAO (C3/C6/Sense) still allows 0x0 or 0x10000; URL or name containing application uses 0x10000 only for those known SKUs — never for unknown SKUs. zip fields only when flash_mode is zip. muse_gadget and full_flash are JSON booleans only, never strings or numbers. Both flags can be true only for Indicator 114993068, Watcher 113991315 or 113991275, E1001 100073581, E1002 100017057, 1.47-inch touch 100069905, or Sticky 100056818; the CLI rejects true for any other device_id, and the editor resets those devices to false. When muse_gadget is true, set full_flash false (the editor forces it) and firmware_url/firmware_name must be muse-gadget.bin linked at 0x20000; flash_address stays the device default (do not write 0x20000). When muse_gadget is false, firmware_url/firmware_name must not look like muse-gadget.bin (rename or enable Muse). When muse_gadget is false on a Muse SKU and full_flash is omitted, CLI defaults full_flash to true so leftover Muse partitions/bootloader are covered (matches the app editor), except Watcher which forces full_flash false (no erase-all; preserves nvsfactory). Sticky/Indicator/1.47 force full_flash true even when explicit false; E100x keeps explicit false. full_flash true erases the whole flash before the write. With muse_gadget false, E1001/E1002 also rewrite the stock bootloader at 0x0 and partition table at 0x8000 while the app stays at 0x90000; Indicator, 1.47-inch touch, and Sticky non-Muse images are already merged at 0x0, so full_flash only turns on erase-all; Watcher non-Muse is also merged at 0x0 but never erase-all. A Muse image uses its own segments and does not erase-all; do not add the reTerminal stock bootloader to it. Do not write an SDK token, flash_target, flash_address_adapted, or bootloader/partition/OTA/PHY files. Segment layout is documentation only. Running the flash stays L3. Type-4 folder upload returns muse hints for matched device folders.",
			},
			{
				"type":        "deploy-ai-model",
				"priority":    "high",
				"when":        "Vision / audio AI model deploy to SenseCraft devices",
				"execute":     "L3",
				"config_path": "config",
				"config_keys": appToolConfigKeys["deploy-ai-model"],
				"notes":       "Vision and sound share this type. config.deploy_tool_entry is vision or sound. sound excludes reCamera Pro (uniform_type 53).",
			},
			{
				"type":        "ai-vibration",
				"priority":    "high",
				"when":        "Vibration detection flows",
				"execute":     "L3",
				"config_path": "config",
				"config_keys": appToolConfigKeys["ai-vibration"],
			},
			{
				"type":        "deploy-container-app",
				"priority":    "high",
				"when":        "Edge container app deploy UX in docs",
				"execute":     "L3",
				"config_path": "config",
				"config_keys": appToolConfigKeys["deploy-container-app"],
				"notes":       "Fields live in config. version is the selected version string, not app_version. Container install is not a CLI command; use Console when the CLI has no container control",
			},
			{
				"type":        "node-red",
				"priority":    "lower",
				"when":        "Node-RED flow + optional model binding",
				"execute":     "L3",
				"config_path": "config",
				"config_keys": appToolConfigKeys["node-red"],
				"notes":       "config.flow_data is an object or array, and config.model_data is an object or null; neither is a JSON string inside the block. Derive top-level flow_data as a JSON string and copy model_data when projecting the payload.",
			},
			{
				"type":        "aily-blockly",
				"priority":    "lower",
				"when":        "Aily Blockly public project",
				"execute":     "L3",
				"config_path": "config",
				"config_keys": appToolConfigKeys["aily-blockly"],
			},
		},
		"reference": "skills/sensecraft-cli/references/tool-blocks.md",
	})
	return raw
}
