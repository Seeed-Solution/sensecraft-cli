package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFlashFirmwareCatalogKeys(t *testing.T) {
	var catalog struct {
		ToolBlocks []struct {
			Type       string   `json:"type"`
			ConfigKeys []string `json:"config_keys"`
			Notes      string   `json:"notes"`
		} `json:"tool_blocks"`
	}
	if err := json.Unmarshal(AppBlocksCatalog(), &catalog); err != nil {
		t.Fatal(err)
	}
	var keys []string
	var notes string
	for _, block := range catalog.ToolBlocks {
		if block.Type == "flash-firmware" {
			keys = block.ConfigKeys
			notes = block.Notes
			break
		}
	}
	want := []string{
		"device_id", "device_type", "flash_mode",
		"firmware_url", "firmware_name", "firmware_size",
		"flash_address",
		"project_zip_url", "project_zip_name", "project_zip_size", "precheck_fields",
		"muse_gadget",
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
	for _, phrase := range []string{"muse_gadget", "114993068", "0x20000", "muse-gadget.bin", "do not write 0x20000"} {
		if !strings.Contains(notes, phrase) {
			t.Fatalf("notes missing %q: %s", phrase, notes)
		}
	}
}
