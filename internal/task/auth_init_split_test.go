package task

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/enum"
	"sensecraft-cloud-terminal/internal/service"
	"sensecraft-cloud-terminal/internal/ui"
)

func TestAuthInitNonInteractiveNamesSplitCommands(t *testing.T) {
	console, _, _ := splitConsole("")
	err := RunAuthInit(context.Background(), console, SplitAuthOptions{})
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"--no-wait", "--device-code", "--key-id", "--label", "--access-level", "end this turn", "unchanged"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %s: %v", want, err)
		}
	}
}

func TestSandboxWithoutSplitEntersWizard(t *testing.T) {
	console, out, _ := splitConsole("")
	err := RunAuthInit(context.Background(), console, SplitAuthOptions{Sandbox: true})
	if err == nil {
		t.Fatal("expected the wizard to stop on stdin")
	}
	if strings.Contains(err.Error(), "--no-wait") {
		t.Fatalf("sandbox should enter the wizard, got %v", err)
	}
	if !strings.Contains(out.String(), "SenseCraft CLI Setup Wizard") {
		t.Fatalf("stdout:\n%s", out.String())
	}
}

func TestAuthInitNoWaitReturnsDeviceCodeWithoutPolling(t *testing.T) {
	dir := isolateAuthConfig(t)
	console, out, _ := splitConsole("")
	polls := 0
	opts := SplitAuthOptions{
		Site:   "global",
		NoWait: true,
		flow: authFlow{
			start: func(context.Context, string, bool) (*service.DeviceAuthSession, error) {
				return fakeDeviceSession(), nil
			},
			poll: func(context.Context, string, *service.DeviceAuthSession, bool) (*service.DeviceTokenResult, error) {
				polls++
				return nil, os.ErrClosed
			},
		},
	}
	if err := RunAuthInit(context.Background(), console, opts); err != nil {
		t.Fatal(err)
	}
	if polls != 0 {
		t.Fatalf("poll calls = %d", polls)
	}

	var got authStartResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("stdout %q: %v", out.String(), err)
	}
	if got.VerificationURL != fakeDeviceSession().VerificationURIComplete || got.DeviceCode != "dev-1" || got.UserCode != "ABCD-2345" || got.Site != enum.RemoteTypeGlobal || got.ExpiresIn != 900 {
		t.Fatalf("result: %+v", got)
	}
	for _, want := range []string{"end this turn", "unchanged", "invalidates", "dev-1"} {
		if !strings.Contains(got.Hint, want) {
			t.Fatalf("hint missing %s: %s", want, got.Hint)
		}
	}
	if strings.Contains(out.String(), "token-should-stay-local") {
		t.Fatalf("stdout leaked a token:\n%s", out.String())
	}

	pendingPath := filepath.Join(dir, "sensecraft-auth-pending.json")
	fi, err := os.Stat(pendingPath)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("pending mode = %o", fi.Mode().Perm())
	}
	body, err := os.ReadFile(pendingPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "access_token") {
		t.Fatalf("pending file stored a token before authorization:\n%s", body)
	}
}

func TestAuthInitNoWaitChinaNotesDataOnly(t *testing.T) {
	isolateAuthConfig(t)
	console, out, _ := splitConsole("")
	opts := SplitAuthOptions{
		Site:   "China",
		NoWait: true,
		flow: authFlow{
			start: func(context.Context, string, bool) (*service.DeviceAuthSession, error) {
				return fakeDeviceSession(), nil
			},
		},
	}
	if err := RunAuthInit(context.Background(), console, opts); err != nil {
		t.Fatal(err)
	}
	var got authStartResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Site != enum.RemoteTypeChina || !strings.Contains(got.Note, "SenseCraft Data only") {
		t.Fatalf("result: %+v", got)
	}
}

func TestAuthInitNoWaitRequiresSite(t *testing.T) {
	console, _, _ := splitConsole("")
	err := RunAuthInit(context.Background(), console, SplitAuthOptions{NoWait: true})
	if err == nil || !strings.Contains(err.Error(), "--site") {
		t.Fatalf("got %v", err)
	}
}

