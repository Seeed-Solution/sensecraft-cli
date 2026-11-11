package service

import (
	"fmt"
	"strings"
)

// FirmwareBuildRecipe is the machine-readable "how to produce the binary"
// contract for one flash-firmware device_defaults row.
type FirmwareBuildRecipe struct {
	Artifact         string   `json:"artifact"`
	Chip             string   `json:"chip"`
	FlashMode        string   `json:"flash_mode"`
	RecommendedName  string   `json:"recommended_name"`
	UploadFolder     string   `json:"upload_folder,omitempty"`
	Forbid           []string `json:"forbid"`
	Steps            []string `json:"steps"`
	Notes            string   `json:"notes"`
	MuseAlt          *FirmwareMuseAlt `json:"muse_alt,omitempty"`
}

// FirmwareMuseAlt is the Muse Gadget path for a Muse-capable board.
type FirmwareMuseAlt struct {
	Artifact         string `json:"artifact"`
	RecommendedName  string `json:"recommended_name"`
	AppWriteOffset   string `json:"app_write_offset"`
	ConfigAddress    string `json:"config_flash_address"`
	StepsNote        string `json:"steps_note"`
}

// FirmwareAdviseResult is returned by `firmware advise` and embedded on
// device_defaults as build_recipe for Agents.
type FirmwareAdviseResult struct {
	Name              string               `json:"name"`
	DeviceID          string               `json:"device_id"`
	DeviceIDs         []string             `json:"device_ids,omitempty"`
	FlashAddress      string               `json:"flash_address"`
	FlashAddressFixed bool                 `json:"flash_address_fixed"`
	MuseCapable       bool                 `json:"muse_capable"`
	FullFlashDefault  bool                 `json:"full_flash_default"`
	BuildArtifact     string               `json:"build_artifact"`
	BuildRecipe       FirmwareBuildRecipe  `json:"build_recipe"`
	UploadExample     string               `json:"upload_example"`
	AskBeforeBuild    string               `json:"ask_before_build,omitempty"`
}

type firmwareDeviceSpec struct {
	name, sku, address, file, chip, folder string
	aliases                                []string
	muse, fixed                            bool
	artifact                               string
	forbid                                 []string
	recommendedName                        string
	notes                                  string
	extraSteps                             []string
}

