package service

import (
	"fmt"
	"path"
	"regexp"
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

// museGadgetSKUs matches tool-blocks.md plus known same-row aliases that App
// keeps via production-type grouping (Indicator D1S/D1Pro, Watcher enclosure).
var museGadgetSKUs = map[string]struct{}{
	"114993068": {}, // Indicator D1
	"114993069": {}, // Indicator D1S
	"114993071": {}, // Indicator D1Pro
	"113991315": {}, // Watcher
	"113991275": {}, // Watcher
	"114993275": {}, // Watcher white enclosure (same production type)
	"100073581": {}, // E1001
	"100017057": {}, // E1002
	"100069905": {}, // XIAO 1.47" Touch
	"100056818": {}, // Sticky
}

func isMuseGadgetSKU(deviceID string) bool {
	_, ok := museGadgetSKUs[strings.TrimSpace(deviceID)]
	return ok
}

// isMuseGadgetDevice mirrors App isMuseGadgetFlashDevice when materials are
// available: known SKUs, plus Indicator / Watcher family by product name.
func isMuseGadgetDevice(deviceID string, materials []map[string]any) bool {
	if isMuseGadgetSKU(deviceID) {
		return true
	}
	name := strings.ToLower(materialNameForSKU(materials, deviceID))
	if name == "" {
		return false
	}
	if strings.Contains(name, "sensecap") && strings.Contains(name, "indicator") {
		return true
	}
	if strings.Contains(name, "watcher") && !strings.Contains(name, "xiao") {
		return true
	}
	return false
}

// flashAddressPolicy returns the device default flash_address.
// fixed means any other address is coerced to the default (matches App
// editor: E100x/Wio/Watcher/Sticky/Indicator/1.47 only keep one safe option).
// Ordinary XIAO keeps 0x0 / 0x10000, so fixed is false.
type flashAddressPolicy struct {
	defaultAddress string
	fixed          bool
	fileHint       string
}

var flashAddressBySKU = map[string]flashAddressPolicy{
	"114993068": {defaultAddress: "0x0", fixed: true, fileHint: "merged ESP32-S3 .bin from 0x0 only; do not write 0x10000"},
	"114993069": {defaultAddress: "0x0", fixed: true, fileHint: "merged ESP32-S3 .bin from 0x0 only; do not write 0x10000"},
	"114993071": {defaultAddress: "0x0", fixed: true, fileHint: "merged ESP32-S3 .bin from 0x0 only; do not write 0x10000"},
	"113991315": {defaultAddress: "0x0", fixed: true, fileHint: "merged .bin from 0x0 on Xiao/ESP32-S3 port; do not write 0x10000"},
	"113991275": {defaultAddress: "0x0", fixed: true, fileHint: "merged .bin from 0x0 on Xiao/ESP32-S3 port; do not write 0x10000"},
	"114993275": {defaultAddress: "0x0", fixed: true, fileHint: "merged .bin from 0x0 on Xiao/ESP32-S3 port; do not write 0x10000"},
	"100073581": {defaultAddress: "0x90000", fixed: true, fileHint: "application-region .bin only; do not write 0x0 (restore bootloader via full_flash)"},
	"100017057": {defaultAddress: "0x90000", fixed: true, fileHint: "application-region .bin only; do not write 0x0 (restore bootloader via full_flash)"},
	"100069905": {defaultAddress: "0x0", fixed: true, fileHint: "merged .bin from 0x0 only; do not write 0x10000"},
	"100056818": {defaultAddress: "0x0", fixed: true, fileHint: "merged .bin from 0x0 only; do not write 0x10000"},
	"102991299": {defaultAddress: "0x4000", fixed: true, fileHint: "application .bin at 0x4000 only; do not write 0x0 (bootloader via UF2/J-Link)"},
	"110061162": {defaultAddress: "0x0", fixed: true, fileHint: ".bin from 0x0 or .hex with in-file addresses"},
	"113991054": {defaultAddress: "0x0", fixed: false, fileHint: "URL with application → 0x10000, otherwise 0x0"},
	"113991115": {defaultAddress: "0x0", fixed: false, fileHint: "URL with application → 0x10000, otherwise 0x0"},
	"113991254": {defaultAddress: "0x0", fixed: false, fileHint: "URL with application → 0x10000, otherwise 0x0"}, // XIAO ESP32C6
}

var xiaoEeDiyKitRE = regexp.MustCompile(`(?i)\bee0[2-5]\b`)

func lookupFlashAddressPolicy(deviceID string) (flashAddressPolicy, bool) {
	p, ok := flashAddressBySKU[strings.TrimSpace(deviceID)]
	return p, ok
}

func materialNameForSKU(materials []map[string]any, deviceID string) string {
	id := strings.TrimSpace(deviceID)
	if id == "" || len(materials) == 0 {
		return ""
	}
	for _, row := range materials {
		if row == nil {
			continue
		}
		for _, key := range []string{"sku", "id", "device_id"} {
			if strings.TrimSpace(stringFromMap(row, key)) == id {
				return strings.TrimSpace(stringFromMap(row, "name"))
			}
		}
	}
	return ""
}

// flashPolicyFromMaterialName mirrors App hmiEspFlashProfile name rules when
// the SKU is not in the hard table (e-paper / IPS / Indicator aliases).
func flashPolicyFromMaterialName(name string) (flashAddressPolicy, bool) {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return flashAddressPolicy{}, false
	}
	if strings.Contains(n, "sticky") {
		return flashAddressPolicy{defaultAddress: "0x0", fixed: true, fileHint: "merged .bin from 0x0 only; do not write 0x10000"}, true
	}
	if strings.Contains(n, "sensecap") && strings.Contains(n, "indicator") {
		return flashAddressPolicy{defaultAddress: "0x0", fixed: true, fileHint: "merged ESP32-S3 .bin from 0x0 only; do not write 0x10000"}, true
	}
	if strings.Contains(n, "watcher") && !strings.Contains(n, "xiao") {
		return flashAddressPolicy{defaultAddress: "0x0", fixed: true, fileHint: "merged .bin from 0x0 only; do not write 0x10000"}, true
	}
	if strings.Contains(n, "wio") && strings.Contains(n, "terminal") {
		return flashAddressPolicy{defaultAddress: "0x4000", fixed: true, fileHint: "application .bin at 0x4000 only; do not write 0x0"}, true
	}
	if strings.Contains(n, "reterminal") && !strings.Contains(n, "sticky") {
		return flashAddressPolicy{defaultAddress: "0x90000", fixed: true, fileHint: "application-region .bin only; do not write 0x0"}, true
	}
	if strings.Contains(n, "xiao") {
		if strings.Contains(n, "nrf") || strings.Contains(n, "52840") {
			return flashAddressPolicy{}, false
		}
		isIps := strings.Contains(n, "ips")
		isTouchEsp := strings.Contains(n, "touch") && strings.Contains(n, "display") && strings.Contains(n, "esp32")
		if isIps || isTouchEsp {
			return flashAddressPolicy{defaultAddress: "0x0", fixed: true, fileHint: "merged .bin from 0x0 only; do not write 0x10000"}, true
		}
		if strings.Contains(n, "sense") || strings.Contains(n, "watcher") || strings.Contains(n, "trmnl") {
			return flashAddressPolicy{}, false
		}
		if strings.Contains(n, "round") {
			return flashAddressPolicy{}, false
		}
		isDisplay := strings.Contains(n, "display")
		isEeDiy := xiaoEeDiyKitRE.MatchString(n)
		isEpaperDiy := strings.Contains(n, "epaper") && strings.Contains(n, "diy")
		if isDisplay || isEeDiy || isEpaperDiy {
			return flashAddressPolicy{defaultAddress: "0x90000", fixed: true, fileHint: "application-region .bin only (XIAO e-paper / EE DIY); do not write 0x0"}, true
		}
	}
	return flashAddressPolicy{}, false
}

