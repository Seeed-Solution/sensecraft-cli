package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/enum"
)

type CloudService struct {
	client  *client.AIApiClient
	sandbox bool
}

func GetCloudService(ctx app.AICmdContext) CloudService {
	return CloudService{client: newAIHTTPClient(ctx), sandbox: ctx.SandBox}
}

type CloudDeviceListParams struct {
	UniformType int
	Name        string
	Page        int
	Length      int
	Adapteds    string
}

func (s CloudService) ListDevices(p CloudDeviceListParams) (json.RawMessage, error) {
	page := p.Page
	if page <= 0 {
		page = 1
	}
	length := p.Length
	if length <= 0 {
		length = 20
	}
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("length", strconv.Itoa(length))
	if p.UniformType > 0 {
		q.Set("uniform_type", strconv.Itoa(p.UniformType))
	}
	if strings.TrimSpace(p.Name) != "" {
		q.Set("device_name", strings.TrimSpace(p.Name))
	}
	if strings.TrimSpace(p.Adapteds) != "" {
		q.Set("adapteds", strings.TrimSpace(p.Adapteds))
	}
	data, err := s.client.Get(enum.AIApiPathDeviceListNode, q)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			return sandboxCloudDeviceList(p), nil
		}
		return nil, err
	}
	return data, nil
}

type CloudDeviceViewParams struct {
	EUI string
}

func (s CloudService) ViewDevice(p CloudDeviceViewParams) (json.RawMessage, error) {
	if strings.TrimSpace(p.EUI) == "" {
		return nil, fmt.Errorf("--eui is required")
	}
	q := url.Values{}
	q.Set("dev_eui", p.EUI)
	data, err := s.client.Get(enum.AIApiPathDeviceView, q)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			raw, _ := json.Marshal(map[string]any{
				"dev_eui":       p.EUI,
				"dev_name":      "Sandbox Jetson",
				"online_status": 1,
			})
			return raw, nil
		}
		return nil, err
	}
	return data, nil
}

type CloudDeployParams struct {
	EUI      string
	ModelID  string
	FromJSON string // stream setting object; optional minimal
	Yes      bool
}

func (s CloudService) DeployModel(p CloudDeployParams) (json.RawMessage, error) {
	if err := requireYes(p.Yes); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.EUI) == "" {
		return nil, fmt.Errorf("--eui is required")
	}
	if strings.TrimSpace(p.ModelID) == "" {
		return nil, fmt.Errorf("--model-id is required")
	}
	setting := map[string]any{}
	if strings.TrimSpace(p.FromJSON) != "" {
		obj, err := LoadJSONObject(p.FromJSON)
		if err != nil {
			return nil, err
		}
		setting = obj
	}
	setting["model_id"] = p.ModelID
	body := map[string]any{
		"device_eui": p.EUI,
		"model_id":   p.ModelID,
		"setting":    setting,
	}
	data, err := s.client.Post(enum.AIApiPathDeviceUpdateStream, body)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			raw, _ := json.Marshal(map[string]any{
				"request_id": "sandbox-deploy-req",
				"device_eui": p.EUI,
				"model_id":   p.ModelID,
				"ok":         true,
			})
			return raw, nil
		}
		return nil, err
	}
	return data, nil
}

type CloudOrderParams struct {
	RequestID  string
	DeviceUUID string // Console deploy.vue also sends device_uuid with request_id
}

func (s CloudService) GetOrderRecord(p CloudOrderParams) (json.RawMessage, error) {
	if strings.TrimSpace(p.RequestID) == "" {
		return nil, fmt.Errorf("--request-id is required")
	}
	q := url.Values{}
	q.Set("request_id", p.RequestID)
	if strings.TrimSpace(p.DeviceUUID) != "" {
		q.Set("device_uuid", strings.TrimSpace(p.DeviceUUID))
	}
	data, err := s.client.Get(enum.AIApiPathDeviceOrderRecord, q)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			raw, _ := json.Marshal(map[string]any{
				"request_id":     p.RequestID,
				"responseStatus": 200,
				"execStatus":     200,
			})
			return raw, nil
		}
		return nil, err
	}
	return data, nil
}

func sandboxCloudDeviceList(p CloudDeviceListParams) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"list": []map[string]any{
			{
				"dev_eui":       "SANDBOXEUI000001",
				"dev_name":      "Sandbox Cloud Device",
				"uniform_type":  p.UniformType,
				"online_status": 1,
			},
		},
		"count": 1,
	})
	return raw
}
