package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/enum"
)

func watcherFlashBlock() map[string]any {
	return map[string]any{
		"id": "block_flash_watcher", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "113991315", "device_type": "37", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/muse-gadget.bin", "firmware_name": "muse-gadget.bin",
			"firmware_size": "2232320", "flash_address": "0x0", "muse_gadget": true, "full_flash": false,
		},
	}
}

func toolBlockBody(block map[string]any) map[string]any {
	return map[string]any{"deployment": map[string]any{"blocks": []any{block}}}
}

func TestToolBlocksRequireOneConfigEnvelope(t *testing.T) {
	for kind := range appToolConfigKeys {
		t.Run(kind, func(t *testing.T) {
			for _, cfg := range []any{nil, "", []any{}} {
				block := map[string]any{"id": "tool", "type": kind, "config": cfg}
				if err := ValidateDeploymentBlocks(toolBlockBody(block)); err == nil {
					t.Fatalf("accepted invalid config %T", cfg)
				}
			}
			block := map[string]any{"id": "tool", "type": kind, "config": map[string]any{}}
			if err := ValidateDeploymentBlocks(toolBlockBody(block)); err != nil {
				t.Fatalf("a pending tool may have an empty config: %v", err)
			}
			block["config"] = map[string]any{"config": map[string]any{"device_id": "123"}}
			if err := ValidateDeploymentBlocks(toolBlockBody(block)); err == nil {
				t.Fatal("accepted an extra config.config wrapper")
			}
		})
	}
}

func TestFlashFirmwareLiftsFlatFieldsThenDropsThem(t *testing.T) {
	block := watcherFlashBlock()
	if err := ValidateDeploymentBlocks(toolBlockBody(block)); err != nil {
		t.Fatal(err)
	}
	if _, flat := block["firmware_url"]; flat {
		t.Fatal("config-only block gained a root firmware_url")
	}

	const firmwareURL = "https://example.invalid/muse-gadget.bin"
	flat := map[string]any{
		"id": "flat", "type": "flash-firmware",
		"device_id": "113991315", "device_type": "37", "flash_mode": "bin",
		"firmware_url": firmwareURL, "firmware_name": "muse-gadget.bin",
		"firmware_size": "2232320", "flash_address": "0x0", "muse_gadget": true, "full_flash": false,
	}
	if err := ValidateDeploymentBlocks(toolBlockBody(flat)); err != nil {
		t.Fatal(err)
	}
	if _, still := flat["firmware_url"]; still {
		t.Fatal("flat firmware_url was posted on the block")
	}
	lifted := flat["config"].(map[string]any)
	if lifted["firmware_url"] != firmwareURL || lifted["muse_gadget"] != true {
		t.Fatalf("flat settings were not copied into config: %#v", lifted)
	}

	mixed := watcherFlashBlock()
	mixed["firmware_url"] = "https://example.invalid/other.bin"
	if err := ValidateDeploymentBlocks(toolBlockBody(mixed)); err != nil {
		t.Fatal(err)
	}
	if _, still := mixed["firmware_url"]; still {
		t.Fatal("root firmware_url survived")
	}
	if mixed["config"].(map[string]any)["firmware_url"] != firmwareURL {
		t.Fatalf("config lost to a different root value: %#v", mixed["config"])
	}
}

