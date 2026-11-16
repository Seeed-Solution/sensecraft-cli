package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFlashFirmwareCatalogKeys(t *testing.T) {
	var catalog struct {
		ToolBlocks []struct {
			Type       string         `json:"type"`
			ConfigKeys []string       `json:"config_keys"`
			Notes      string         `json:"notes"`
			ConfigPath string         `json:"config_path"`
			Example    map[string]any `json:"example"`
		} `json:"tool_blocks"`
	}
	if err := json.Unmarshal(AppBlocksCatalog(), &catalog); err != nil {
		t.Fatal(err)
	}
	var keys []string
	var notes string
	for _, block := range catalog.ToolBlocks {
		if block.ConfigPath != "config" {
			t.Fatalf("%s must declare its renderer config path", block.Type)
		}
		if block.Type == "flash-firmware" {
			keys = block.ConfigKeys
			notes = block.Notes
			if err := ValidateDeploymentBlocks(toolBlockBody(block.Example)); err != nil {
				t.Fatalf("catalog example must be writable: %v", err)
			}
		}
	}
	want := []string{
		"device_id", "device_type", "flash_mode",
		"firmware_url", "firmware_name", "firmware_size",
		"flash_address",
		"project_zip_url", "project_zip_name", "project_zip_size", "precheck_fields",
		"muse_gadget",
		"full_flash",
	}
	if len(keys) != len(want) {
		t.Fatalf("config_keys=%v", keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("config_keys=%v", keys)
		}
	}
	for _, forbidden := range []string{"flash_target", "flash_address_adapted", "mgst_"} {
		for _, key := range keys {
			if key == forbidden {
				t.Fatalf("catalog must not list %s", forbidden)
			}
		}
	}
	for _, phrase := range []string{"muse_gadget", "full_flash", "114993068", "0x20000", "Muse application image", "does not have to stay that name", "the CLI rejects true", "do not write 0x20000", "0x8000", "set full_flash false", "merged-binary.bin", "must not contain application", "defaults full_flash to true", "device_defaults", "build_artifact", "build_recipe", "merged image", "firmware advise"} {
		if !strings.Contains(notes, phrase) {
			t.Fatalf("notes missing %q: %s", phrase, notes)
		}
	}
	if strings.Contains(notes, "set full_flash true") {
		t.Fatalf("notes still tell agents to set full_flash true with Muse: %s", notes)
	}
	var catalogFull struct {
		ToolBlocks []struct {
			Type           string           `json:"type"`
			DeviceDefaults []map[string]any `json:"device_defaults"`
		} `json:"tool_blocks"`
	}
	if err := json.Unmarshal(AppBlocksCatalog(), &catalogFull); err != nil {
		t.Fatal(err)
	}
	for _, block := range catalogFull.ToolBlocks {
		if block.Type != "flash-firmware" {
			continue
		}
		if len(block.DeviceDefaults) < 8 {
			t.Fatalf("device_defaults too short: %#v", block.DeviceDefaults)
		}
		var sawMerged, sawAppRegion, sawRecipe bool
		for _, row := range block.DeviceDefaults {
			hint, _ := row["build_artifact"].(string)
			if hint == "" {
				t.Fatalf("device_defaults row missing build_artifact: %#v", row)
			}
			if row["build_recipe"] == nil {
				t.Fatalf("device_defaults row missing build_recipe: %#v", row)
			}
			sawRecipe = true
			addr, _ := row["flash_address"].(string)
			switch addr {
			case "0x0":
				if !strings.Contains(hint, "merged") {
					t.Fatalf("0x0 build_artifact should mention merged: %q", hint)
				}
				sawMerged = true
			case "0x90000", "0x4000":
				if !strings.Contains(hint, "application-region") {
					t.Fatalf("%s build_artifact should mention application-region: %q", addr, hint)
				}
				sawAppRegion = true
			}
		}
		if !sawRecipe {
			t.Fatal("expected build_recipe on device_defaults")
		}
		if !sawMerged || !sawAppRegion {
			t.Fatalf("expected both merged and application-region build hints in device_defaults")
		}
		return
	}
	t.Fatal("flash-firmware device_defaults missing")
}