func TestAuthInitDeviceCodeListsKeysWithoutPrintingSecrets(t *testing.T) {
	dir := isolateAuthConfig(t)
	console, out, _ := splitConsole("")
	start := SplitAuthOptions{
		Site:   "Global",
		NoWait: true,
		flow: authFlow{start: func(context.Context, string, bool) (*service.DeviceAuthSession, error) {
			return fakeDeviceSession(), nil
		}},
	}
	if err := RunAuthInit(context.Background(), console, start); err != nil {
		t.Fatal(err)
	}

	console, out, _ = splitConsole("")
	polls := 0
	opts := SplitAuthOptions{
		DeviceCode: "dev-1",
		flow: authFlow{
			poll: func(context.Context, string, *service.DeviceAuthSession, bool) (*service.DeviceTokenResult, error) {
				polls++
				return &service.DeviceTokenResult{AccessToken: "token-should-stay-local"}, nil
			},
			org: func(context.Context, string, string, bool) (*service.UserOrgInfo, error) {
				return &service.UserOrgInfo{Account: "a@example.com", Organization: "Org", OrgID: "org-1"}, nil
			},
			list: func(context.Context, string, string, bool) ([]service.AccessKeyItem, error) {
				return []service.AccessKeyItem{{
					Label: "cli", AccessID: "key-1", AccessKey: "key-should-stay-off-stdout", AccessLevel: "write",
				}}, nil
			},
		},
	}
	if err := RunAuthInit(context.Background(), console, opts); err != nil {
		t.Fatal(err)
	}
	if polls != 1 {
		t.Fatalf("poll calls = %d", polls)
	}
	if strings.Contains(out.String(), "token-should-stay-local") || strings.Contains(out.String(), "key-should-stay-off-stdout") {
		t.Fatalf("stdout leaked a secret:\n%s", out.String())
	}
	var got authListResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("stdout %q: %v", out.String(), err)
	}
	if got.OrgID != "org-1" || len(got.Keys) != 1 || got.Keys[0].AccessID != "key-1" || got.Keys[0].AccessLevel != "write" {
		t.Fatalf("result: %+v", got)
	}
	for _, want := range []string{"--key-id", "--label", "Do not pick", "--access-level", "select one existing", "Access Key name"} {
		if !strings.Contains(got.Hint, want) {
			t.Fatalf("hint missing %s: %s", want, got.Hint)
		}
	}

	body, err := os.ReadFile(filepath.Join(dir, "sensecraft-auth-pending.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "token-should-stay-local") {
		t.Fatalf("pending file:\n%s", body)
	}

	console, out, _ = splitConsole("")
	opts.flow.poll = func(context.Context, string, *service.DeviceAuthSession, bool) (*service.DeviceTokenResult, error) {
		polls++
		return nil, os.ErrClosed
	}
	if err := RunAuthInit(context.Background(), console, opts); err != nil {
		t.Fatal(err)
	}
	if polls != 1 {
		t.Fatalf("second list polled again, polls = %d", polls)
	}
}

