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
	if res.UploadGuide.Kind != "merged_at_0x0" || !strings.Contains(res.UploadGuide.UploadWhat, "merged") {
		t.Fatalf("upload_guide %#v", res.UploadGuide)
	}
	if !strings.Contains(res.Rule, "merged") {
		t.Fatalf("rule %q", res.Rule)
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
	if res.AskBeforeUpload == "" || res.AskBeforeBuild == "" {
		t.Fatal("expected Muse ask prompt")
	}
}

func TestFirmwareAdviseE1001AppRegion(t *testing.T) {
	res, err := FirmwareAdvise("100073581", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.UploadGuide.Kind != "app_at_0x90000" || res.FlashAddress != "0x90000" {
		t.Fatalf("%#v", res.UploadGuide)
	}
	if !strings.Contains(strings.Join(res.UploadGuide.DoNotUpload, ","), "0x0") {
		t.Fatalf("do_not_upload %#v", res.UploadGuide.DoNotUpload)
	}
}

func TestFirmwareAdviseGroveChip(t *testing.T) {
	res, err := FirmwareAdvise("110061162", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.BuildRecipe.Chip != "atmega328p" {
		t.Fatalf("grove chip %#v", res.BuildRecipe.Chip)
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
	epaper, err := FirmwareAdvise("", "epaper")
	if err != nil {
		t.Fatal(err)
	}
	if epaper.FlashAddress != "0x90000" || epaper.UploadGuide.Kind != "app_at_0x90000" {
		t.Fatalf("epaper %#v", epaper)
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

func TestFirmwareAdviseList(t *testing.T) {
	rows := FirmwareAdviseList()
	if len(rows) < 8 {
		t.Fatalf("list too short: %d", len(rows))
	}
	var sticky bool
	for _, row := range rows {
		if row["device_id"] == "100056818" {
			sticky = true
			if row["upload_what"] == nil || row["flash_address"] != "0x0" {
				t.Fatalf("sticky row %#v", row)
			}
		}
	}
	if !sticky {
		t.Fatal("sticky missing from --list")
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
	if sticky["upload_guide"] == nil || sticky["rule"] == nil {
		t.Fatalf("missing upload_guide/rule %#v", sticky)
	}
}