func TestFlashFirmwareChecksConfigValues(t *testing.T) {
	for _, tc := range []struct {
		key   string
		value any
	}{
		{"firmware_size", 2232320}, {"firmware_size", "large"}, {"firmware_url", "./firmware.bin"},
		{"flash_address", "0x20000"}, {"flash_mode", "hex"}, {"device_id", 113991315},
		{"muse_gadget", "false"}, {"full_flash", 1},
	} {
		t.Run(tc.key+"/"+reflect.TypeOf(tc.value).String(), func(t *testing.T) {
			block := watcherFlashBlock()
			block["config"].(map[string]any)[tc.key] = tc.value
			if err := ValidateDeploymentBlocks(toolBlockBody(block)); err == nil {
				t.Fatal("accepted invalid firmware settings")
			}
		})
	}
	coerced := watcherFlashBlock()
	coerced["config"].(map[string]any)["full_flash"] = true
	if err := ValidateDeploymentBlocks(toolBlockBody(coerced)); err != nil {
		t.Fatalf("muse_gadget true should coerce full_flash false: %v", err)
	}
	if coerced["config"].(map[string]any)["full_flash"] != false {
		t.Fatalf("expected full_flash coerced to false: %#v", coerced["config"])
	}
	block := watcherFlashBlock()
	cfg := block["config"].(map[string]any)
	cfg["device_id"] = "113991115"
	cfg["device_type"] = "32"
	cfg["muse_gadget"] = false
	cfg["full_flash"] = false
	cfg["firmware_url"] = "https://example.invalid/app.bin"
	cfg["firmware_name"] = "app.bin"
	cfg["flash_mode"] = "zip"
	cfg["project_zip_url"] = "https://example.invalid/project.zip"
	cfg["project_zip_size"] = "1234"
	if err := ValidateDeploymentBlocks(toolBlockBody(block)); err != nil {
		t.Fatalf("nested ZIP fields should remain supported: %v", err)
	}
}

func TestFlashFirmwareMuseFlagsRequireMuseSKU(t *testing.T) {
	block := watcherFlashBlock()
	cfg := block["config"].(map[string]any)
	cfg["muse_gadget"] = false
	cfg["full_flash"] = true
	cfg["firmware_url"] = "https://example.invalid/merged.bin"
	cfg["firmware_name"] = "merged.bin"
	if err := ValidateDeploymentBlocks(toolBlockBody(block)); err != nil {
		t.Fatalf("full_flash on a Muse SKU should be allowed: %v", err)
	}

	cfg["device_id"] = "113991115"
	cfg["firmware_url"] = "https://example.invalid/app.bin"
	cfg["firmware_name"] = "app.bin"
	cfg["muse_gadget"] = true
	cfg["full_flash"] = false
	if err := ValidateDeploymentBlocks(toolBlockBody(block)); err != nil {
		t.Fatalf("non-Muse muse_gadget should be coerced off: %v", err)
	}
	if block["config"].(map[string]any)["muse_gadget"] != false {
		t.Fatal("non-Muse muse_gadget should become false")
	}
	cfg["muse_gadget"] = false
	cfg["full_flash"] = true
	if err := ValidateDeploymentBlocks(toolBlockBody(block)); err != nil {
		t.Fatalf("non-Muse full_flash should be coerced off: %v", err)
	}
	if block["config"].(map[string]any)["full_flash"] != false {
		t.Fatal("non-Muse full_flash should become false")
	}
}

func TestFlashFirmwareRequiresDeviceIDAndRejectsUnsafeZip(t *testing.T) {
	pending := map[string]any{"id": "flash_pending", "type": "flash-firmware", "config": map[string]any{}}
	if err := ValidateDeploymentBlocks(toolBlockBody(pending)); err != nil {
		t.Fatalf("empty flash config scaffold should remain allowed: %v", err)
	}

	missingSKU := map[string]any{
		"id": "flash_nosku", "type": "flash-firmware",
		"config": map[string]any{
			"flash_mode": "bin", "firmware_url": "https://example.invalid/app.bin",
			"firmware_name": "app.bin", "firmware_size": "1234",
		},
	}
	if err := ValidateDeploymentBlocks(toolBlockBody(missingSKU)); err == nil {
		t.Fatal("accepted flash-firmware with firmware but empty device_id")
	} else if !strings.Contains(err.Error(), "device_id is required") {
		t.Fatalf("expected clear device_id reason, got: %v", err)
	}

	e1001Zip := map[string]any{
		"id": "flash_e1001_zip", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "100073581", "flash_mode": "zip",
			"project_zip_url": "https://example.invalid/project.zip", "project_zip_size": "1234",
			"muse_gadget": false, "full_flash": true,
		},
	}
	if err := ValidateDeploymentBlocks(toolBlockBody(e1001Zip)); err == nil {
		t.Fatal("accepted zip on E1001")
	} else if !strings.Contains(err.Error(), "flash_mode \"zip\" rejected") {
		t.Fatalf("expected clear zip rejection, got: %v", err)
	}

	unknownZip := map[string]any{
		"id": "flash_unknown_zip", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "999999999", "flash_mode": "zip",
			"project_zip_url": "https://example.invalid/project.zip", "project_zip_size": "1234",
		},
	}
	if err := ValidateDeploymentBlocks(toolBlockBody(unknownZip)); err == nil {
		t.Fatal("accepted zip on unknown SKU")
	}
}

