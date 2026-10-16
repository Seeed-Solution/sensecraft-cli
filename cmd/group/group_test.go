package group

import (
	"net/http"
	"testing"

	"sensecraft-cloud-terminal/internal/cmdtest"
	"sensecraft-cloud-terminal/internal/enum"

	"github.com/spf13/cobra"
)

func groupRoot() *cobra.Command {
	root := &cobra.Command{SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(NewGroupCmd())
	return root
}

func TestGroupList(t *testing.T) {
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodGet,
		PathSuffix: enum.OpenApiUrlGroupList,
	}, []map[string]string{{"group_uuid": "g-1", "name": "default"}})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, groupRoot(), []string{"group", "list"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestGroupCreate(t *testing.T) {
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodPost,
		PathSuffix: enum.OpenApiUrlGroupCreate,
		Body:       map[string]any{"name": "new-group"},
	}, map[string]string{"group_uuid": "g-new"})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, groupRoot(), []string{
		"group", "create", "--name", "new-group",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestGroupCreate_MissingName(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, groupRoot(), []string{"group", "create"})
	if err == nil {
		t.Fatal("expected error for missing --name")
	}
}

func TestGroupUpdate(t *testing.T) {
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodPost,
		PathSuffix: enum.OpenApiUrlGroupUpdate,
		Body: map[string]any{
			"group_uuid": "g-1",
			"name":       "renamed",
		},
	}, map[string]string{"ok": "true"})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, groupRoot(), []string{
		"group", "update", "--group-uuid", "g-1", "--name", "renamed",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestGroupUpdate_MissingRequiredFlags(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, groupRoot(), []string{
		"group", "update", "--group-uuid", "g-1",
	})
	if err == nil {
		t.Fatal("expected error for missing --name")
	}
}

func TestGroupDelete(t *testing.T) {
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodPost,
		PathSuffix: enum.OpenApiUrlGroupDelete,
		Query:      map[string]string{"group_uuid": "g-1"},
	}, map[string]string{"ok": "true"})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, groupRoot(), []string{
		"group", "delete", "--group-uuid", "g-1", "--yes",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestGroupDelete_RequiresYes(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, groupRoot(), []string{
		"group", "delete", "--group-uuid", "g-1",
	})
	if err == nil {
		t.Fatal("expected error for missing --yes")
	}
}

func TestGroupDelete_MissingGroupUUID(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, groupRoot(), []string{"group", "delete"})
	if err == nil {
		t.Fatal("expected error for missing --group-uuid")
	}
}

func TestGroupMoveDevices(t *testing.T) {
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodPost,
		PathSuffix: enum.OpenApiUrlGroupMoveDevices,
		Body: map[string]any{
			"group_uuid": "g-1",
			"devices":    []any{"eui-1", "eui-2"},
		},
	}, map[string]string{"ok": "true"})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, groupRoot(), []string{
		"group", "move-devices",
		"--group-uuid", "g-1",
		"--eui", "eui-1",
		"--eui", "eui-2",
		"--yes",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestGroupMoveDevices_RequiresYes(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, groupRoot(), []string{
		"group", "move-devices",
		"--group-uuid", "g-1",
		"--eui", "eui-1",
	})
	if err == nil {
		t.Fatal("expected error for missing --yes")
	}
}

func TestGroupMoveDevices_MissingRequiredFlags(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, groupRoot(), []string{
		"group", "move-devices", "--group-uuid", "g-1",
	})
	if err == nil {
		t.Fatal("expected error for missing --eui")
	}
}
