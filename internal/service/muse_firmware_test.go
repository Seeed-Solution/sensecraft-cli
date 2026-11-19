package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMusePackage(t *testing.T, dir string, extra ...string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, museGadgetFile), []byte("gadget"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range extra {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMuseFolderAliasesAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, dev := range museDeviceTable {
		if dev.name == "" || len(dev.deviceIDs) == 0 || dev.flashAddress == "" {
			t.Fatalf("incomplete device %#v", dev)
		}
		for _, alias := range dev.aliases {
			key := normMuseKey(alias.name)
			if prev, ok := seen[key]; ok {
				t.Fatalf("alias %q used by %s and %s", key, prev, dev.name)
			}
			seen[key] = dev.name
		}
	}
	if len(seen) != len(museByAlias) {
		t.Fatalf("alias map %d, table %d", len(museByAlias), len(seen))
	}
}

func TestUploadMuseDeviceFolderUploadsGadgetOnly(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		folder   string
		deviceID string
		address  string
		ids      int
	}{
		{"sensecap-indicator", "114993068", "0x0", 1},
		{"sensecap-watcher", "113991315", "0x0", 2},
		{"muse-sensecap-watcher", "113991315", "0x0", 2},
		{"reterminal-e1001", "100073581", "0x90000", 1},
		{"reterminal-e1002", "100017057", "0x90000", 1},
		{"xiao-esp32-s3-147", "100069905", "0x0", 1},
		{"xiao147", "100069905", "0x0", 1},
		{"muse-seeed-xiao-s3-147", "100069905", "0x0", 1},
		{"reterminal-sticky", "100056818", "0x0", 1},
		{"reterminal-sticky-link", "100056818", "0x0", 1},
	}
	s := FileService{sandbox: true}
	for _, tc := range cases {
		dir := filepath.Join(root, tc.folder)
		writeMusePackage(t, dir, "bootloader.bin", "partition-table.bin", "ota_data_initial.bin", "manifest.json")
		res, err := s.UploadFile(FileUploadParams{Path: dir, FileType: FileTypeFirmware})
		if err != nil {
			t.Fatalf("%s: %v", tc.folder, err)
		}
		if res.FileName != museGadgetFile || res.FileType != FileTypeFirmware {
			t.Fatalf("%s uploaded %#v", tc.folder, res)
		}
		if res.Muse == nil || res.Muse.DeviceID != tc.deviceID || res.Muse.FlashAddress != tc.address {
			t.Fatalf("%s muse %#v", tc.folder, res.Muse)
		}
		if !res.Muse.MuseGadget || res.Muse.FullFlash || res.Muse.FlashMode != "bin" {
			t.Fatalf("%s flags %#v", tc.folder, res.Muse)
		}
		if len(res.Muse.DeviceIDs) != tc.ids || res.Muse.DeviceIDs[0] != tc.deviceID {
			t.Fatalf("%s ids %#v", tc.folder, res.Muse.DeviceIDs)
		}
		if len(res.Muse.IgnoredFiles) != 4 {
			t.Fatalf("%s ignored %#v", tc.folder, res.Muse.IgnoredFiles)
		}
		attached := FileAttachFromUpload(res)
		if attached.Muse == nil || attached.Muse.DeviceID != tc.deviceID {
			t.Fatalf("%s attach dropped muse", tc.folder)
		}
	}
}

func TestUploadMuseGadgetFileUsesParentFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "SenseCAP_Indicator")
	writeMusePackage(t, dir, "phy_init_data.bin")
	res, err := FileService{sandbox: true}.UploadFile(FileUploadParams{
		Path:     filepath.Join(dir, museGadgetFile),
		FileType: FileTypeFirmware,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Muse == nil || res.Muse.DeviceName != "SenseCAP Indicator" || res.Muse.DeviceID != "114993068" {
		t.Fatalf("%#v", res.Muse)
	}
}

func TestUploadMuseSiblingIsRejected(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sensecap-indicator")
	writeMusePackage(t, dir, "bootloader.bin")
	_, err := FileService{sandbox: true}.UploadFile(FileUploadParams{
		Path:     filepath.Join(dir, "bootloader.bin"),
		FileType: FileTypeFirmware,
	})
	if err == nil || !strings.Contains(err.Error(), museGadgetFile) {
		t.Fatalf("got %v", err)
	}
}

func TestUploadMuseBundleAsksForEachFolder(t *testing.T) {
	root := t.TempDir()
	writeMusePackage(t, filepath.Join(root, "sensecap-indicator"), "bootloader.bin")
	writeMusePackage(t, filepath.Join(root, "xiao-esp32-s3-147"))
	_, err := FileService{sandbox: true}.UploadFile(FileUploadParams{Path: root, FileType: FileTypeFirmware})
	if err == nil || !strings.Contains(err.Error(), "sensecap-indicator") || !strings.Contains(err.Error(), "xiao-esp32-s3-147") {
		t.Fatalf("got %v", err)
	}
}

func TestUploadPlainFirmwareIsUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "firmware.bin")
	if err := os.WriteFile(path, []byte("fw"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := FileService{sandbox: true}.UploadFile(FileUploadParams{Path: path, FileType: FileTypeFirmware})
	if err != nil {
		t.Fatal(err)
	}
	if res.FileName != "firmware.bin" {
		t.Fatalf("%#v", res)
	}
	if res.Muse == nil || res.Muse.DeviceID != "" || !strings.Contains(res.Muse.Note, "--device-id") {
		t.Fatalf("bare file should return incomplete muse hint: %#v", res.Muse)
	}

	withSKU, err := FileService{sandbox: true}.UploadFile(FileUploadParams{
		Path: path, FileType: FileTypeFirmware, DeviceID: "100073581",
	})
	if err != nil {
		t.Fatal(err)
	}
	if withSKU.Muse == nil || withSKU.Muse.DeviceID != "100073581" || withSKU.Muse.FlashAddress != "0x90000" {
		t.Fatalf("expected E1001 hints: %#v", withSKU.Muse)
	}
	if withSKU.Muse.MuseGadget || !withSKU.Muse.FullFlash {
		t.Fatalf("normal bin on Muse SKU should default full_flash: %#v", withSKU.Muse)
	}
	if withSKU.Muse.BuildArtifact == "" || !strings.Contains(withSKU.Muse.BuildArtifact, "0x90000") {
		t.Fatalf("expected build_artifact hint: %#v", withSKU.Muse)
	}

	musePath := filepath.Join(t.TempDir(), "muse-gadget.bin")
	if err := os.WriteFile(musePath, []byte("mg"), 0o644); err != nil {
		t.Fatal(err)
	}
	museRes, err := FileService{sandbox: true}.UploadFile(FileUploadParams{
		Path: musePath, FileType: FileTypeFirmware, DeviceID: "100073581",
	})
	if err != nil {
		t.Fatal(err)
	}
	if museRes.Muse == nil || !museRes.Muse.MuseGadget || museRes.Muse.FullFlash {
		t.Fatalf("muse-gadget.bin + Muse SKU: %#v", museRes.Muse)
	}
	if _, err := (FileService{sandbox: true}).UploadFile(FileUploadParams{
		Path: musePath, FileType: FileTypeFirmware, DeviceID: "113991054",
	}); err == nil {
		t.Fatal("accepted muse-gadget.bin for non-Muse XIAO")
	}
}

func TestUploadMuseFolderMissingBin(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "reterminal-e1002")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bootloader.bin"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := FileService{sandbox: true}.UploadFile(FileUploadParams{Path: dir, FileType: FileTypeFirmware})
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("got %v", err)
	}
}

func TestUploadNonMuseDirectoryStillRejected(t *testing.T) {
	dir := t.TempDir()
	_, err := FileService{sandbox: true}.UploadFile(FileUploadParams{Path: dir, FileType: FileTypeFirmware})
	if err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("got %v", err)
	}
}