func firmwareDeviceSpecs() []firmwareDeviceSpec {
	return []firmwareDeviceSpec{
		{
			name: "SenseCAP Indicator D1 / D1S / D1Pro", sku: "114993068", address: "0x0",
			file: "merged ESP32-S3 .bin only; aliases 114993069/114993071; Muse → muse-gadget.bin",
			chip: "esp32s3", folder: "sensecap-indicator", aliases: []string{"114993069", "114993071"},
			muse: true, fixed: true, artifact: "merged_at_0x0",
			forbid: []string{"0x10000", "zip", "app-only-at-0x0"},
			recommendedName: "merged.bin",
			notes: "Non-Muse: one merged image from 0x0. Muse: upload muse-gadget.bin only; site writes app at 0x20000.",
		},
		{
			name: "SenseCAP Watcher", sku: "113991315", address: "0x0",
			file: "merged .bin from 0x0 only (no 0x10000); also 113991275 / 114993275; Muse → muse-gadget.bin",
			chip: "esp32s3", folder: "sensecap-watcher", aliases: []string{"113991275", "114993275"},
			muse: true, fixed: true, artifact: "merged_at_0x0",
			forbid: []string{"0x10000", "zip", "app-only-at-0x0"},
			recommendedName: "merged.bin",
			notes: "Flash on the Xiao / ESP32-S3 port. Non-Muse needs a merged image, not an SSCMA application package.",
		},
		{
			name: "reTerminal E1001", sku: "100073581", address: "0x90000",
			file: "application-region .bin only (no 0x0); Muse → muse-gadget.bin; restore bootloader via full_flash",
			chip: "esp32s3", folder: "reterminal-e1001",
			muse: true, fixed: true, artifact: "app_at_0x90000",
			forbid: []string{"0x0", "zip", "merged-whole-flash-as-app"},
			recommendedName: "app.bin",
			notes: "Ship application-region .bin only. full_flash true rewrites stock bootloader at 0x0 and table at 0x8000.",
		},
		{
			name: "reTerminal E1002", sku: "100017057", address: "0x90000",
			file: "application-region .bin only (no 0x0); Muse → muse-gadget.bin; restore bootloader via full_flash",
			chip: "esp32s3", folder: "reterminal-e1002",
			muse: true, fixed: true, artifact: "app_at_0x90000",
			forbid: []string{"0x0", "zip", "merged-whole-flash-as-app"},
			recommendedName: "app.bin",
			notes: "Same as E1001: application-region .bin at 0x90000; do not upload a 0x0 merge as the flash file.",
		},
		{
			name: `XIAO 1.47" Touch Display`, sku: "100069905", address: "0x0",
			file: "merged .bin from 0x0 only; Muse → muse-gadget.bin",
			chip: "esp32s3", folder: "xiao-esp32-s3-147",
			muse: true, fixed: true, artifact: "merged_at_0x0",
			forbid: []string{"0x10000", "zip", "app-only-at-0x0"},
			recommendedName: "merged.bin",
			notes: "Non-Muse merged image from 0x0. Muse uses muse-gadget.bin with config flash_address still 0x0.",
		},
		{
			name: "reTerminal Sticky", sku: "100056818", address: "0x0",
			file: "merged .bin from 0x0 only; Muse → muse-gadget.bin",
			chip: "esp32s3", folder: "reterminal-sticky",
			muse: true, fixed: true, artifact: "merged_at_0x0",
			forbid: []string{"0x10000", "zip", "app-only-at-0x0"},
			recommendedName: "merged.bin",
			notes: "Sticky non-Muse = merged .bin at 0x0 (full_flash defaults true). Muse = muse-gadget.bin only; site multi-segment, app at 0x20000; USB 1a86:55d3.",
			extraSteps: []string{
				"If Muse is unclear, ask once: Muse Gadget or normal Sticky firmware?",
				"Normal: produce a merged ESP32-S3 image (bootloader+partitions+app) named merged.bin",
				"Muse: build with Muse SDK and keep only muse-gadget.bin",
			},
		},
		{
			name: "Wio Terminal", sku: "102991299", address: "0x4000",
			file: "application .bin at 0x4000 only (no 0x0; bootloader via UF2/J-Link)",
			chip: "samd51", folder: "wio-terminal",
			muse: false, fixed: true, artifact: "app_at_0x4000",
			forbid: []string{"0x0", "zip", "muse-gadget"},
			recommendedName: "app.bin",
			notes: "Application-region only. Never flash a 0x0 image (would overwrite UF2 bootloader).",
		},
		{
			name: "Grove Beginner Kit", sku: "110061162", address: "0x0",
			file: ".bin or .hex",
			chip: "nrf52840", folder: "grove-beginner-kit",
			muse: false, fixed: true, artifact: "bin_or_hex_at_0x0",
			forbid: []string{"zip", "muse-gadget"},
			recommendedName: "firmware.bin",
			notes: ".hex embeds addresses; .bin uses 0x0. Small files are allowed (not a merge gate).",
		},
		{
			name: "XIAO ESP32C3", sku: "113991054", address: "0x0",
			file: ".bin or zip; application URL → 0x10000",
			chip: "esp32c3", folder: "",
			muse: false, fixed: false, artifact: "xiao_merged_or_application",
			forbid: []string{"muse-gadget"},
			recommendedName: "firmware.bin",
			notes: "Merged full image → 0x0. SSCMA/application package → 0x10000 (name/URL often contains application). Zip allowed.",
		},
		{
			name: "XIAO ESP32C6", sku: "113991254", address: "0x0",
			file: ".bin or zip; application URL → 0x10000",
			chip: "esp32c6", folder: "",
			muse: false, fixed: false, artifact: "xiao_merged_or_application",
			forbid: []string{"muse-gadget"},
			recommendedName: "firmware.bin",
			notes: "Same dual-address rule as other ordinary XIAO boards.",
		},
		{
			name: "XIAO ESP32S3 Sense", sku: "113991115", address: "0x0",
			file: ".bin or zip; application URL → 0x10000",
			chip: "esp32s3", folder: "",
			muse: false, fixed: false, artifact: "xiao_merged_or_application",
			forbid: []string{"muse-gadget"},
			recommendedName: "firmware.bin",
			notes: "Same dual-address rule as other ordinary XIAO boards.",
		},
		{
			name: `XIAO 0.96"/1.14" IPS (ESP32-S3)`, sku: "", address: "0x0",
			file: "by material name (IPS); merged .bin only; no 0x10000; nRF not supported",
			chip: "esp32s3", folder: "",
			muse: false, fixed: true, artifact: "merged_at_0x0",
			forbid: []string{"0x10000", "zip", "app-only-at-0x0", "nrf52840"},
			recommendedName: "merged.bin",
			notes: "Resolve the catalog SKU via app bom search and include materials[].name containing IPS so CLI can apply the 0x0 rule.",
		},
		{
			name: "XIAO e-paper / EE02–EE05 DIY", sku: "", address: "0x90000",
			file: "by material name (Display/EE0x/epaper DIY, not IPS/Touch/Sense); application-region .bin only",
			chip: "esp32s3", folder: "",
			muse: false, fixed: true, artifact: "app_at_0x90000",
			forbid: []string{"0x0", "zip", "merged-whole-flash-as-app"},
			recommendedName: "app.bin",
			notes: "Include materials[].name (e-paper / EE DIY). Do not ship a 0x0 merge as the flash file.",
		},
	}
}

