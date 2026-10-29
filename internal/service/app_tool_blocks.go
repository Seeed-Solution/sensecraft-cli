package service

import (
	"fmt"
	"strconv"
	"strings"
)

// These are config-relative paths for modern deployment.blocks entries.
// Legacy deployment[].tools[] flattened fields are a different wire format.
var appToolConfigKeys = map[string][]string{
	"flash-firmware": {
		"device_id", "device_type", "flash_mode",
		"firmware_url", "firmware_name", "firmware_size", "flash_address",
		"project_zip_url", "project_zip_name", "project_zip_size", "precheck_fields",
		"muse_gadget", "full_flash",
	},
	"deploy-ai-model": {
		"model", "model.type", "model.model", "device_type", "powerMode", "conf", "iou", "threshold", "triggerAction", "deploy_tool_entry", "firmware_url",
	},
	"ai-vibration": {"detectionType", "deviceType", "windowSize", "isPower", "gpio", "dStatus"},
	"deploy-container-app": {
		"app_id", "app_key", "app_display_name", "app_icon", "app_description",
		"architecture", "architectures", "version", "version_options", "install_form",
		"merged_service", "user_override_flags", "form_field_values", "raw_by_version", "detail_form_fields",
	},
	"node-red":     {"flow_data_url", "flow_data", "model_id", "model_data"},
	"aily-blockly": {"project_id", "device_id", "config"},
}

func validateAppToolBlock(block map[string]any, kind string, index int) error {
	keys, tool := appToolConfigKeys[kind]
	if !tool {
		return nil
	}
	label := fmt.Sprintf("deployment.blocks[%d] (%s)", index, kind)
	cfg, ok := block["config"].(map[string]any)
	if !ok || cfg == nil {
		return fmt.Errorf("%s requires a config object; put tool settings inside block.config (flat legacy tool fields are not rendered)", label)
	}
	for _, key := range keys {
		root := strings.SplitN(key, ".", 2)[0]
		if root == "config" {
			continue // Aily's config.config is a string, not another tool wrapper.
		}
		if _, flat := block[root]; flat {
			return fmt.Errorf("%s: move %s inside config; do not mix flat settings and config", label, root)
		}
	}
	switch cfg["config"].(type) {
	case map[string]any, []any:
		return fmt.Errorf("%s: remove the extra config.config wrapper; tool settings belong directly in block.config", label)
	}
	if kind == "node-red" {
		if flow := cfg["flow_data"]; flow != nil {
			switch flow.(type) {
			case map[string]any, []any:
			default:
				return fmt.Errorf("%s: config.flow_data must be an object or array, not a JSON string", label)
			}
		}
		if model := cfg["model_data"]; model != nil {
			if _, ok := model.(map[string]any); !ok {
				return fmt.Errorf("%s: config.model_data must be an object or null", label)
			}
		}
	}
	if kind != "flash-firmware" {
		return nil
	}
	for _, key := range []string{"device_id", "device_type", "firmware_name", "project_zip_name", "firmware_size", "project_zip_size", "flash_mode", "flash_address", "firmware_url", "project_zip_url"} {
		if value, present := cfg[key]; present {
			if _, ok := value.(string); !ok {
				return fmt.Errorf("%s: config.%s must be a string (use an empty string for a pending value)", label, key)
			}
		}
	}
	for _, key := range []string{"firmware_url", "project_zip_url"} {
		if value, _ := cfg[key].(string); value != "" {
			if err := validateHTTPOrHTTPSURL(value, label+" config."+key); err != nil {
				// Do not echo private/signed URLs from the payload.
				return fmt.Errorf("%s: config.%s must be an http(s) URL", label, key)
			}
		}
	}
	for _, key := range []string{"firmware_size", "project_zip_size"} {
		if value, _ := cfg[key].(string); value != "" {
			size, err := strconv.ParseUint(value, 10, 64)
			if err != nil || size == 0 {
				return fmt.Errorf("%s: config.%s must be a positive byte count string", label, key)
			}
		}
	}
	if mode, _ := cfg["flash_mode"].(string); mode != "" && mode != "bin" && mode != "zip" {
		return fmt.Errorf("%s: config.flash_mode must be bin or zip", label)
	}
	if address, _ := cfg["flash_address"].(string); address != "" && address != "0x0" && address != "0x4000" && address != "0x10000" && address != "0x90000" {
		return fmt.Errorf("%s: unsupported config.flash_address; use the device default from the tool-block reference", label)
	}
	for _, key := range []string{"muse_gadget", "full_flash"} {
		if value, present := cfg[key]; present {
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("%s: config.%s must be a JSON boolean, not a string or number", label, key)
			}
		}
	}
	if cfg["muse_gadget"] == true && cfg["full_flash"] == true {
		return fmt.Errorf("%s: config.full_flash must be false when config.muse_gadget is true", label)
	}
	return nil
}

// alignFlashFirmwareBlock copies flat flash settings into config, then drops
// those fields from the block. The app site reads block.config only.
// A filled config wins over a different value on the block.
func alignFlashFirmwareBlock(block map[string]any) {
	if block == nil {
		return
	}
	raw, exists := block["config"]
	cfg, isMap := raw.(map[string]any)
	if exists && !isMap {
		return
	}
	if !exists {
		if !flashBlockHasFlatSettings(block) {
			return
		}
		cfg = map[string]any{}
	}
	keys := appToolConfigKeys["flash-firmware"]
	for _, key := range keys {
		root, rootOK := block[key]
		cfgVal, cfgOK := cfg[key]
		if rootOK && (!cfgOK || flashSettingBlank(cfgVal) && !flashSettingBlank(root)) {
			cfg[key] = root
		}
	}
	block["config"] = cfg
	for _, key := range keys {
		delete(block, key)
	}
}

func flashBlockHasFlatSettings(block map[string]any) bool {
	for _, key := range appToolConfigKeys["flash-firmware"] {
		if _, ok := block[key]; ok {
			return true
		}
	}
	return false
}

func flashSettingBlank(v any) bool {
	if v == nil {
		return true
	}
	text, ok := v.(string)
	return ok && strings.TrimSpace(text) == ""
}