func TestFlashFirmwareRejectsMuseFilenameWithoutMuseFlag(t *testing.T) {
	block := map[string]any{
		"id": "flash_muse_name", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "100073581", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/muse-gadget.bin", "firmware_name": "muse-gadget.bin",
			"firmware_size": "524288", "flash_address": "0x90000",
			"muse_gadget": false, "full_flash": true,
		},
	}
	if err := ValidateDeploymentBlocks(toolBlockBody(block)); err == nil {
		t.Fatal("accepted muse-gadget.bin while muse_gadget is false")
	} else if !strings.Contains(err.Error(), "muse_gadget is false") {
		t.Fatalf("expected clear muse filename mismatch reason, got: %v", err)
	}
}

func TestFlashFirmwareRequiresFirmwareSizeWithURL(t *testing.T) {
	block := map[string]any{
		"id": "flash_nosize", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "113991054", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/application.bin", "firmware_name": "application.bin",
			"muse_gadget": false, "full_flash": false,
		},
	}
	if err := ValidateDeploymentBlocks(toolBlockBody(block)); err == nil {
		t.Fatal("accepted firmware_url without firmware_size")
	} else if !strings.Contains(err.Error(), "firmware_size is required") {
		t.Fatalf("expected size required reason, got: %v", err)
	}
}

func TestFlashFirmwareRejectsImplausibleSizes(t *testing.T) {
	tooSmall := map[string]any{
		"id": "flash_small", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "113991315", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/app.bin", "firmware_name": "app.bin",
			"firmware_size": "1234", "flash_address": "0x0",
			"muse_gadget": false, "full_flash": true,
		},
	}
	if err := ValidateDeploymentBlocks(toolBlockBody(tooSmall)); err == nil {
		t.Fatal("accepted tiny bin for Watcher 0x0 merged address")
	} else if !strings.Contains(err.Error(), "too small for flash_address 0x0") {
		t.Fatalf("expected clear small-merge reason, got: %v", err)
	}

	tooLarge := map[string]any{
		"id": "flash_huge", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "100073581", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/merged.bin", "firmware_name": "merged.bin",
			"firmware_size": "8388608", "flash_address": "0x90000",
			"muse_gadget": false, "full_flash": true,
		},
	}
	if err := ValidateDeploymentBlocks(toolBlockBody(tooLarge)); err == nil {
		t.Fatal("accepted 8MiB image for E1001 0x90000")
	} else if !strings.Contains(err.Error(), "too large for flash_address 0x90000") {
		t.Fatalf("expected clear large-app-region reason, got: %v", err)
	}
}

func TestFlashFirmwareDefaultsFullFlashAndAddress(t *testing.T) {
	block := map[string]any{
		"id": "flash_e1002", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "100017057", "device_type": "29", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/app.bin", "firmware_name": "app.bin",
			"firmware_size": "1234", "muse_gadget": false,
		},
	}
	if err := ValidateDeploymentBlocks(toolBlockBody(block)); err != nil {
		t.Fatal(err)
	}
	cfg := block["config"].(map[string]any)
	if cfg["full_flash"] != true {
		t.Fatalf("Muse SKU with muse off should default full_flash true: %#v", cfg)
	}
	if cfg["flash_address"] != "0x90000" {
		t.Fatalf("E1002 should fill 0x90000: %#v", cfg)
	}

	keepOff := map[string]any{
		"id": "flash_keep", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "100017057", "device_type": "29", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/app.bin", "firmware_name": "app.bin",
			"firmware_size": "1234", "flash_address": "0x90000",
			"muse_gadget": false, "full_flash": false,
		},
	}
	if err := ValidateDeploymentBlocks(toolBlockBody(keepOff)); err != nil {
		t.Fatal(err)
	}
	if keepOff["config"].(map[string]any)["full_flash"] != false {
		t.Fatal("explicit full_flash false must be kept")
	}
}