func TestUploadDisplayNameMatchesDevice(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		folder  string
		file    string
		name    string
		id      string
		address string
		muse    bool
		body    string
	}{
		{"E1001设备", "app.bin", "reTerminal E1001", "100073581", "0x90000", false, "fw"},
		{"E1002设备", "app.bin", "reTerminal E1002", "100017057", "0x90000", false, "fw"},
		{"sticky设备", "sticky.bin", "reTerminal Sticky", "100056818", "0x0", false, strings.Repeat("m", flashMinMergedFirmwareBytes)},
		{"Grove Beginner Kit 设备", "kit.hex", "Grove Beginner Kit", "110061162", "0x0", false, "fw"},
		{"Wio Terminal", "wio.bin", "Wio Terminal", "102991299", "0x4000", false, "fw"},
	}
	s := FileService{sandbox: true}
	for _, tc := range cases {
		dir := filepath.Join(root, tc.folder)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, tc.file), []byte(tc.body), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("n"), 0o644); err != nil {
			t.Fatal(err)
		}
		res, err := s.UploadFile(FileUploadParams{Path: dir, FileType: FileTypeFirmware})
		if err != nil {
			t.Fatalf("%s: %v", tc.folder, err)
		}
		if res.FileName != tc.file || res.Muse == nil || res.Muse.MuseGadget != tc.muse {
			t.Fatalf("%s file %s muse %#v", tc.folder, res.FileName, res.Muse)
		}
		if res.Muse.DeviceName != tc.name || res.Muse.DeviceID != tc.id || res.Muse.FlashAddress != tc.address {
			t.Fatalf("%s %#v", tc.folder, res.Muse)
		}
		wantFull := !tc.muse && isMuseGadgetSKU(tc.id) && !isWatcherMuseFlashDevice(tc.id)
		if res.Muse.FullFlash != wantFull {
			t.Fatalf("%s full_flash=%v want %v", tc.folder, res.Muse.FullFlash, wantFull)
		}
		if tc.address == "0x0" && tc.id != "110061162" && !strings.Contains(res.Muse.Note, "flash_address 0x0 requires a merged") {
			t.Fatalf("%s missing merged guidance in note: %s", tc.folder, res.Muse.Note)
		}
	}
}

func TestUploadRejectsTinyMergedBinAtZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.bin")
	if err := os.WriteFile(path, []byte("tiny"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := (FileService{sandbox: true}).UploadFile(FileUploadParams{
		Path: path, FileType: FileTypeFirmware, DeviceID: "100056818",
	})
	if err == nil || !strings.Contains(err.Error(), "too small for device_id 100056818 at flash_address 0x0") {
		t.Fatalf("expected tiny Sticky merge rejection, got: %v", err)
	}

	okPath := filepath.Join(t.TempDir(), "merged.bin")
	if err := os.WriteFile(okPath, []byte(strings.Repeat("m", flashMinMergedFirmwareBytes)), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := (FileService{sandbox: true}).UploadFile(FileUploadParams{
		Path: okPath, FileType: FileTypeFirmware, DeviceID: "100056818",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Muse == nil || !strings.Contains(res.Muse.Note, "flash_address 0x0 requires a merged") {
		t.Fatalf("expected merged guidance: %#v", res.Muse)
	}
}

func TestUploadRejectsApplicationPackageOnWatcherMerged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo_viewer-application.bin")
	if err := os.WriteFile(path, []byte(strings.Repeat("a", 281376)), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := (FileService{sandbox: true}).UploadFile(FileUploadParams{
		Path: path, FileType: FileTypeFirmware, DeviceID: "113991315",
	})
	if err == nil || !strings.Contains(err.Error(), "application-only package") {
		t.Fatalf("expected Watcher application-package rejection, got: %v", err)
	}

	// Ordinary XIAO still accepts *application* at 0x10000.
	xiaoPath := filepath.Join(t.TempDir(), "demo-application.bin")
	if err := os.WriteFile(xiaoPath, []byte(strings.Repeat("x", 64*1024)), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := (FileService{sandbox: true}).UploadFile(FileUploadParams{
		Path: xiaoPath, FileType: FileTypeFirmware, DeviceID: "113991054",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Muse == nil || res.Muse.FlashAddress != "0x10000" {
		t.Fatalf("XIAO application upload should hint 0x10000: %#v", res.Muse)
	}
}

func TestUploadDisplayNameMusePackageStillUploadsGadget(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sticky设备")
	writeMusePackage(t, dir, "bootloader.bin")
	res, err := FileService{sandbox: true}.UploadFile(FileUploadParams{Path: dir, FileType: FileTypeFirmware})
	if err != nil {
		t.Fatal(err)
	}
	if res.FileName != museGadgetFile || res.Muse == nil || !res.Muse.MuseGadget || res.Muse.DeviceID != "100056818" {
		t.Fatalf("%#v", res)
	}
}

func TestUploadAmbiguousFolderNamesDoNotMatch(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"muse固件", "xiao s3", "停车场"} {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "firmware.bin"), []byte("fw"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := FileService{sandbox: true}.UploadFile(FileUploadParams{Path: dir, FileType: FileTypeFirmware})
		if err == nil || !strings.Contains(err.Error(), "directory") {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestFirmwareFolderSelfCheck(t *testing.T) {
	s := FileService{sandbox: true}
	root := t.TempDir()
	parent := filepath.Join(root, "各种设备固件")
	writeFile := func(dir, name, body string) {
		t.Helper()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(filepath.Join(parent, "E1001设备"), "app.bin", "e1001")
	writeFile(filepath.Join(parent, "E1002设备"), "app.bin", "e1002")
	writeFile(filepath.Join(parent, "Grove Beginner Kit 设备"), "kit.hex", "grove")
	writeFile(filepath.Join(parent, "sticky设备"), "sticky.bin", strings.Repeat("m", flashMinMergedFirmwareBytes))
	writeFile(filepath.Join(parent, "Wio Terminal"), "wio.bin", "wio")
	writeFile(filepath.Join(parent, "muse固件"), "firmware.bin", "muse-parent")
	writeFile(filepath.Join(parent, "xiao s3"), "firmware.bin", "xiao")
	writeFile(filepath.Join(parent, "停车场"), "firmware.bin", "park")

	_, err := s.UploadFile(FileUploadParams{Path: parent, FileType: FileTypeFirmware})
	if err == nil {
		t.Fatal("parent folder should not upload as one device")
	}
	for _, want := range []string{"E1001设备", "E1002设备", "Grove Beginner Kit 设备", "sticky设备", "Wio Terminal"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("parent error missing %s: %v", want, err)
		}
	}
	for _, skip := range []string{"muse固件", "xiao s3", "停车场"} {
		if strings.Contains(err.Error(), skip) {
			t.Fatalf("parent error should not treat %s as a device: %v", skip, err)
		}
	}

	checks := []struct {
		folder  string
		file    string
		device  string
		id      string
		address string
		body    string
	}{
		{"E1001设备", "app.bin", "reTerminal E1001", "100073581", "0x90000", "e1001"},
		{"E1002设备", "app.bin", "reTerminal E1002", "100017057", "0x90000", "e1002"},
		{"Grove Beginner Kit 设备", "kit.hex", "Grove Beginner Kit", "110061162", "0x0", "grove"},
		{"sticky设备", "sticky.bin", "reTerminal Sticky", "100056818", "0x0", strings.Repeat("m", flashMinMergedFirmwareBytes)},
		{"Wio Terminal", "wio.bin", "Wio Terminal", "102991299", "0x4000", "wio"},
	}
	for _, tc := range checks {
		res, err := s.UploadFile(FileUploadParams{
			Path:     filepath.Join(parent, tc.folder),
			FileType: FileTypeFirmware,
		})
		if err != nil {
			t.Fatalf("%s: %v", tc.folder, err)
		}
		if res.FileName != tc.file || res.Size != int64(len(tc.body)) || res.FileType != FileTypeFirmware {
			t.Fatalf("%s uploaded %#v", tc.folder, res)
		}
		wantFull := isMuseGadgetSKU(tc.id)
		if res.Muse == nil || res.Muse.MuseGadget || res.Muse.FullFlash != wantFull || res.Muse.FlashMode != "bin" {
			t.Fatalf("%s flags %#v want full_flash=%v", tc.folder, res.Muse, wantFull)
		}
		if res.Muse.DeviceName != tc.device || res.Muse.DeviceID != tc.id || res.Muse.FlashAddress != tc.address {
			t.Fatalf("%s device %#v", tc.folder, res.Muse)
		}
		one, err := s.UploadFile(FileUploadParams{
			Path:     filepath.Join(parent, tc.folder, tc.file),
			FileType: FileTypeFirmware,
		})
		if err != nil || one.FileName != tc.file || one.Muse == nil || one.Muse.DeviceID != tc.id || one.Muse.MuseGadget {
			t.Fatalf("%s file: %v %#v", tc.folder, err, one.Muse)
		}
	}

	_, err = s.UploadFile(FileUploadParams{Path: filepath.Join(parent, "E1001设备"), FileType: FileTypeFile})
	if err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("type 2 must not match a device folder: %v", err)
	}

	wioHex := filepath.Join(root, "wio-hex")
	writeFile(filepath.Join(wioHex, "Wio Terminal"), "only.hex", "nope")
	_, err = s.UploadFile(FileUploadParams{Path: filepath.Join(wioHex, "Wio Terminal"), FileType: FileTypeFirmware})
	if err == nil || !strings.Contains(err.Error(), "no firmware") {
		t.Fatalf("wio hex: %v", err)
	}
	groveBoth := filepath.Join(root, "grove-both")
	writeFile(filepath.Join(groveBoth, "Grove Beginner Kit 设备"), "a.bin", "a")
	writeFile(filepath.Join(groveBoth, "Grove Beginner Kit 设备"), "b.hex", "b")
	_, err = s.UploadFile(FileUploadParams{Path: filepath.Join(groveBoth, "Grove Beginner Kit 设备"), FileType: FileTypeFirmware})
	if err == nil || !strings.Contains(err.Error(), "a.bin") || !strings.Contains(err.Error(), "b.hex") {
		t.Fatalf("grove both: %v", err)
	}

	museDir := filepath.Join(root, "muse-display", "E1001设备")
	writeMusePackage(t, museDir, "bootloader.bin", "partition-table.bin")
	res, err := s.UploadFile(FileUploadParams{Path: museDir, FileType: FileTypeFirmware})
	if err != nil {
		t.Fatal(err)
	}
	if res.FileName != museGadgetFile || res.Muse == nil || !res.Muse.MuseGadget || res.Muse.DeviceID != "100073581" || res.Muse.FlashAddress != "0x90000" {
		t.Fatalf("e1001 muse %#v", res.Muse)
	}
	if res.Size != int64(len("gadget")) {
		t.Fatalf("uploaded size %d", res.Size)
	}
	_, err = s.UploadFile(FileUploadParams{Path: filepath.Join(museDir, "bootloader.bin"), FileType: FileTypeFirmware})
	if err == nil || !strings.Contains(err.Error(), museGadgetFile) {
		t.Fatalf("e1001 sibling: %v", err)
	}

	real := "/Users/guli/Downloads/Muse固件-Indicator-XIAO147/xiao-esp32-s3-147"
	gadget := filepath.Join(real, museGadgetFile)
	info, statErr := os.Stat(gadget)
	if statErr != nil {
		t.Logf("skip real folder: %v", statErr)
		return
	}
	res, err = s.UploadFile(FileUploadParams{Path: real, FileType: FileTypeFirmware})
	if err != nil {
		t.Fatal(err)
	}
	if res.FileName != museGadgetFile || res.Size != info.Size() || res.Muse == nil || !res.Muse.MuseGadget || res.Muse.DeviceID != "100069905" || res.Muse.FlashAddress != "0x0" {
		t.Fatalf("real xiao %#v size %d", res.Muse, res.Size)
	}
	_, err = s.UploadFile(FileUploadParams{Path: filepath.Join(real, "bootloader.bin"), FileType: FileTypeFirmware})
	if err == nil {
		t.Fatal("real bootloader should be rejected")
	}
	_, err = s.UploadFile(FileUploadParams{Path: filepath.Dir(real), FileType: FileTypeFirmware})
	if err == nil || !strings.Contains(err.Error(), "sensecap-indicator") || !strings.Contains(err.Error(), "xiao-esp32-s3-147") {
		t.Fatalf("real parent: %v", err)
	}
}

func TestUploadDisplayNameTwoFirmwareFilesAsks(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "E1001设备")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.bin", "b.bin"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("fw"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := FileService{sandbox: true}.UploadFile(FileUploadParams{Path: dir, FileType: FileTypeFirmware})
	if err == nil || !strings.Contains(err.Error(), "a.bin") || !strings.Contains(err.Error(), "b.bin") {
		t.Fatalf("got %v", err)
	}
}
