package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// museGadgetFile is the only file a Muse device package uploads.
// Bootloader, partition table, OTA, and PHY stay on the app site.
const museGadgetFile = "muse-gadget.bin"

// MuseUploadInfo is returned with a type-4 upload of a Muse device folder.
// Folder name selects the device. Only muse-gadget.bin is uploaded.
type MuseUploadInfo struct {
	DeviceName    string   `json:"device_name"`
	DeviceID      string   `json:"device_id"`
	DeviceIDs     []string `json:"device_ids"`
	FlashAddress  string   `json:"flash_address"`
	FlashMode     string   `json:"flash_mode"`
	MuseGadget    bool     `json:"muse_gadget"`
	FullFlash     bool     `json:"full_flash"`
	BuildArtifact string   `json:"build_artifact,omitempty"`
	IgnoredFiles  []string `json:"ignored_files,omitempty"`
	Note          string   `json:"note"`
}

type folderAlias struct {
	name string
	// musePackage is a Muse SDK output folder. It uploads muse-gadget.bin only.
	// A display name such as "E1001设备" is not a package: muse_gadget is on
	// only when that file is actually in the folder.
	musePackage bool
}

type museDevice struct {
	name         string
	deviceIDs    []string
	flashAddress string
	aliases      []folderAlias
	// muse is true when the editor keeps muse_gadget for this SKU.
	muse bool
	// allowHex is Grove Beginner Kit, which also accepts a .hex firmware.
	allowHex bool
}

// Devices a firmware folder name can select.
// Package aliases are Muse SDK directory slugs. Display aliases cover the
// names used in firmware collections, including a trailing "设备".
var museDeviceTable = []museDevice{
	{
		name:         "SenseCAP Indicator",
		deviceIDs:    []string{"114993068"},
		flashAddress: "0x0",
		muse:         true,
		aliases:      []folderAlias{{"sensecap-indicator", true}},
	},
	{
		name:         "SenseCAP Watcher",
		deviceIDs:    []string{"113991315", "113991275"},
		flashAddress: "0x0",
		muse:         true,
		aliases: []folderAlias{
			{"sensecap-watcher", true},
			{"muse-sensecap-watcher", true},
		},
	},
	{
		name:         "reTerminal E1001",
		deviceIDs:    []string{"100073581"},
		flashAddress: "0x90000",
		muse:         true,
		aliases: []folderAlias{
			{"reterminal-e1001", true},
			{"e1001", false},
		},
	},
	{
		name:         "reTerminal E1002",
		deviceIDs:    []string{"100017057"},
		flashAddress: "0x90000",
		muse:         true,
		aliases: []folderAlias{
			{"reterminal-e1002", true},
			{"e1002", false},
		},
	},
	{
		name:         `XIAO 1.47" Touch Display`,
		deviceIDs:    []string{"100069905"},
		flashAddress: "0x0",
		muse:         true,
		aliases: []folderAlias{
			{"xiao-esp32-s3-147", true},
			{"xiao147", true},
			{"muse-seeed-xiao-s3-147", true},
		},
	},
	{
		name:         "reTerminal Sticky",
		deviceIDs:    []string{"100056818"},
		flashAddress: "0x0",
		muse:         true,
		aliases: []folderAlias{
			{"reterminal-sticky", true},
			{"reterminal-sticky-link", true},
			{"sticky", false},
		},
	},
	{
		name:         "Grove Beginner Kit",
		deviceIDs:    []string{"110061162"},
		flashAddress: "0x0",
		allowHex:     true,
		aliases:      []folderAlias{{"grove-beginner-kit", false}},
	},
	{
		name:         "Wio Terminal",
		deviceIDs:    []string{"102991299"},
		flashAddress: "0x4000",
		aliases:      []folderAlias{{"wio-terminal", false}},
	},
}

type aliasHit struct {
	dev         museDevice
	musePackage bool
}

var museByAlias map[string]aliasHit

func init() {
	museByAlias = make(map[string]aliasHit, len(museDeviceTable))
	for _, dev := range museDeviceTable {
		for _, alias := range dev.aliases {
			key := normMuseKey(alias.name)
			if prev, ok := museByAlias[key]; ok {
				panic(fmt.Sprintf("duplicate firmware folder alias %q (%s and %s)", key, prev.dev.name, dev.name))
			}
			museByAlias[key] = aliasHit{dev: dev, musePackage: alias.musePackage}
		}
	}
}