func TestFlashFirmwareRejectsIllegalFixedAddresses(t *testing.T) {
	cases := []struct {
		name, sku, bad, want, size string
	}{
		{"e1002", "100017057", "0x0", "0x90000", "524288"},
		{"wio", "102991299", "0x0", "0x4000", "131072"},
		{"watcher", "113991315", "0x10000", "0x0", "1048576"},
		{"sticky", "100056818", "0x10000", "0x0", "1048576"},
		{"indicator", "114993068", "0x10000", "0x0", "1048576"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			block := map[string]any{
				"id": "flash_reject_addr", "type": "flash-firmware",
				"config": map[string]any{
					"device_id": tc.sku, "device_type": "", "flash_mode": "bin",
					"firmware_url": "https://example.invalid/app.bin", "firmware_name": "app.bin",
					"firmware_size": tc.size, "flash_address": tc.bad,
					"muse_gadget": false, "full_flash": isMuseGadgetSKU(tc.sku),
				},
			}
			err := ValidateDeploymentBlocks(toolBlockBody(block))
			if err == nil {
				t.Fatalf("accepted unsafe flash_address %s for %s (must be %s)", tc.bad, tc.sku, tc.want)
			}
			if !strings.Contains(err.Error(), "unsafe for device_id") || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected fixed-address rejection naming %s, got: %v", tc.want, err)
			}
		})
	}

	muse := watcherFlashBlock()
	muse["config"].(map[string]any)["firmware_url"] = "https://example.invalid/app.bin"
	muse["config"].(map[string]any)["firmware_name"] = "app.bin"
	if err := ValidateDeploymentBlocks(toolBlockBody(muse)); err == nil {
		t.Fatal("accepted muse_gadget with a non muse-gadget.bin")
	}
}

func TestFlashFirmwareGroveAllowsSmallBinAtZero(t *testing.T) {
	block := map[string]any{
		"id": "flash_grove", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "110061162", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/fw.bin", "firmware_name": "fw.bin",
			"firmware_size": "1234", "flash_address": "0x0",
			"muse_gadget": false, "full_flash": false,
		},
	}
	if err := ValidateDeploymentBlocks(toolBlockBody(block)); err != nil {
		t.Fatalf("Grove Beginner must allow small .bin at 0x0 (not an ESP merge): %v", err)
	}
}

func TestFlashFirmwareRejectsApplicationPackageOnWatcherMerged(t *testing.T) {
	// Repro: mid-size app-only bin (≥200KiB) named *application* with 0x0 after
	// an Agent mistakenly treated Watcher like ordinary XIAO.
	block := map[string]any{
		"id": "flash_watcher_app", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "113991315", "device_type": "37", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/photo_viewer-application.bin",
			"firmware_name": "photo_viewer-application.bin",
			"firmware_size": "281376", "flash_address": "0x0",
			"muse_gadget": false, "full_flash": true,
		},
	}
	err := ValidateDeploymentBlocks(toolBlockBody(block))
	if err == nil {
		t.Fatal("accepted Watcher application-only package as merged 0x0 firmware")
	}
	if !strings.Contains(err.Error(), "application-only package") {
		t.Fatalf("expected application-package rejection, got: %v", err)
	}

	// Same package with leftover 0x10000 must also fail (no silent coerce).
	block["config"].(map[string]any)["flash_address"] = "0x10000"
	err = ValidateDeploymentBlocks(toolBlockBody(block))
	if err == nil {
		t.Fatal("accepted Watcher flash_address 0x10000")
	}
	if !strings.Contains(err.Error(), "unsafe for device_id") {
		t.Fatalf("expected fixed-address rejection, got: %v", err)
	}
}

