package service

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"

	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/definition"
	"sensecraft-cloud-terminal/internal/enum"
)

type DeviceService struct {
	client *client.OpenApiClient
}

func GetDeviceService(ctx app.CmdContext) DeviceService {
	s := DeviceService{}
	s.client = client.NewOpenApiClient(ctx.Config, ctx.SandBox)
	return s
}

func (s DeviceService) ListDevice(deviceType, groupUuid string) (definition.ListDeviceResponse, error) {
	q := url.Values{}
	if deviceType != "" {
		q.Set("device_type", deviceType)
	}
	if groupUuid != "" {
		q.Set("group_uuid", groupUuid)
	}

	body, err := s.client.GetBody(enum.OpenApiUrlDeviceList, q)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			return definition.ListDeviceResponse{
				{DeviceEui: "SANDBOXEUI000001", DeviceName: "Sandbox Node"},
			}, nil
		}
		return nil, err
	}
	resp := definition.ListDeviceResponse{}
	if err := body.Unmarshal(&resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s DeviceService) ViewDevices(deviceEUIs []string, deviceType string) (json.RawMessage, error) {
	body := map[string]any{"device_euis": deviceEUIs}
	if deviceType != "" {
		t, err := strconv.Atoi(deviceType)
		if err != nil {
			return nil, err
		}
		body["device_type"] = t
	}
	return s.client.Post(enum.OpenApiUrlDeviceView, body)
}

func (s DeviceService) ListDeviceChannels(deviceEUIs []string) (json.RawMessage, error) {
	return s.client.Post(enum.OpenApiUrlDeviceChannels, map[string]any{
		"device_euis": deviceEUIs,
	})
}

func (s DeviceService) ViewDeviceRunningStatus(deviceEUIs []string) (json.RawMessage, error) {
	return s.client.Post(enum.OpenApiUrlDeviceRunningStatus, map[string]any{
		"device_euis": deviceEUIs,
	})
}

func (s DeviceService) BindDevice(body map[string]string) (json.RawMessage, error) {
	return s.client.Post(enum.OpenApiUrlDeviceBind, body)
}

func (s DeviceService) UnbindDevices(deviceEUIs []string, yes bool) (json.RawMessage, error) {
	if err := requireYes(yes); err != nil {
		return nil, err
	}
	return s.client.Post(enum.OpenApiUrlDeviceDelete, map[string]any{
		"device_euis": deviceEUIs,
	})
}

func (s DeviceService) ViewSimcard(iccid string) (json.RawMessage, error) {
	return s.client.Get(enum.OpenApiUrlDeviceSimcard, url.Values{"iccid": {iccid}})
}
