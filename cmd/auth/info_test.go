package auth

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/cmdtest"
	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/enum"
)

func TestAuthInfoJSON(t *testing.T) {
	t.Setenv(enum.EnvNameConfigPath, t.TempDir())
	t.Setenv(enum.EnvNameConfigFile, "")
	config.ResetConfigCache()
	t.Cleanup(config.ResetConfigCache)
	path, err := config.ResolveConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	privateHost := "https://private.example/?token=test-host-secret"
	cli := &config.CliConfig{
		UseApi: "key-1",
		AppConfigList: []config.AccessApiConfig{
			{RemoteType: "Global", OrgId: "org-1", KeyId: "key-1", AccessKey: "test-access-secret", CreateTime: "2026-11-05T10:00:00Z"},
			{RemoteType: "Private", OrgId: "org-2", KeyId: "key-2", AccessKey: "test-other-secret", Host: &privateHost},
			{KeyId: "key-3"},
		},
	}
	if err := config.SaveCliConfigTo(path, cli); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd := newInfoCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"test-access-secret", "test-other-secret", "test-host-secret", "access_key", "host"} {
		if strings.Contains(out.String(), forbidden) {
			t.Fatal("JSON contains a private config field")
		}
	}
	var result struct {
		Profiles []map[string]any `json:"profiles"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Profiles) != 3 {
		t.Fatal("missing profiles")
	}
	first := result.Profiles[0]
	if len(first) != 6 || first["site"] != "Global" || first["org_id"] != "org-1" || first["key_id"] != "key-1" || first["kind"] != "sensecap" || first["created_at"] != "2026-11-05T10:00:00Z" || first["active"] != true {
		t.Fatalf("unexpected profile metadata: %#v", first)
	}
	if result.Profiles[1]["active"] != false || result.Profiles[2]["site"] != "Global" {
		t.Fatal("incorrect inactive flag or default site")
	}

	text, err := cmdtest.Execute(t, newInfoCmd(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Active Key ID: key-1") || !strings.Contains(text, "[inactive]") {
		t.Fatal("default output is not the existing text summary")
	}
	if strings.Contains(text, "test-access-secret") || strings.Contains(text, "test-other-secret") {
		t.Fatal("text output contains an Access Key")
	}
}

func TestAuthInfoJSONEmpty(t *testing.T) {
	t.Setenv(enum.EnvNameConfigPath, t.TempDir())
	t.Setenv(enum.EnvNameConfigFile, "")
	config.ResetConfigCache()
	t.Cleanup(config.ResetConfigCache)
	var out bytes.Buffer
	cmd := newInfoCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var result map[string][]authProfileInfo
	if err := json.Unmarshal(out.Bytes(), &result); err != nil || result["profiles"] == nil || len(result["profiles"]) != 0 {
		t.Fatalf("expected an empty profiles array, got %s (error %v)", out.String(), err)
	}
}