func flashBuildRecipeForSpec(s firmwareDeviceSpec) FirmwareBuildRecipe {
	steps := []string{
		fmt.Sprintf("Resolve board as %s (device_id %s)", s.name, emptySKU(s.sku)),
		"Run: sensecraft-cli firmware advise --device-id " + emptySKU(s.sku) + " (or --name) before compiling",
	}
	if s.sku == "" {
		steps = []string{
			fmt.Sprintf("Resolve board as %s via app bom search; copy SKU into materials and flash-firmware.device_id", s.name),
			"Run: sensecraft-cli firmware advise --name \"" + s.name + "\" before compiling",
		}
	}
	steps = append(steps, s.extraSteps...)
	switch s.artifact {
	case "merged_at_0x0":
		steps = append(steps,
			"Produce a merged firmware image for "+s.chip+" (bootloader + partition table + app) starting at 0x0",
			"Do not upload an app-only / SSCMA application package for this board",
		)
	case "app_at_0x90000":
		steps = append(steps, "Produce an application-region .bin linked for 0x90000 (no bootloader inside the file)")
	case "app_at_0x4000":
		steps = append(steps, "Produce an application-region .bin for 0x4000; keep the UF2 bootloader out of the file")
	case "xiao_merged_or_application":
		steps = append(steps,
			"Choose artifact: merged full image (flash_address 0x0) OR application package (0x10000, name/URL contains application)",
			"Zip is allowed for this ordinary XIAO SKU",
		)
	case "bin_or_hex_at_0x0":
		steps = append(steps, "Produce .bin (0x0) or .hex with in-file addresses")
	}
	if s.muse {
		steps = append(steps, "If Muse Gadget: build muse-gadget.bin only; set muse_gadget true, full_flash false; keep config flash_address as "+s.address)
	}
	upload := "sensecraft-cli file upload --path ./" + s.recommendedName + " --type 4 --checksum"
	if s.sku != "" {
		upload += " --device-id " + s.sku
	}
	if s.folder != "" {
		steps = append(steps, "Prefer folder upload: sensecraft-cli file upload --path ./"+s.folder+" --type 4 --checksum")
	}
	steps = append(steps,
		"Upload: "+upload,
		"Copy muse.* + size into flash-firmware config; then app create/update",
	)

	recipe := FirmwareBuildRecipe{
		Artifact:        s.artifact,
		Chip:            s.chip,
		FlashMode:       "bin",
		RecommendedName: s.recommendedName,
		UploadFolder:    s.folder,
		Forbid:          append([]string(nil), s.forbid...),
		Steps:           steps,
		Notes:           s.notes,
	}
	if s.artifact == "xiao_merged_or_application" {
		recipe.FlashMode = "bin_or_zip"
	}
	if s.muse {
		recipe.MuseAlt = &FirmwareMuseAlt{
			Artifact:        "muse_gadget",
			RecommendedName: "muse-gadget.bin",
			AppWriteOffset:  "0x20000",
			ConfigAddress:   s.address,
			StepsNote:       "Do not write flash_address 0x20000 in config; the app site applies the Muse multi-segment layout.",
		}
	}
	return recipe
}

func emptySKU(sku string) string {
	if strings.TrimSpace(sku) == "" {
		return "<resolve via bom search>"
	}
	return sku
}

// flashFirmwareDeviceDefaults is exposed on app blocks for Agents that skip
// tool-blocks.md. Order matches the product Muse / address table.
func flashFirmwareDeviceDefaults() []map[string]any {
	specs := firmwareDeviceSpecs()
	out := make([]map[string]any, 0, len(specs))
	for _, s := range specs {
		ids := []string{}
		if s.sku != "" {
			ids = append([]string{s.sku}, s.aliases...)
		}
		out = append(out, map[string]any{
			"name":                 s.name,
			"device_id":            s.sku,
			"device_ids":           ids,
			"flash_address":        s.address,
			"flash_address_fixed":  s.fixed,
			"muse_capable":         s.muse,
			"file":                 s.file,
			"full_flash_default":   s.muse,
			"muse_gadget_requires": "muse-gadget.bin when muse_gadget is true; keep flash_address as device default",
			"build_artifact":       flashBuildArtifactHint(s.address, s.muse),
			"build_recipe":         flashBuildRecipeForSpec(s),
		})
	}
	return out
}

