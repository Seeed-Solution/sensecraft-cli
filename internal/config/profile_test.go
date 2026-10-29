package config

import (
	"strings"
	"testing"
)

func TestFormatCliConfigInfo(t *testing.T) {
	cli := &CliConfig{
		UseApi: "key-a",
		AppConfigList: []AccessApiConfig{
			{
				RemoteType: "Global",
				OrgId:      "org-1",
				KeyId:      "key-a",
				AccessKey:  "secret-a",
				CreateTime: "2026-01-01T00:00:00Z",
			},
			{
				RemoteType: "China",
				OrgId:      "org-2",
				KeyId:      "key-b",
				CreateTime: "2026-02-01T00:00:00Z",
			},
		},
	}
	out := FormatCliConfigInfo(cli, "/tmp/sensecraft.json")
	if !strings.Contains(out, "Active Key ID: key-a") {
		t.Fatalf("missing active key: %s", out)
	}
	if !strings.Contains(out, "[active]") {
		t.Fatalf("missing active marker: %s", out)
	}
	if !strings.Contains(out, "Organization ID: org-1") {
		t.Fatalf("missing org id: %s", out)
	}
	if strings.Contains(out, "secret-a") {
		t.Fatalf("access key should not be printed: %s", out)
	}
}

func TestFormatCliConfigInfoWhenEmptyTellsAgentToStartSplitLogin(t *testing.T) {
	out := FormatCliConfigInfo(nil, "/tmp/sensecraft.json")
	for _, want := range []string{"--site Global --no-wait", "unchanged", "end this turn", "--device-code"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %s", want, out)
		}
	}
	if strings.Contains(out, "Run: sensecraft-cli auth init\n") {
		t.Fatalf("empty config still points at the wizard: %s", out)
	}
}