func TestFlashFirmwareApplicationURLFillsXiaoAddress(t *testing.T) {
	block := map[string]any{
		"id": "flash_xiao", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "113991054", "device_type": "32", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/application.bin", "firmware_name": "application.bin",
			"firmware_size": "1234", "muse_gadget": false, "full_flash": false,
		},
	}
	if err := ValidateDeploymentBlocks(toolBlockBody(block)); err != nil {
		t.Fatal(err)
	}
	if block["config"].(map[string]any)["flash_address"] != "0x10000" {
		t.Fatalf("application firmware should fill 0x10000: %#v", block["config"])
	}

	unknown := map[string]any{
		"id": "flash_unknown", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "999999999", "device_type": "", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/application.bin", "firmware_name": "application.bin",
			"firmware_size": "1234", "muse_gadget": false, "full_flash": false,
		},
	}
	if err := ValidateDeploymentBlocks(toolBlockBody(unknown)); err == nil {
		t.Fatal("accepted unknown device_id without materials name")
	} else if !strings.Contains(err.Error(), "not in flash-firmware.device_defaults") {
		t.Fatalf("expected unknown-SKU reason, got: %v", err)
	}
	named := toolBlockBody(map[string]any{
		"id": "flash_unknown_named", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "999999999", "device_type": "", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/application.bin", "firmware_name": "application.bin",
			"firmware_size": "524288", "muse_gadget": false, "full_flash": false,
		},
	})
	named["materials"] = []any{
		map[string]any{"id": "999999999", "name": "XIAO 2.13\" ePaper Display", "quantity": "1"},
	}
	if err := ValidateDeploymentBlocks(named); err != nil {
		t.Fatal(err)
	}
	if addr := named["deployment"].(map[string]any)["blocks"].([]any)[0].(map[string]any)["config"].(map[string]any)["flash_address"]; addr != "0x90000" {
		t.Fatalf("named e-paper unknown SKU should coerce to 0x90000: %#v", addr)
	}
}

func TestFlashFirmwareNameRulesForEpaperAndIPS(t *testing.T) {
	body := toolBlockBody(map[string]any{
		"id": "flash_epaper", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "epaper-sku-1", "device_type": "", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/app.bin", "firmware_name": "app.bin",
			"firmware_size": "1234", "flash_address": "0x0",
			"muse_gadget": false, "full_flash": false,
		},
	})
	body["materials"] = []any{
		map[string]any{"id": "epaper-sku-1", "name": "XIAO 2.13\" ePaper Display", "quantity": "1"},
	}
	if err := ValidateDeploymentBlocks(body); err == nil || !strings.Contains(err.Error(), "0x90000") {
		t.Fatalf("e-paper by name should reject 0x0 (fixed at 0x90000), got: %v", err)
	}

	// Empty address still fills the name-based default.
	fill := toolBlockBody(map[string]any{
		"id": "flash_epaper_fill", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "epaper-sku-1", "device_type": "", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/app.bin", "firmware_name": "app.bin",
			"firmware_size": "1234", "muse_gadget": false, "full_flash": false,
		},
	})
	fill["materials"] = body["materials"]
	if err := ValidateDeploymentBlocks(fill); err != nil {
		t.Fatal(err)
	}
	if fill["deployment"].(map[string]any)["blocks"].([]any)[0].(map[string]any)["config"].(map[string]any)["flash_address"] != "0x90000" {
		t.Fatalf("e-paper by name should fill 0x90000 when address omitted")
	}

	ips := toolBlockBody(map[string]any{
		"id": "flash_ips", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "ips-sku-1", "device_type": "", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/merged.bin", "firmware_name": "merged.bin",
			"firmware_size": "1048576", "flash_address": "0x10000",
			"muse_gadget": false, "full_flash": false,
		},
	})
	ips["materials"] = []any{
		map[string]any{"sku": "ips-sku-1", "name": `XIAO 1.14" IPS Display (ESP32-S3)`, "quantity": "1"},
	}
	if err := ValidateDeploymentBlocks(ips); err == nil || !strings.Contains(err.Error(), "0x0") {
		t.Fatalf("IPS by name should reject 0x10000 (fixed at 0x0), got: %v", err)
	}
}

func TestFlashFirmwareIndicatorAliasAllowsMuse(t *testing.T) {
	block := map[string]any{
		"id": "flash_d1s", "type": "flash-firmware",
		"config": map[string]any{
			"device_id": "114993069", "device_type": "", "flash_mode": "bin",
			"firmware_url": "https://example.invalid/muse-gadget.bin", "firmware_name": "muse-gadget.bin",
			"firmware_size": "1234", "flash_address": "0x0",
			"muse_gadget": true, "full_flash": false,
		},
	}
	if err := ValidateDeploymentBlocks(toolBlockBody(block)); err != nil {
		t.Fatalf("Indicator D1S alias should allow Muse: %v", err)
	}
}

