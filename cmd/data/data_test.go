package data

import (
	"net/http"
	"testing"

	"sensecraft-cloud-terminal/internal/cmdtest"
	"sensecraft-cloud-terminal/internal/enum"

	"github.com/spf13/cobra"
)

func dataRoot() *cobra.Command {
	root := &cobra.Command{SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(NewDataCmd())
	return root
}

func TestDataLatest(t *testing.T) {
	const eui = "node-eui-1"
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodGet,
		PathSuffix: enum.OpenApiUrlDataViewLatestTelemetry,
		Query: map[string]string{
			"device_eui":     eui,
			"measurement_id": "m-1",
			"channel_index":  "1",
		},
	}, []map[string]any{{"channel_index": 1}})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, dataRoot(), []string{
		"data", "latest",
		"--eui", eui,
		"--measurement-id", "m-1",
		"--channel-index", "1",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestDataLatest_MissingEui(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, dataRoot(), []string{"data", "latest"})
	if err == nil {
		t.Fatal("expected error for missing --eui")
	}
}

func TestDataList(t *testing.T) {
	const eui = "node-eui-1"
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodGet,
		PathSuffix: enum.OpenApiUrlDataListTelemetry,
		Query: map[string]string{
			"device_eui":     eui,
			"measurement_id": "m-1",
			"channel_index":  "1",
			"limit":          "100",
			"time_start":     "1700000000000",
			"time_end":       "1700086400000",
		},
	}, map[string]any{"list": []any{}})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, dataRoot(), []string{
		"data", "list",
		"--eui", eui,
		"--measurement-id", "m-1",
		"--channel-index", "1",
		"--limit", "100",
		"--time-start", "1700000000000",
		"--time-end", "1700086400000",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestDataList_MissingEui(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, dataRoot(), []string{"data", "list"})
	if err == nil {
		t.Fatal("expected error for missing --eui")
	}
}

func TestDataChart(t *testing.T) {
	const eui = "node-eui-1"
	srv := cmdtest.NewMockServer(t, cmdtest.RequestSpec{
		Method:     http.MethodGet,
		PathSuffix: enum.OpenApiUrlDataAggregateChart,
		Query: map[string]string{
			"device_eui":     eui,
			"measurement_id": "m-1",
			"channel_index":  "1",
			"interval":       "60",
			"time_start":     "1700000000000",
			"time_end":       "1700086400000",
		},
	}, []map[string]any{{"channel": 1}})
	defer srv.Close()

	cmdtest.SetupConfig(t, srv.URL)
	_, err := cmdtest.Execute(t, dataRoot(), []string{
		"data", "chart",
		"--eui", eui,
		"--measurement-id", "m-1",
		"--channel-index", "1",
		"--time-start", "1700000000000",
		"--time-end", "1700086400000",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestDataChart_MissingEui(t *testing.T) {
	cmdtest.SetupConfig(t, "http://127.0.0.1:1")
	_, err := cmdtest.Execute(t, dataRoot(), []string{"data", "chart"})
	if err == nil {
		t.Fatal("expected error for missing --eui")
	}
}
