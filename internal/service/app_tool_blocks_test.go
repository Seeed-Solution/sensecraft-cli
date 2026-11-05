package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
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
		{"muse_gadget", "false"}, {"full_flash", 1}, {"full_flash", true},
	} {
		t.Run(tc.key+"/"+reflect.TypeOf(tc.value).String(), func(t *testing.T) {
			block := watcherFlashBlock()
			block["config"].(map[string]any)[tc.key] = tc.value
			if err := ValidateDeploymentBlocks(toolBlockBody(block)); err == nil {
				t.Fatal("accepted invalid firmware settings")
			}
		})
	}
	block := watcherFlashBlock()
	cfg := block["config"].(map[string]any)
	cfg["device_id"] = "113991115"
	cfg["device_type"] = "32"
	cfg["muse_gadget"] = false
	cfg["flash_mode"] = "zip"
	cfg["project_zip_url"] = "https://example.invalid/project.zip"
	cfg["project_zip_size"] = "1234"
	if err := ValidateDeploymentBlocks(toolBlockBody(block)); err != nil {
		t.Fatalf("nested ZIP fields should remain supported: %v", err)
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
	api := newHandlerAIClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{"app_id": "12345", "be_public": 0}
		if r.URL.Path == enum.AIApiPathAppList {
			data = map[string]any{"list": []any{}}
		}
		if r.URL.Path == enum.AIApiPathAppUpdate {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			writes = append(writes, body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
	}))
	s := AppService{client: api, remoteType: "Global"}
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