func normMuseKey(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	return strings.Trim(name, "-")
}

func lookupMuseDevice(folder string) (aliasHit, bool) {
	key := normMuseKey(folder)
	if hit, ok := museByAlias[key]; ok {
		return hit, true
	}
	trimmed := strings.TrimSuffix(key, "设备")
	trimmed = strings.Trim(trimmed, "-")
	if trimmed == "" || trimmed == key {
		return aliasHit{}, false
	}
	hit, ok := museByAlias[trimmed]
	return hit, ok
}

var museSegmentFiles = map[string]struct{}{
	"bootloader.bin":       {},
	"partition-table.bin":  {},
	"ota_data_initial.bin": {},
	"phy_init_data.bin":    {},
	"manifest.json":        {},
}

func isMuseSegmentFile(name string) bool {
	_, ok := museSegmentFiles[strings.ToLower(name)]
	return ok
}

type preparedMuseUpload struct {
	uploadPath string
	info       MuseUploadInfo
}

// prepareMuseFirmwareUpload matches a type-4 path to a device folder.
// A Muse SDK package uploads muse-gadget.bin only. A display name such as
// "E1001设备" or "Wio Terminal" selects that device; muse_gadget is set only
// when muse-gadget.bin is in the folder. Names that match several boards, or
// no board, return nil.
func prepareMuseFirmwareUpload(path string) (*preparedMuseUpload, error) {
	path = strings.TrimSpace(path)
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil
	}
	if info.IsDir() {
		return prepareMuseDir(path)
	}
	parent := filepath.Dir(path)
	hit, ok := lookupMuseDevice(filepath.Base(parent))
	if !ok {
		return nil, nil
	}
	base := filepath.Base(path)
	if hit.dev.muse && strings.EqualFold(base, museGadgetFile) {
		ignored, err := museIgnoredFiles(parent)
		if err != nil {
			return nil, err
		}
		return &preparedMuseUpload{uploadPath: path, info: museUploadInfo(hit.dev, ignored)}, nil
	}
	if hit.musePackage || (hit.dev.muse && museGadgetPresent(parent)) {
		return nil, fmt.Errorf(
			"Muse firmware for %s only uploads %s; %s is not uploaded",
			hit.dev.name, museGadgetFile, base,
		)
	}
	ignored, err := otherFiles(parent, base)
	if err != nil {
		return nil, err
	}
	return &preparedMuseUpload{uploadPath: path, info: deviceUploadInfo(hit.dev, ignored)}, nil
}

func prepareMuseDir(dir string) (*preparedMuseUpload, error) {
	if hit, ok := lookupMuseDevice(filepath.Base(dir)); ok {
		bin, ignored, err := museGadgetIn(dir)
		if err != nil {
			return nil, err
		}
		if hit.dev.muse && bin != "" {
			return &preparedMuseUpload{uploadPath: bin, info: museUploadInfo(hit.dev, ignored)}, nil
		}
		if hit.musePackage {
			return nil, fmt.Errorf("Muse device folder %s is missing %s", filepath.Base(dir), museGadgetFile)
		}
		return pickOneFirmware(dir, hit.dev)
	}
	children, err := matchedChildFolders(dir)
	if err != nil {
		return nil, err
	}
	if len(children) == 0 {
		return nil, nil
	}
	names := make([]string, len(children))
	for i, child := range children {
		hit, _ := lookupMuseDevice(child)
		names[i] = fmt.Sprintf("%s (%s)", child, hit.dev.name)
	}
	return nil, fmt.Errorf(
		"directory holds %d device folders; upload each folder separately with --type 4: %s",
		len(children), strings.Join(names, ", "),
	)
}

func matchedChildFolders(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		if _, ok := lookupMuseDevice(ent.Name()); !ok {
			continue
		}
		names = append(names, ent.Name())
	}
	sort.Strings(names)
	return names, nil
}

