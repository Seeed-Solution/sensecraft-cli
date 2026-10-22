package device

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/cmdtest"
	"sensecraft-cloud-terminal/internal/enum"

	"github.com/spf13/cobra"
)

func deviceRoot() *cobra.Command {
	root := &cobra.Command{SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(NewDeviceCmd())
	return root
}

func TestDeviceList(t *testing.T) {
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodGet,
		PathSuffix: enum.OpenApiUrlDeviceList,
		Query:      map[string]string{"device_type": "2"},
	}, []map[string]string{{"device_eui": "abc"}})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	stdout, err := cmdtest.Execute(t, deviceRoot(), []string{"device", "list"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if stdout == "" {
		t.Fatal("expected stdout output")
	}
}

func TestDeviceList_RejectedAccessKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":        11104,
			"msg":         "AccessId or accessKey error",
			"serviceName": "sensecap",
		})
	}))
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, deviceRoot(), []string{"device", "list"})
	if err == nil {
		t.Fatal("expected error")
	}
	got := err.Error()
	if strings.Contains(got, "cannot unmarshal") || strings.Contains(got, "ListDeviceResponse") {
		t.Fatalf("leaked decode error: %s", got)
	}
	if !strings.Contains(got, "11104") || !strings.Contains(got, "Access Key") {
		t.Fatalf("got %s", got)
	}
}