func resolveFlashAddressPolicy(deviceID string, materials []map[string]any) (flashAddressPolicy, bool) {
	if p, ok := lookupFlashAddressPolicy(deviceID); ok {
		return p, true
	}
	return flashPolicyFromMaterialName(materialNameForSKU(materials, deviceID))
}

func flashMaterialsFromBody(body map[string]any) []map[string]any {
	if body == nil {
		return nil
	}
	var out []map[string]any
	appendRows := func(raw any) {
		list, ok := raw.([]any)
		if !ok {
			return
		}
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
	}
	appendRows(body["materials"])
	if bom, ok := body["bom"].(map[string]any); ok {
		appendRows(bom["materials"])
	}
	return out
}

func isSupportedFlashAddress(addr string) bool {
	switch strings.TrimSpace(addr) {
	case "0x0", "0x4000", "0x10000", "0x90000":
		return true
	default:
		return false
	}
}

// flashBuildArtifactHint tells Agents what binary to compile for this address.
func flashBuildArtifactHint(address string, museCapable bool) string {
	switch address {
	case "0x0":
		hint := "merged firmware from 0x0 (bootloader+partitions+app in one .bin)"
		if museCapable {
			hint += "; or Muse SDK muse-gadget.bin when muse_gadget is true"
		}
		return hint
	case "0x90000":
		hint := "application-region .bin only for 0x90000 (no bootloader in the file)"
		if museCapable {
			hint += "; or Muse SDK muse-gadget.bin when muse_gadget is true"
		}
		return hint
	case "0x4000":
		return "application-region .bin for 0x4000 (Wio); do not overwrite UF2 bootloader with a 0x0 image"
	case "0x10000":
		return "application package .bin (often *application*); use 0x0 only when shipping a merged image"
	default:
		return "follow tool-blocks.md for this device"
	}
}