func museGadgetPresent(dir string) bool {
	bin, _, err := museGadgetIn(dir)
	return err == nil && bin != ""
}

func museGadgetIn(dir string) (string, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, err
	}
	var bin string
	var ignored []string
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		if strings.EqualFold(ent.Name(), museGadgetFile) {
			if bin != "" {
				return "", nil, fmt.Errorf("Muse device folder %s contains more than one %s", filepath.Base(dir), museGadgetFile)
			}
			bin = filepath.Join(dir, ent.Name())
			continue
		}
		ignored = append(ignored, ent.Name())
	}
	sort.Strings(ignored)
	return bin, ignored, nil
}

func museIgnoredFiles(dir string) ([]string, error) {
	_, ignored, err := museGadgetIn(dir)
	return ignored, err
}

func museUploadInfo(dev museDevice, ignored []string) MuseUploadInfo {
	return firmwareUploadInfo(dev, true, ignored,
		"Uploaded muse-gadget.bin only. Write config.device_id, flash_address, flash_mode, muse_gadget=true, full_flash=false. Keep flash_address as the device default (not 0x20000); the app site writes the app at 0x20000 and supplies bootloader, partition table, OTA, and PHY. Ignored companion files stay local.")
}

func deviceUploadInfo(dev museDevice, ignored []string) MuseUploadInfo {
	note := "Folder name matched this device. Write config.device_id, flash_address, flash_mode, muse_gadget, and full_flash. The app site reads config."
	if dev.muse {
		note += " This SKU supports Muse: muse_gadget=false so default full_flash=true to cover leftover Muse partitions/bootloader. Use an application-region or merged bin for this board — not muse-gadget.bin unless muse_gadget is true."
	} else {
		note += " muse_gadget stays false; full_flash stays false on non-Muse boards."
	}
	return firmwareUploadInfo(dev, false, ignored, note)
}

// flashMergedAtZeroGuidance is appended to type-4 muse.note when the board
// flashes a non-Muse image at 0x0 (Watcher/Sticky/Indicator/IPS/…).
const flashMergedAtZeroGuidance = "flash_address 0x0 requires a merged .bin (bootloader+partitions+app from 0x0); an app-only image linked here will not boot. See app blocks → flash-firmware.device_defaults.build_artifact."

func firmwareUploadInfo(dev museDevice, muse bool, ignored []string, note string) MuseUploadInfo {
	ids := append([]string(nil), dev.deviceIDs...)
	// Align with the app editor: Muse ON never erase-all; Muse-capable board
	// flashing a normal image defaults to full erase to cover Muse layout.
	fullFlash := !muse && dev.muse
	addr := dev.flashAddress
	info := MuseUploadInfo{
		DeviceName:    dev.name,
		DeviceID:      ids[0],
		DeviceIDs:     ids,
		FlashAddress:  addr,
		FlashMode:     "bin",
		MuseGadget:    muse,
		FullFlash:     fullFlash,
		BuildArtifact: flashBuildArtifactHint(addr, dev.muse),
		IgnoredFiles:  ignored,
		Note:          note,
	}
	enrichFirmwareUploadGuidance(&info)
	return info
}

// boardNeedsMergedFirmwareAtZero is true for non-Muse 0x0 targets that expect a
// merged image. Grove Beginner (.bin/.hex at 0x0) is excluded.
func boardNeedsMergedFirmwareAtZero(info *MuseUploadInfo) bool {
	if info == nil || info.MuseGadget || info.FlashAddress != "0x0" {
		return false
	}
	return boardExpectsMergedFirmwareAtZero(info.DeviceID, nil)
}

// isGroveBeginnerSKU is the ATmega328P kit that flashes plain .bin/.hex at 0x0
// (not an ESP merge image).
func isGroveBeginnerSKU(deviceID string) bool {
	id := strings.TrimSpace(deviceID)
	if id == "110061162" {
		return true
	}
	if dev, ok := lookupMuseDeviceByID(id); ok && dev.allowHex {
		return true
	}
	return false
}

