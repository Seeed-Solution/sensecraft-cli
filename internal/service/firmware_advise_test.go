package service

import (
	"strings"
	"testing"
)

func TestFirmwareAdviseSticky(t *testing.T) {
	res, err := FirmwareAdvise("100056818", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.DeviceID != "100056818" || res.FlashAddress != "0x0" || !res.MuseCapable {
		t.Fatalf("%#v", res)
	}
	if res.BuildRecipe.Artifact != "merged_at_0x0" || res.BuildRecipe.Chip != "esp32s3" {
		t.Fatalf("recipe %#v", res.BuildRecipe)
	}
	if res.BuildRecipe.MuseAlt == nil || res.BuildRecipe.MuseAlt.RecommendedName != "muse-gadget.bin" {
		t.Fatalf("muse alt %#v", res.BuildRecipe.MuseAlt)
	}
	joined := strings.Join(res.BuildRecipe.Forbid, ",")
	if !strings.Contains(joined, "0x10000") || !strings.Contains(joined, "app-only-at-0x0") {
		t.Fatalf("forbid %#v", res.BuildRecipe.Forbid)
	}
	if !strings.Contains(res.UploadExample, "--device-id 100056818") {
		t.Fatalf("upload example %q", res.UploadExample)
	}
	if res.AskBeforeBuild == "" {
		t.Fatal("expected Muse ask prompt")
	}
}

func TestFirmwareAdviseByNameAndAlias(t *testing.T) {
	res, err := FirmwareAdvise("", "sticky")
	if err != nil {
		t.Fatal(err)
	}
	if res.DeviceID != "100056818" {
		t.Fatalf("%#v", res)
	}
	alias, err := FirmwareAdvise("113991275", "")
	if err != nil {
		t.Fatal(err)
	}
	if alias.Name != "SenseCAP Watcher" || alias.DeviceID != "113991315" {
		t.Fatalf("watcher alias %#v", alias)
	}
}

func TestFirmwareAdviseUnknown(t *testing.T) {
	if _, err := FirmwareAdvise("999999999", ""); err == nil {
		t.Fatal("expected unknown SKU error")
	}
	if _, err := FirmwareAdvise("", ""); err == nil {
		t.Fatal("expected missing args error")
	}
}

func TestDeviceDefaultsIncludeBuildRecipe(t *testing.T) {
	rows := flashFirmwareDeviceDefaults()
	var sticky map[string]any
	for _, row := range rows {
		if row["device_id"] == "100056818" {
			sticky = row
			break
		}
	}
	if sticky == nil {
		t.Fatal("sticky missing from device_defaults")
	}
	recipe, ok := sticky["build_recipe"].(FirmwareBuildRecipe)
	if !ok || recipe.Artifact != "merged_at_0x0" {
		t.Fatalf("build_recipe %#v", sticky["build_recipe"])
	}
}