func looksLikeApplicationFirmware(firmwareURL, firmwareName string) bool {
	combined := strings.ToLower(firmwareURL + " " + firmwareName + " " + path.Base(strings.Split(firmwareURL, "?")[0]))
	return strings.Contains(combined, "application")
}

// flashFirmwareConfigHasPayload is true when the block is more than an empty
// scaffold and therefore needs a concrete device_id before create/update.
func flashFirmwareConfigHasPayload(cfg map[string]any) bool {
	if cfg == nil {
		return false
	}
	for _, key := range []string{
		"firmware_url", "firmware_name", "firmware_size", "flash_address", "flash_mode",
		"project_zip_url", "project_zip_name", "project_zip_size", "device_type",
	} {
		if strings.TrimSpace(stringFromMap(cfg, key)) != "" {
			return true
		}
	}
	if _, ok := cfg["muse_gadget"]; ok {
		return true
	}
	if _, ok := cfg["full_flash"]; ok {
		return true
	}
	if _, ok := cfg["precheck_fields"]; ok {
		return true
	}
	return false
}

// flashZipAllowed reports whether flash_mode=zip is valid for this board.
// Only ordinary XIAO ESP32 rows (non-fixed address policy) accept zip.
func flashZipAllowed(deviceID string, materials []map[string]any) (bool, string) {
	id := strings.TrimSpace(deviceID)
	if id == "" {
		return false, "set config.device_id to an ordinary XIAO ESP32 SKU (for example 113991054 C3, 113991254 C6, 113991115 S3 Sense) before using flash_mode zip"
	}
	if policy, ok := resolveFlashAddressPolicy(id, materials); ok {
		if policy.fixed {
			return false, fmt.Sprintf(
				"device_id %s only accepts a single .bin (%s) at flash_address %s; set flash_mode to \"bin\". Zip is only for ordinary XIAO ESP32 (C3/C6/S3 Sense, …)",
				id, policy.fileHint, policy.defaultAddress,
			)
		}
		return true, ""
	}
	return false, fmt.Sprintf(
		"device_id %q is not a known ordinary XIAO zip target. Use flash_mode \"bin\", or set device_id to a listed XIAO SKU from sensecraft-cli app blocks → flash-firmware.device_defaults (zip examples: 113991054, 113991254, 113991115)",
		id,
	)
}