func TestOtherToolConfigRepresentations(t *testing.T) {
	for _, flow := range []any{map[string]any{"nodes": []any{}}, []any{map[string]any{"id": "node-1"}}} {
		block := map[string]any{"id": "flow", "type": "node-red", "config": map[string]any{"flow_data": flow}}
		if err := ValidateDeploymentBlocks(toolBlockBody(block)); err != nil {
			t.Fatal(err)
		}
	}
	bad := map[string]any{"id": "flow", "type": "node-red", "config": map[string]any{"flow_data": "[]"}}
	if err := ValidateDeploymentBlocks(toolBlockBody(bad)); err == nil {
		t.Fatal("accepted a stringified flow inside config")
	}
	aily := map[string]any{"id": "aily", "type": "aily-blockly", "config": map[string]any{"project_id": "123", "config": ""}}
	if err := ValidateDeploymentBlocks(toolBlockBody(aily)); err != nil {
		t.Fatalf("Aily's documented extension string is not a config wrapper: %v", err)
	}
	flat := map[string]any{"id": "model", "type": "deploy-ai-model", "model": map[string]any{}, "config": map[string]any{}}
	if err := ValidateDeploymentBlocks(toolBlockBody(flat)); err == nil {
		t.Fatal("accepted flattened settings on a different tool type")
	}
}

func TestMalformedToolRejectedBeforeCreateOrUpdateWrites(t *testing.T) {
	posts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts++
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"app_id": "12345", "be_public": 0}})
	}))
	defer srv.Close()
	s := appServiceForTest(t, srv)
	block := map[string]any{"id": "tool", "type": "flash-firmware", "config": "flat"}
	body := toolBlockBody(block)
	body["app_name"] = "Watcher"
	encoded, _ := json.Marshal(body)
	path := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateApp(AppCreateParams{FromJSON: path}); err == nil {
		t.Fatal("create accepted a malformed tool")
	}
	if _, err := s.UpdateApp(AppUpdateParams{AppID: "12345", FromJSON: path, Yes: true}); err == nil {
		t.Fatal("update accepted a malformed tool")
	}
	if posts != 0 {
		t.Fatalf("malformed tools must fail before creating an orphan or writing an update; posts=%d", posts)
	}
}

func TestCreateAndUpdatePersistRendererReadableConfig(t *testing.T) {
	var writes []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{"app_id": "12345", "be_public": 0}
		if r.URL.Path == enum.AIApiPathAppUpdate {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			writes = append(writes, body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
	}))
	defer srv.Close()
	s := appServiceForTest(t, srv)
	blocks := []any{
		map[string]any{"id": "intro", "type": "paragraph", "content": "Connect the Watcher."},
		map[string]any{"id": "step", "type": "step", "title": "Flash firmware", "content": "Flash firmware"},
		watcherFlashBlock(),
	}
	body := map[string]any{"deployment": map[string]any{"blocks": blocks}}
	path := filepath.Join(t.TempDir(), "payload.json")
	write := func() {
		t.Helper()
		encoded, _ := json.Marshal(body)
		if err := os.WriteFile(path, encoded, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	body["app_name"] = "Watcher"
	write()
	if _, err := s.CreateApp(AppCreateParams{FromJSON: path}); err != nil {
		t.Fatal(err)
	}
	delete(body, "app_name")
	write()
	if _, err := s.UpdateApp(AppUpdateParams{AppID: "12345", FromJSON: path, Yes: true}); err != nil {
		t.Fatal(err)
	}
	if len(writes) != 2 {
		t.Fatalf("update writes=%d", len(writes))
	}
	for _, sent := range writes {
		saved := sent["deployment"].(map[string]any)["blocks"].([]any)
		if len(saved) != 3 || !reflect.DeepEqual(saved[:2], blocks[:2]) {
			t.Fatal("surrounding document blocks were lost or changed")
		}
		flash := saved[2].(map[string]any)
		cfg := flash["config"].(map[string]any)
		if !reflect.DeepEqual(cfg, blocks[2].(map[string]any)["config"]) {
			t.Fatal("renderer-readable config was changed or discarded")
		}
		if _, flat := flash["firmware_url"]; flat {
			t.Fatal("posted flat firmware fields")
		}
	}
}
