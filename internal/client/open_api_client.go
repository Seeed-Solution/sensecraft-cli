package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/enum"
)

type APIError struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

// OpenAPIPermissionDenied is SenseCAP / OpenAPI business code for insufficient
// Access Key permission (typically a read-only key on a write operation).
const OpenAPIPermissionDenied = "11201"

func (e APIError) Error() string {
	if e.Code == OpenAPIPermissionDenied {
		return accessKeyPermissionDeniedMessage(e.Code)
	}
	if e.Msg != "" {
		return fmt.Sprintf("API error code=%s: %s", e.Code, e.Msg)
	}
	return fmt.Sprintf("API error code=%s", e.Code)
}

func accessKeyPermissionDeniedMessage(code string) string {
	return fmt.Sprintf(
		"Access Key lacks write permission (API code=%s); create or switch to a read-write key via: sensecraft-cli auth init --kind sensecap",
		code,
	)
}

// OpenApiClient 基于 BaseClient，封装 SenseCAP OpenAPI 的认证与响应 envelope 解析。
type OpenApiClient struct {
	base *BaseClient
}

func NewOpenApiClient(cfg config.AccessApiConfig, sandbox bool) *OpenApiClient {
	host := ""
	if cfg.Host != nil {
		host = strings.TrimSuffix(strings.TrimSpace(*cfg.Host), "/")
	}
	base := NewBaseClient(
		fmt.Sprintf("%s/openapi", host),
		WithSandbox(sandbox),
		WithDefaultRequestOptions(WithBasicAuth(cfg.KeyId, cfg.AccessKey)),
	)
	return &OpenApiClient{base: base}
}

func (c *OpenApiClient) Base() *BaseClient {
	return c.base
}

func (c *OpenApiClient) Get(path string, query url.Values) (json.RawMessage, error) {
	raw, err := c.base.Get(path, query)
	if err != nil {
		return nil, err
	}
	return parseOpenAPIEnvelope(raw)
}

func (c *OpenApiClient) Post(path string, body any) (json.RawMessage, error) {
	raw, err := c.base.Post(path, body)
	if err != nil {
		return nil, err
	}
	return parseOpenAPIEnvelope(raw)
}

func (c *OpenApiClient) PostQuery(path string, query url.Values) (json.RawMessage, error) {
	raw, err := c.base.PostQuery(path, query)
	if err != nil {
		return nil, err
	}
	return parseOpenAPIEnvelope(raw)
}

func (c *OpenApiClient) CreateDevelopmentKit(sku, devName string, deviceInfo any) (json.RawMessage, error) {
	body := map[string]any{"sku": sku}
	if devName != "" {
		body["dev_name"] = devName
	}
	if deviceInfo != nil {
		body["device_info"] = deviceInfo
	}
	return c.Post(enum.OpenApiUrlDeviceCreateDevelopmentKit, body)
}

func (c *OpenApiClient) ViewNodeKey(eui string) (json.RawMessage, error) {
	return c.Get(enum.OpenApiUrlDeviceViewNodeKey, url.Values{"node_eui": {eui}})
}

func parseOpenAPIEnvelope(raw []byte) (json.RawMessage, error) {
	var envelope struct {
		Code string          `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return raw, nil
	}
	if envelope.Code != "" && envelope.Code != "0" {
		return nil, APIError{Code: envelope.Code, Msg: envelope.Msg}
	}
	if len(envelope.Data) > 0 {
		return envelope.Data, nil
	}
	return raw, nil
}
