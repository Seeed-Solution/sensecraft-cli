package service

import (
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/utils"
)

func sandboxSenseCAPServiceClients(t *testing.T) (DeviceService, GroupService, DataService) {
	t.Helper()
	cfg := config.AccessApiConfig{
		KeyId:      "sandbox",
		AccessKey:  "sandbox",
		RemoteType: "Global",
		Host:       utils.String2Ptr("https://example.invalid"),
	}
	c := client.NewOpenApiClient(cfg, true)
	return DeviceService{client: c}, GroupService{client: c}, DataService{client: c}
}

func TestSenseCAPListSandbox(t *testing.T) {
	dev, grp, data := sandboxSenseCAPServiceClients(t)

	list, err := dev.ListDevice("2", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].DeviceEui != "SANDBOXEUI000001" {
		t.Fatalf("devices=%#v", list)
	}

	raw, err := grp.ListGroups()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "sandbox-group-1") {
		t.Fatalf("groups=%s", raw)
	}

	raw, err = data.ViewLatestTelemetry(TelemetryQuery{DeviceEUI: "SANDBOXEUI000001"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "4097") {
		t.Fatalf("latest=%s", raw)
	}
}