func lookupFirmwareSpec(deviceID, name string) (firmwareDeviceSpec, error) {
	id := strings.TrimSpace(deviceID)
	query := strings.ToLower(strings.TrimSpace(name))
	specs := firmwareDeviceSpecs()

	if id != "" {
		for _, s := range specs {
			if s.sku == id {
				return s, nil
			}
			for _, a := range s.aliases {
				if a == id {
					return s, nil
				}
			}
		}
		// Known SKU in address table but not a named defaults row (shouldn't happen often).
		if policy, ok := lookupFlashAddressPolicy(id); ok {
			return firmwareDeviceSpec{
				name: policy.fileHint, sku: id, address: policy.defaultAddress,
				file: policy.fileHint, chip: "unknown",
				muse: isMuseGadgetSKU(id), fixed: policy.fixed,
				artifact: artifactFromAddress(policy.defaultAddress, isMuseGadgetSKU(id)),
				forbid:   []string{}, recommendedName: "firmware.bin",
				notes: "Matched via flash address policy only; prefer a device_defaults SKU.",
			}, nil
		}
		return firmwareDeviceSpec{}, fmt.Errorf(
			"unknown --device-id %q. Use a SKU from `sensecraft-cli app blocks` → flash-firmware.device_defaults, or pass --name (examples: Sticky 100056818, Watcher 113991315, E1001 100073581)",
			id,
		)
	}

	if query == "" {
		return firmwareDeviceSpec{}, fmt.Errorf("pass --device-id <SKU> or --name <board> (example: sensecraft-cli firmware advise --device-id 100056818)")
	}

	var hits []firmwareDeviceSpec
	for _, s := range specs {
		n := strings.ToLower(s.name)
		folder := strings.ToLower(strings.TrimSpace(s.folder))
		if strings.Contains(n, query) {
			hits = append(hits, s)
			continue
		}
		if folder != "" && (folder == query || strings.Contains(folder, query) || strings.Contains(query, folder)) {
			hits = append(hits, s)
		}
	}
	if len(hits) == 1 {
		return hits[0], nil
	}
	if len(hits) == 0 {
		return firmwareDeviceSpec{}, fmt.Errorf(
			"no board matched --name %q. Try --device-id, or a name like sticky, watcher, e1001, indicator, wio, xiao c3",
			name,
		)
	}
	var names []string
	for _, h := range hits {
		names = append(names, fmt.Sprintf("%s (%s)", h.name, emptySKU(h.sku)))
	}
	return firmwareDeviceSpec{}, fmt.Errorf(
		"--name %q matched multiple boards: %s. Pass --device-id to disambiguate",
		name, strings.Join(names, "; "),
	)
}

func artifactFromAddress(addr string, muse bool) string {
	switch addr {
	case "0x0":
		if muse {
			return "merged_at_0x0"
		}
		return "merged_at_0x0"
	case "0x90000":
		return "app_at_0x90000"
	case "0x4000":
		return "app_at_0x4000"
	default:
		return "unknown"
	}
}

// FirmwareAdvise returns per-device firmware production guidance for Agents.
func FirmwareAdvise(deviceID, name string) (FirmwareAdviseResult, error) {
	s, err := lookupFirmwareSpec(deviceID, name)
	if err != nil {
		return FirmwareAdviseResult{}, err
	}
	recipe := flashBuildRecipeForSpec(s)
	ids := []string{}
	if s.sku != "" {
		ids = append([]string{s.sku}, s.aliases...)
	}
	upload := "sensecraft-cli file upload --path ./" + s.recommendedName + " --type 4 --checksum"
	if s.sku != "" {
		upload += " --device-id " + s.sku
	} else if s.folder != "" {
		upload = "sensecraft-cli file upload --path ./" + s.folder + " --type 4 --checksum"
	}
	ask := ""
	if s.muse {
		ask = "Ask once if unclear: Muse Gadget (muse-gadget.bin) or normal firmware (" + recipe.RecommendedName + ")?"
	} else if s.artifact == "xiao_merged_or_application" {
		ask = "Ask once if unclear: merged image at 0x0, or application package at 0x10000?"
	}
	return FirmwareAdviseResult{
		Name:              s.name,
		DeviceID:          s.sku,
		DeviceIDs:         ids,
		FlashAddress:      s.address,
		FlashAddressFixed: s.fixed,
		MuseCapable:       s.muse,
		FullFlashDefault:  s.muse,
		BuildArtifact:     flashBuildArtifactHint(s.address, s.muse),
		BuildRecipe:       recipe,
		UploadExample:     upload,
		AskBeforeBuild:    ask,
	}, nil
}