func looksLikeMuseGadgetFirmware(firmwareURL, firmwareName string) bool {
	combined := strings.ToLower(firmwareURL + " " + firmwareName + " " + path.Base(strings.Split(firmwareURL, "?")[0]))
	return strings.Contains(combined, "muse-gadget")
}

// normalizeFlashFirmwareConfig mirrors App editor defaults for Muse / address.
// It only fills missing values and forces Muse↔full_flash mutual exclusion.
// materials (BOM / create payload) unlock name-based e-paper / IPS policies.
func normalizeFlashFirmwareConfig(cfg map[string]any, materials []map[string]any) {
	if cfg == nil {
		return
	}
	deviceID := strings.TrimSpace(stringFromMap(cfg, "device_id"))
	museGadget, museOK := cfg["muse_gadget"].(bool)
	if !museOK {
		museGadget = false
	}
	museDevice := isMuseGadgetDevice(deviceID, materials)
	if museGadget && !museDevice {
		cfg["muse_gadget"] = false
		museGadget = false
	}
	if museGadget {
		if v, present := cfg["full_flash"]; !present {
			cfg["full_flash"] = false
		} else if _, ok := v.(bool); ok {
			cfg["full_flash"] = false
		}
		// Non-bool values stay for validateAppToolBlock to reject.
	} else if museDevice {
		if _, present := cfg["full_flash"]; !present {
			// App opens Muse-capable boards with full erase when Muse is off,
			// so leftover Muse partitions/bootloader are covered.
			cfg["full_flash"] = true
		}
	} else if v, ok := cfg["full_flash"].(bool); ok && v {
		cfg["full_flash"] = false
	}

	addr := strings.TrimSpace(stringFromMap(cfg, "flash_address"))
	if policy, ok := resolveFlashAddressPolicy(deviceID, materials); ok {
		switch {
		case addr == "":
			next := policy.defaultAddress
			if !policy.fixed && looksLikeApplicationFirmware(stringFromMap(cfg, "firmware_url"), stringFromMap(cfg, "firmware_name")) {
				next = "0x10000"
			}
			cfg["flash_address"] = next
		case policy.fixed && addr != policy.defaultAddress && isSupportedFlashAddress(addr):
			// Match App: coerce leftover UI options that are now unsafe for
			// this board (Watcher 0x10000, E100x/Wio/e-paper 0x0, IPS 0x10000).
			cfg["flash_address"] = policy.defaultAddress
		}
		return
	}
	// Do not apply XIAO application→0x10000 to unknown SKUs (would mis-address
	// e-paper / IPS when BOM names are missing).
}