// boardExpectsMergedFirmwareAtZero is true for fixed 0x0 SKUs that need a
// bootloader+partitions+app merge (Watcher/Sticky/Indicator/IPS/1.47). Grove
// Beginner keeps a plain 0x0 .bin/.hex and is excluded.
func boardExpectsMergedFirmwareAtZero(deviceID string, materials []map[string]any) bool {
	id := strings.TrimSpace(deviceID)
	if id == "" || isGroveBeginnerSKU(id) {
		return false
	}
	policy, ok := resolveFlashAddressPolicy(id, materials)
	if !ok || !policy.fixed || policy.defaultAddress != "0x0" {
		return false
	}
	return true
}

// enrichFirmwareUploadGuidance makes 0x0→merged guidance impossible to miss in muse.note.
func enrichFirmwareUploadGuidance(info *MuseUploadInfo) {
	if info == nil || !boardNeedsMergedFirmwareAtZero(info) {
		return
	}
	if !strings.Contains(info.Note, "flash_address 0x0 requires a merged") {
		info.Note = strings.TrimSpace(info.Note + " " + flashMergedAtZeroGuidance)
	}
}

// rejectTinyMergedFirmwareUpload blocks type-4 uploads that are clearly too
// small for a non-Muse 0x0 merge target (same threshold as app create/update).
func rejectTinyMergedFirmwareUpload(uploadPath string, size int64, muse *MuseUploadInfo) error {
	if !boardNeedsMergedFirmwareAtZero(muse) {
		return nil
	}
	if size >= flashMinMergedFirmwareBytes {
		return nil
	}
	base := filepath.Base(uploadPath)
	return fmt.Errorf(
		"firmware file %s is %d bytes — too small for device_id %s at flash_address 0x0. That address needs a merged image (bootloader+partitions+app), typically ≥ %d bytes. Rebuild with the board merge recipe (device_defaults.build_artifact), then re-upload. App-only bins belong at 0x10000 (ordinary XIAO) or 0x90000/0x4000 (e-paper/Wio), not 0x0",
		base, size, muse.DeviceID, flashMinMergedFirmwareBytes,
	)
}

// rejectApplicationOnlyFirmwareOnMergedBoard blocks type-4 uploads whose name
// looks like an SSCMA/application package when the board only accepts a 0x0 merge.
func rejectApplicationOnlyFirmwareOnMergedBoard(uploadPath string, muse *MuseUploadInfo) error {
	if !boardNeedsMergedFirmwareAtZero(muse) {
		return nil
	}
	base := filepath.Base(uploadPath)
	if !looksLikeApplicationFirmware("", base) {
		return nil
	}
	return fmt.Errorf(
		"firmware file %s looks like an application-only package (name contains \"application\"), but device_id %s flash_address 0x0 requires a merged image (bootloader+partitions+app). Rebuild with the board merge recipe (device_defaults.build_artifact), then re-upload. Ordinary XIAO application packages use 0x10000 — not Watcher/Sticky/Indicator/IPS/1.47",
		base, muse.DeviceID,
	)
}

func lookupMuseDeviceByID(deviceID string) (museDevice, bool) {
	id := strings.TrimSpace(deviceID)
	if id == "" {
		return museDevice{}, false
	}
	for _, d := range museDeviceTable {
		for _, did := range d.deviceIDs {
			if did == id {
				return d, true
			}
		}
	}
	return museDevice{}, false
}