func TestAuthInitDeviceCodeSavesSelectedKey(t *testing.T) {
	dir := isolateAuthConfig(t)
	console, _, _ := splitConsole("")
	if err := RunAuthInit(context.Background(), console, SplitAuthOptions{
		Site:   "Global",
		NoWait: true,
		flow: authFlow{start: func(context.Context, string, bool) (*service.DeviceAuthSession, error) {
			return fakeDeviceSession(), nil
		}},
	}); err != nil {
		t.Fatal(err)
	}

	console, out, _ := splitConsole("")
	err := RunAuthInit(context.Background(), console, SplitAuthOptions{
		DeviceCode: "dev-1",
		KeyID:      "key-1",
		flow: authFlow{
			poll: func(context.Context, string, *service.DeviceAuthSession, bool) (*service.DeviceTokenResult, error) {
				return &service.DeviceTokenResult{AccessToken: "token-should-stay-local"}, nil
			},
			org: func(context.Context, string, string, bool) (*service.UserOrgInfo, error) {
				return &service.UserOrgInfo{Account: "a@example.com", Organization: "Org", OrgID: "org-1"}, nil
			},
			list: func(context.Context, string, string, bool) ([]service.AccessKeyItem, error) {
				return []service.AccessKeyItem{{
					Label: "cli", AccessID: "key-1", AccessKey: "key-should-stay-off-stdout", AccessLevel: "write",
				}}, nil
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "token-should-stay-local") || strings.Contains(out.String(), "key-should-stay-off-stdout") {
		t.Fatalf("stdout leaked a secret:\n%s", out.String())
	}
	var got authSaveResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("stdout %q: %v", out.String(), err)
	}
	if got.KeyID != "key-1" || got.OrgID != "org-1" || got.Site != enum.RemoteTypeGlobal {
		t.Fatalf("result: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "sensecraft-auth-pending.json")); !os.IsNotExist(err) {
		t.Fatalf("pending file still present: %v", err)
	}

	config.ResetConfigCache()
	cli, err := config.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cli.UseApi != "key-1" || len(cli.AppConfigList) != 1 || cli.AppConfigList[0].AccessKey != "key-should-stay-off-stdout" || cli.AppConfigList[0].OrgId != "org-1" {
		t.Fatalf("config: %+v", cli)
	}
}

func TestAuthInitDeviceCodeCreatesKey(t *testing.T) {
	isolateAuthConfig(t)
	console, _, _ := splitConsole("")
	if err := RunAuthInit(context.Background(), console, SplitAuthOptions{
		Site:    "Global",
		Sandbox: true,
		NoWait:  true,
		flow: authFlow{start: func(_ context.Context, remoteType string, sandbox bool) (*service.DeviceAuthSession, error) {
			if remoteType != enum.RemoteTypeGlobal || !sandbox {
				t.Fatalf("start remote=%s sandbox=%v", remoteType, sandbox)
			}
			return fakeDeviceSession(), nil
		}},
	}); err != nil {
		t.Fatal(err)
	}

	console, out, _ := splitConsole("")
	var sawLevel string
	var sawSandbox bool
	err := RunAuthInit(context.Background(), console, SplitAuthOptions{
		DeviceCode:  "dev-1",
		Label:       "agent",
		AccessLevel: "read-write",
		flow: authFlow{
			poll: func(_ context.Context, _ string, _ *service.DeviceAuthSession, sandbox bool) (*service.DeviceTokenResult, error) {
				sawSandbox = sandbox
				return &service.DeviceTokenResult{AccessToken: "token-should-stay-local"}, nil
			},
			org: func(context.Context, string, string, bool) (*service.UserOrgInfo, error) {
				return &service.UserOrgInfo{OrgID: "org-1", Account: "a@example.com", Organization: "Org"}, nil
			},
			create: func(_ context.Context, _, _, label, level string, _ bool) (*service.AccessKeyItem, error) {
				sawLevel = level
				return &service.AccessKeyItem{Label: label, AccessID: "key-new", AccessKey: "created-secret", AccessLevel: level}, nil
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sawSandbox {
		t.Fatal("resume dropped sandbox")
	}
	if sawLevel != enum.ApiAccessLevelRW {
		t.Fatalf("level = %s", sawLevel)
	}
	var got authSaveResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.KeyID != "key-new" || strings.Contains(out.String(), "created-secret") {
		t.Fatalf("stdout:\n%s", out.String())
	}
}

func TestAuthInitSplitRejectsMixedFlags(t *testing.T) {
	console, _, _ := splitConsole("")
	cases := []SplitAuthOptions{
		{NoWait: true, DeviceCode: "dev-1", Site: "Global"},
		{NoWait: true, Site: "Global", KeyID: "key-1"},
		{DeviceCode: "dev-1", KeyID: "key-1", Label: "agent"},
		{DeviceCode: "dev-1", KeyID: "key-1", AccessLevel: "only-read"},
		{Site: "Mars", NoWait: true},
	}
	for _, opts := range cases {
		if err := RunAuthInit(context.Background(), console, opts); err == nil {
			t.Fatalf("expected error for %+v", opts)
		}
	}
}

func TestAuthInitDeviceCodeMustMatchPending(t *testing.T) {
	isolateAuthConfig(t)
	console, _, _ := splitConsole("")
	if err := RunAuthInit(context.Background(), console, SplitAuthOptions{
		Site:   "Global",
		NoWait: true,
		flow: authFlow{start: func(context.Context, string, bool) (*service.DeviceAuthSession, error) {
			return fakeDeviceSession(), nil
		}},
	}); err != nil {
		t.Fatal(err)
	}
	err := RunAuthInit(context.Background(), console, SplitAuthOptions{DeviceCode: "other"})
	if err == nil || !strings.Contains(err.Error(), "--no-wait") {
		t.Fatalf("got %v", err)
	}
}

func splitConsole(in string) (*ui.Console, *bytes.Buffer, *bytes.Buffer) {
	var out, errOut bytes.Buffer
	return ui.NewConsole(strings.NewReader(in), &out, &errOut), &out, &errOut
}

func isolateAuthConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(enum.EnvNameConfigPath, dir)
	t.Setenv(enum.EnvNameConfigFile, "")
	config.ResetConfigCache()
	t.Cleanup(config.ResetConfigCache)
	return dir
}

func fakeDeviceSession() *service.DeviceAuthSession {
	return &service.DeviceAuthSession{
		DeviceCode:              "dev-1",
		UserCode:                "ABCD-2345",
		VerificationURI:         "https://account.seeed.cc/device",
		VerificationURIComplete: "https://account.seeed.cc/device?user_code=ABCD-2345",
		ExpiresIn:               900,
		Interval:                5,
	}
}