func validateAppToolBlock(block map[string]any, kind string, index int, materials []map[string]any) error {
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
				return fmt.Errorf("%s: config.%s must be a positive byte count string (decimal bytes, no units). Copy the `size` field from `file upload` JSON — do not invent a value", label, key)
			}
		}
	}
	if modeHint, _ := cfg["flash_mode"].(string); modeHint != "zip" {
		if url := strings.TrimSpace(stringFromMap(cfg, "firmware_url")); url != "" {
			if strings.TrimSpace(stringFromMap(cfg, "firmware_size")) == "" {
				return fmt.Errorf("%s: config.firmware_size is required when config.firmware_url is set — copy the numeric `size` from the type-4 `file upload` response so CLI can check merged vs app-region plausibility", label)
			}
		}
	} else if zipURL := strings.TrimSpace(stringFromMap(cfg, "project_zip_url")); zipURL != "" {
		if strings.TrimSpace(stringFromMap(cfg, "project_zip_size")) == "" {
			return fmt.Errorf("%s: config.project_zip_size is required when config.project_zip_url is set — copy the numeric `size` from the upload response", label)
		}
	}
	mode, _ := cfg["flash_mode"].(string)
	if mode != "" && mode != "bin" && mode != "zip" {
		return fmt.Errorf("%s: config.flash_mode must be \"bin\" or \"zip\" (got %q); .hex is only selected inside Grove Beginner bin mode when the file itself is .hex", label, mode)
	}
	address, _ := cfg["flash_address"].(string)
	if address != "" && address != "0x0" && address != "0x4000" && address != "0x10000" && address != "0x90000" {
		return fmt.Errorf("%s: unsupported config.flash_address %q — allowed values are 0x0, 0x4000, 0x10000, 0x90000 (never 0x20000; Muse app offset is applied by the site). See sensecraft-cli app blocks → flash-firmware.device_defaults", label, address)
	}
	deviceID := strings.TrimSpace(stringFromMap(cfg, "device_id"))
	if deviceID == "" && flashFirmwareConfigHasPayload(cfg) {
		return fmt.Errorf("%s: config.device_id is required before create/update once flash-firmware has firmware or flash settings. Resolve the material SKU with `sensecraft-cli app bom search` or `app blocks` → flash-firmware.device_defaults (examples: Watcher 113991315, E1001 100073581, XIAO C3 113991054). An empty config object {} is still allowed as a pending scaffold", label)
	}
	if deviceID != "" && flashFirmwareConfigHasPayload(cfg) {
		if _, knownSKU := lookupFlashAddressPolicy(deviceID); !knownSKU {
			if _, named := flashPolicyFromMaterialName(materialNameForSKU(materials, deviceID)); !named {
				return fmt.Errorf("%s: config.device_id %q is not in flash-firmware.device_defaults and no matching materials[].name was provided for that id. Unknown SKUs cannot safely choose 0x0 vs 0x10000 vs 0x90000 (e-paper/IPS depend on the catalog name). Fix by: (1) use a listed SKU from `sensecraft-cli app blocks`, or (2) add a BOM materials row with id/sku=%s and the exact product name from `app bom search`, then retry", label, deviceID, deviceID)
			}
		}
	}
	if mode == "zip" {
		if ok, reason := flashZipAllowed(deviceID, materials); !ok {
			return fmt.Errorf("%s: config.flash_mode \"zip\" rejected — %s", label, reason)
		}
	}
	if policy, ok := resolveFlashAddressPolicy(deviceID, materials); ok && policy.fixed && address != "" && address != policy.defaultAddress {
		return fmt.Errorf("%s: config.flash_address %q is unsafe for device_id %s — this board is fixed at %s (%s). Omit flash_address to let CLI fill the default, or set it to %s", label, address, deviceID, policy.defaultAddress, policy.fileHint, policy.defaultAddress)
	}
	for _, key := range []string{"muse_gadget", "full_flash"} {
		if value, present := cfg[key]; present {
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("%s: config.%s must be a JSON boolean true/false, not a string or number (got %T). String \"false\" can be treated as truthy in some readers", label, key, value)
			}
		}
	}
	if cfg["muse_gadget"] == true && cfg["full_flash"] == true {
		return fmt.Errorf("%s: config.full_flash must be false when config.muse_gadget is true — Muse uses a multi-segment layout without erase-all; the editor hides full_flash while Muse is on", label)
	}
	if (cfg["muse_gadget"] == true || cfg["full_flash"] == true) && !isMuseGadgetDevice(deviceID, materials) {
		return fmt.Errorf("%s: config.muse_gadget/full_flash can be true only for Muse-capable SKUs (Indicator/Watcher/E1001/E1002/1.47 Touch/Sticky). device_id %q is not in that set — set both flags false, or change device_id after checking app blocks → flash-firmware.device_defaults", label, deviceID)
	}
	name := stringFromMap(cfg, "firmware_name")
	url := stringFromMap(cfg, "firmware_url")
	if cfg["muse_gadget"] == true {
		if (name != "" || url != "") && !looksLikeMuseGadgetFirmware(url, name) {
			return fmt.Errorf("%s: muse_gadget is true but firmware_url/firmware_name do not look like muse-gadget.bin (firmware_name=%q). Upload Muse SDK output as muse-gadget.bin only; the app site writes it at 0x20000 and keeps flash_address as the device default (do not write 0x20000). Or set muse_gadget false for a normal merged/app-region image", label, name)
		}
	} else if (name != "" || url != "") && looksLikeMuseGadgetFirmware(url, name) {
		return fmt.Errorf("%s: firmware_url/firmware_name look like muse-gadget.bin but muse_gadget is false (firmware_name=%q). Either set muse_gadget true and full_flash false on a Muse-capable SKU, or upload a normal board image (merged at 0x0 / app-region at 0x90000|0x4000|0x10000) and rename so the file does not contain \"muse-gadget\"", label, name)
	}
	if err := validateFlashFirmwareShape(label, cfg, materials); err != nil {
		return err
	}
	return nil
}