func deviceIDInList(id string, ids []string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// singleFileFirmwareHint fills muse hints for a bare type-4 file when the
// parent folder did not match a device. Pass --device-id for a concrete SKU.
func singleFileFirmwareHint(path, deviceID string) (*MuseUploadInfo, error) {
	id := strings.TrimSpace(deviceID)
	base := filepath.Base(path)
	museName := looksLikeMuseGadgetFirmware("", base)

	if id == "" {
		note := "Single firmware file uploaded without --device-id, so muse hints are incomplete. Re-run with --device-id <SKU> from `sensecraft-cli app bom search` or `app blocks` → flash-firmware.device_defaults, or upload a matched device folder, then copy muse.* into flash-firmware config (including firmware_size from this upload's size). If the board uses flash_address 0x0 (Watcher/Sticky/Indicator/IPS/1.47), upload a merged .bin — not an app-only image."
		if museName {
			note = "File name looks like muse-gadget.bin but --device-id was omitted. Re-run with --device-id set to a Muse-capable SKU (Indicator/Watcher/E1001/E1002/1.47 Touch/Sticky), or upload a matched Muse SDK folder, before writing flash-firmware with muse_gadget true."
		}
		return &MuseUploadInfo{FlashMode: "bin", Note: note}, nil
	}

	if museName && !isMuseGadgetSKU(id) {
		return nil, fmt.Errorf(
			"file %s looks like muse-gadget.bin but --device-id %s is not Muse-capable. Use a Muse-capable SKU from app blocks → flash-firmware.device_defaults, or upload a normal board image whose name does not contain \"muse-gadget\"",
			base, id,
		)
	}

	if dev, ok := lookupMuseDeviceByID(id); ok {
		if museName {
			info := museUploadInfo(dev, nil)
			info.Note = "Single-file upload with --device-id matched this Muse-capable board and a muse-gadget.bin name. Write config from muse.* (muse_gadget true, full_flash false, device default flash_address). Copy upload size into firmware_size."
			enrichFirmwareUploadGuidance(&info)
			return &info, nil
		}
		info := deviceUploadInfo(dev, nil)
		info.Note = "Single-file upload with --device-id matched this board. Write config from muse.* (including build_artifact). Copy upload size into firmware_size. For Muse Gadget builds, name the file muse-gadget.bin and set muse_gadget true."
		enrichFirmwareUploadGuidance(&info)
		return &info, nil
	}

	policy, ok := lookupFlashAddressPolicy(id)
	if !ok {
		return nil, fmt.Errorf(
			"--device-id %q is not a known flash target. Use a SKU from `sensecraft-cli app blocks` → flash-firmware.device_defaults, or omit --device-id and upload a matched device folder (e.g. reterminal-e1001, sensecap-watcher)",
			id,
		)
	}
	addr := policy.defaultAddress
	if !policy.fixed && looksLikeApplicationFirmware("", base) {
		addr = "0x10000"
	}
	museCapable := isMuseGadgetSKU(id)
	info := MuseUploadInfo{
		DeviceName:    policy.fileHint,
		DeviceID:      id,
		DeviceIDs:     []string{id},
		FlashAddress:  addr,
		FlashMode:     "bin",
		MuseGadget:    false,
		FullFlash:     museCapable,
		BuildArtifact: flashBuildArtifactHint(addr, museCapable),
		Note:          "Single-file upload with --device-id. Write config.device_id, flash_address, flash_mode, muse_gadget, full_flash from muse.*; follow build_artifact when generating the binary; copy upload size into firmware_size.",
	}
	enrichFirmwareUploadGuidance(&info)
	return &info, nil
}

func pickOneFirmware(dir string, dev museDevice) (*preparedMuseUpload, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var candidates []string
	var ignored []string
	for _, ent := range entries {
		if ent.IsDir() || strings.HasPrefix(ent.Name(), ".") {
			continue
		}
		if isMuseSegmentFile(ent.Name()) || !firmwareExt(ent.Name(), dev.allowHex) {
			ignored = append(ignored, ent.Name())
			continue
		}
		candidates = append(candidates, ent.Name())
	}
	sort.Strings(candidates)
	sort.Strings(ignored)
	folder := filepath.Base(dir)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("folder %s matches %s but has no firmware file to upload", folder, dev.name)
	}
	if len(candidates) > 1 {
		return nil, fmt.Errorf(
			"folder %s matches %s and has %d firmware files; pass one file: %s",
			folder, dev.name, len(candidates), strings.Join(candidates, ", "),
		)
	}
	return &preparedMuseUpload{
		uploadPath: filepath.Join(dir, candidates[0]),
		info:       deviceUploadInfo(dev, append(ignored, candidates[1:]...)),
	}, nil
}

func firmwareExt(name string, allowHex bool) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".bin" {
		return true
	}
	return allowHex && ext == ".hex"
}

func otherFiles(dir, keep string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var ignored []string
	for _, ent := range entries {
		if ent.IsDir() || strings.HasPrefix(ent.Name(), ".") || ent.Name() == keep {
			continue
		}
		ignored = append(ignored, ent.Name())
	}
	sort.Strings(ignored)
	return ignored, nil
}
