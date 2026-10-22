package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
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

// OpenAPIAccessKeyRejected is SenseCAP / OpenAPI business code when the
// Key ID or Access Key is not accepted.
const OpenAPIAccessKeyRejected = "11104"

func (e APIError) Error() string {
	switch e.Code {
	case OpenAPIPermissionDenied:
		return accessKeyPermissionDeniedMessage(e.Code)
	case OpenAPIAccessKeyRejected:
		return accessKeyRejectedMessage(e)
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

func accessKeyRejectedMessage(e APIError) string {
	if e.Msg != "" {
		return fmt.Sprintf("Access Key was rejected (API code=%s: %s). Check the Key ID, Access Key, and site, then run: sensecraft-cli auth init --kind sensecap", e.Code, e.Msg)
	}
	return fmt.Sprintf("Access Key was rejected (API code=%s). Check the Key ID, Access Key, and site, then run: sensecraft-cli auth init --kind sensecap", e.Code)
}

// DecodeError is a 2xx response the CLI could not turn into its expected value.
// StatusCode is the HTTP status. Body is the server's original payload, which
// may contain a business code or may be some other text.
type DecodeError struct {
	StatusCode int
	Body       []byte
}

func (e DecodeError) Error() string {
	body := strings.TrimSpace(string(e.Body))
	if body == "" {
		return fmt.Sprintf("could not read server response (HTTP %d)", e.StatusCode)
	}
	return fmt.Sprintf("could not read server response (HTTP %d): %s", e.StatusCode, body)
}

// APIBody is one OpenAPI response. Raw is the original payload. Data is the
// envelope data field when the envelope itself could be read.
type APIBody struct {
	Status int
	Raw    []byte
	Data   json.RawMessage
}

// Payload is the JSON a caller should decode. It is the envelope data when
// present, otherwise the original body.
func (b APIBody) Payload() json.RawMessage {
	if len(b.Data) > 0 {
		return b.Data
	}
	return b.Raw
}

// Unmarshal decodes Payload into dest. On failure the error includes the HTTP
// status and the original server body.
func (b APIBody) Unmarshal(dest any) error {
	if err := json.Unmarshal(b.Payload(), dest); err != nil {
		return DecodeError{StatusCode: b.Status, Body: b.Raw}
	}
	return nil
}

// OpenApiClient wraps BaseClient with SenseCAP OpenAPI auth and response-envelope parsing.
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
	body, err := c.GetBody(path, query)
	if err != nil {
		return nil, err
	}
	return body.Payload(), nil
}

func (c *OpenApiClient) GetBody(path string, query url.Values) (APIBody, error) {
	raw, status, err := c.base.Get(path, query)
	if err != nil {
		return APIBody{}, err
	}
	return parseOpenAPIEnvelope(status, raw)
}

func (c *OpenApiClient) Post(path string, body any) (json.RawMessage, error) {
	parsed, err := c.PostBody(path, body)
	if err != nil {
		return nil, err
	}
	return parsed.Payload(), nil
}

func (c *OpenApiClient) PostBody(path string, body any) (APIBody, error) {
	raw, status, err := c.base.Post(path, body)
	if err != nil {
		return APIBody{}, err
	}
	return parseOpenAPIEnvelope(status, raw)
}

func (c *OpenApiClient) PostQuery(path string, query url.Values) (json.RawMessage, error) {
	parsed, err := c.PostQueryBody(path, query)
	if err != nil {
		return nil, err
	}
	return parsed.Payload(), nil
}

func (c *OpenApiClient) PostQueryBody(path string, query url.Values) (APIBody, error) {
	raw, status, err := c.base.PostQuery(path, query)
	if err != nil {
		return APIBody{}, err
	}
	return parseOpenAPIEnvelope(status, raw)
}

func (c *OpenApiClient) CreateDevelopmentKit(sku, devName string, deviceInfo any) (APIBody, error) {
	body := map[string]any{"sku": sku}
	if devName != "" {
		body["dev_name"] = devName
	}
	if deviceInfo != nil {
		body["device_info"] = deviceInfo
	}
	return c.PostBody(enum.OpenApiUrlDeviceCreateDevelopmentKit, body)
}

func (c *OpenApiClient) ViewNodeKey(eui string) (APIBody, error) {
	return c.GetBody(enum.OpenApiUrlDeviceViewNodeKey, url.Values{"node_eui": {eui}})
}

func parseOpenAPIEnvelope(status int, raw []byte) (APIBody, error) {
	parsed := APIBody{Status: status, Raw: append([]byte(nil), raw...)}
	var envelope struct {
		Code json.RawMessage `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return APIBody{}, DecodeError{StatusCode: status, Body: parsed.Raw}
	}
	code, ok := parseFlexibleCode(envelope.Code)
	if !ok {
		if len(strings.TrimSpace(string(envelope.Code))) == 0 {
			parsed.Data = parsed.Raw
			return parsed, nil
		}
		return APIBody{}, DecodeError{StatusCode: status, Body: parsed.Raw}
	}
	if code != 0 {
		return APIBody{}, APIError{Code: strconv.Itoa(code), Msg: envelope.Msg}
	}
	if len(envelope.Data) > 0 {
		parsed.Data = envelope.Data
		return parsed, nil
	}
	parsed.Data = parsed.Raw
	return parsed, nil
}