func TestDeviceList_UnreadablePayload(t *testing.T) {
	const raw = `{"code":"0","data":{"unexpected":true}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(raw))
	}))
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, deviceRoot(), []string{"device", "list"})
	if err == nil {
		t.Fatal("expected error")
	}
	got := err.Error()
	for _, want := range []string{"HTTP 200", raw} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, "cannot unmarshal") || strings.Contains(got, "ListDeviceResponse") {
		t.Fatalf("leaked decode error: %s", got)
	}
}

func TestDeviceListWithGroupUUID(t *testing.T) {
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodGet,
		PathSuffix: enum.OpenApiUrlDeviceList,
		Query: map[string]string{
			"device_type": "1",
			"group_uuid":  "group-1",
		},
	}, []map[string]string{})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, deviceRoot(), []string{
		"device", "list", "--device-type", "1", "--group-uuid", "group-1",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestDeviceView(t *testing.T) {
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodPost,
		PathSuffix: enum.OpenApiUrlDeviceView,
		Body: map[string]any{
			"device_euis": []any{"eui-1"},
			"device_type": float64(2),
		},
	}, map[string]string{"device_eui": "eui-1"})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, deviceRoot(), []string{
		"device", "view", "--eui", "eui-1",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestDeviceView_MissingEui(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, deviceRoot(), []string{"device", "view"})
	if err == nil {
		t.Fatal("expected error for missing --eui")
	}
}

func TestDeviceChannels(t *testing.T) {
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodPost,
		PathSuffix: enum.OpenApiUrlDeviceChannels,
		Body:       map[string]any{"device_euis": []any{"eui-1"}},
	}, map[string]any{"channels": []any{}})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, deviceRoot(), []string{
		"device", "channels", "--eui", "eui-1",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestDeviceChannels_MissingEui(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, deviceRoot(), []string{"device", "channels"})
	if err == nil {
		t.Fatal("expected error for missing --eui")
	}
}

func TestDeviceStatus(t *testing.T) {
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodPost,
		PathSuffix: enum.OpenApiUrlDeviceRunningStatus,
		Body:       map[string]any{"device_euis": []any{"eui-1"}},
	}, map[string]string{"status": "online"})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, deviceRoot(), []string{
		"device", "status", "--eui", "eui-1",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestDeviceStatus_MissingEui(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, deviceRoot(), []string{"device", "status"})
	if err == nil {
		t.Fatal("expected error for missing --eui")
	}
}

func TestDeviceBind(t *testing.T) {
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodPost,
		PathSuffix: enum.OpenApiUrlDeviceBind,
		Body: map[string]any{
			"eui":         "eui-1",
			"code":        "code-1",
			"device_name": "my-device",
		},
	}, map[string]string{"ok": "true"})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, deviceRoot(), []string{
		"device", "bind",
		"--eui", "eui-1",
		"--code", "code-1",
		"--name", "my-device",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestDeviceBind_MissingRequiredFlags(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, deviceRoot(), []string{"device", "bind", "--eui", "eui-1"})
	if err == nil {
		t.Fatal("expected error for missing --code")
	}
}

func TestDeviceUnbind(t *testing.T) {
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodPost,
		PathSuffix: enum.OpenApiUrlDeviceDelete,
		Body:       map[string]any{"device_euis": []any{"eui-1", "eui-2"}},
	}, map[string]string{"ok": "true"})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, deviceRoot(), []string{
		"device", "unbind", "--eui", "eui-1", "--eui", "eui-2", "--yes",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestDeviceUnbind_RequiresYes(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, deviceRoot(), []string{
		"device", "unbind", "--eui", "eui-1",
	})
	if err == nil {
		t.Fatal("expected error for missing --yes")
	}
}

func TestDeviceUnbind_MissingEui(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, deviceRoot(), []string{"device", "unbind"})
	if err == nil {
		t.Fatal("expected error for missing --eui")
	}
}

func TestDeviceSim(t *testing.T) {
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodGet,
		PathSuffix: enum.OpenApiUrlDeviceSimcard,
		Query:      map[string]string{"iccid": "iccid-1"},
	}, map[string]string{"status": "active"})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, deviceRoot(), []string{
		"device", "sim", "--iccid", "iccid-1",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestDeviceSim_MissingIccid(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, deviceRoot(), []string{"device", "sim"})
	if err == nil {
		t.Fatal("expected error for missing --iccid")
	}
}

func TestDeviceDevkitCreate(t *testing.T) {
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodPost,
		PathSuffix: enum.OpenApiUrlDeviceCreateDevelopmentKit,
		Body: map[string]any{
			"sku":      enum.DevKitSKUXiao,
			"dev_name": "kit-1",
		},
	}, map[string]string{"eui": "0123456789ABCDEF", "device_key": "key"})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	stdout, err := cmdtest.Execute(t, deviceRoot(), []string{
		"device", "devkit", "create",
		"--sku", enum.DevKitSKUXiao,
		"--name", "kit-1",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	cmdtest.AssertJSONOutput(t, stdout, map[string]any{
		"eui":        "0123456789ABCDEF",
		"device_key": "key",
	})
}

func TestDeviceDevkitCreate_MissingSku(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, deviceRoot(), []string{"device", "devkit", "create"})
	if err == nil {
		t.Fatal("expected error for missing --sku")
	}
}

func TestDeviceDevkitKey(t *testing.T) {
	const eui = "0123456789ABCDEF"
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodGet,
		PathSuffix: enum.OpenApiUrlDeviceViewNodeKey,
		Query:      map[string]string{"node_eui": eui},
	}, map[string]string{"eui": eui, "device_key": "key", "token": "token"})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	stdout, err := cmdtest.Execute(t, deviceRoot(), []string{
		"device", "devkit", "key", "--eui", eui,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	cmdtest.AssertJSONOutput(t, stdout, map[string]any{
		"eui":        eui,
		"device_key": "key",
		"token":      "token",
	})
}

func TestDeviceDevkitKey_MissingEui(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, deviceRoot(), []string{"device", "devkit", "key"})
	if err == nil {
		t.Fatal("expected error for missing --eui")
	}
}

func TestDeviceDevkitKey_InvalidEuiLength(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, deviceRoot(), []string{
		"device", "devkit", "key", "--eui", "short",
	})
	if err == nil {
		t.Fatal("expected error for invalid EUI length")
	}
}