// Size heuristics when only firmware_size is available (create/update has no
// local bytes). Conservative gates to catch the common AI mistake of shipping
// an app-only bin as a 0x0 merge, or a whole-flash image as an app-region bin.
const (
	flashMinMergedFirmwareBytes   = 200 * 1024       // bootloader+table+app is rarely smaller
	flashMaxAppRegionFirmwareBytes = 4 * 1024 * 1024 // >4MiB usually means a full-chip dump/merge
)

func validateFlashFirmwareShape(label string, cfg map[string]any, materials []map[string]any) error {
	if cfg == nil {
		return nil
	}
	mode := strings.TrimSpace(stringFromMap(cfg, "flash_mode"))
	if mode == "zip" {
		return nil
	}
	sizeStr := strings.TrimSpace(stringFromMap(cfg, "firmware_size"))
	if sizeStr == "" {
		return nil
	}
	size, err := strconv.ParseUint(sizeStr, 10, 64)
	if err != nil || size == 0 {
		return nil
	}
	deviceID := strings.TrimSpace(stringFromMap(cfg, "device_id"))
	addr := strings.TrimSpace(stringFromMap(cfg, "flash_address"))
	policy, hasPolicy := resolveFlashAddressPolicy(deviceID, materials)
	if addr == "" && hasPolicy {
		addr = policy.defaultAddress
	}
	name := stringFromMap(cfg, "firmware_name")
	if name == "" {
		name = "(unnamed)"
	}

	if cfg["muse_gadget"] == true {
		if size > flashMaxAppRegionFirmwareBytes {
			return fmt.Errorf("%s: firmware_size %s bytes (%s) is too large for Muse muse-gadget.bin (app segment at 0x20000). A file this big usually looks like a whole-flash/merged image — rebuild/export the Muse app binary only, or set muse_gadget false and use the board's merged/app-region recipe from device_defaults.build_artifact", label, sizeStr, name)
		}
		return nil
	}

	switch addr {
	case "0x0":
		if size < flashMinMergedFirmwareBytes {
			return fmt.Errorf("%s: firmware_size %s bytes (%s) is too small for flash_address 0x0 on device_id %s — 0x0 expects a merged image (bootloader+partitions+app), typically ≥ %d bytes. Rebuild with the board merge recipe (see device_defaults.build_artifact), or if this is an application-only package on ordinary XIAO use flash_address 0x10000 and a name/URL containing \"application\"", label, sizeStr, name, deviceID, flashMinMergedFirmwareBytes)
		}
	case "0x90000", "0x4000", "0x10000":
		if size > flashMaxAppRegionFirmwareBytes {
			hint := "application-region .bin only (no bootloader in the file)"
			if hasPolicy {
				hint = policy.fileHint
			}
			return fmt.Errorf("%s: firmware_size %s bytes (%s) is too large for flash_address %s on device_id %s — that offset expects an %s, typically ≤ %d bytes. A file this big usually looks like a 0x0 merged/whole-flash image; rebuild the app-region artifact or switch the board/address to match a merged image at 0x0", label, sizeStr, name, addr, deviceID, hint, flashMaxAppRegionFirmwareBytes)
		}
	}
	return nil
}

// alignFlashFirmwareBlock copies flat flash settings into config, then drops
// those fields from the block. The app site reads block.config only.
// A filled config wins over a different value on the block.
func alignFlashFirmwareBlock(block map[string]any, materials []map[string]any) {
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
	normalizeFlashFirmwareConfig(cfg, materials)
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
