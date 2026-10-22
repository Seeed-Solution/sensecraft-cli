package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/enum"
)

// AIAPIError is a SenseCraft AI (/openapi) business error.
type AIAPIError struct {
	Code int
	Msg  string
}

func (e AIAPIError) Error() string {
	if e.Code == 11201 {
		return accessKeyPermissionDeniedMessage("11201")
	}
	if e.Msg != "" {
		return fmt.Sprintf("AI API error code=%d: %s", e.Code, e.Msg)
	}
	return fmt.Sprintf("AI API error code=%d", e.Code)
}

// AIApiClient calls SenseCraft AI HTTP APIs under {host}/openapi
// with HTTP Basic (key_id:access_key).
type AIApiClient struct {
	base        *BaseClient
	basicAuth   bool
	accessKeyOK bool
	sandbox     bool
	hostErr     error
}

// NewAIApiClientAt points AI calls at an explicit base URL (no /openapi suffix added).
// Tests use this to stand in for the SenseCraft host.
func NewAIApiClientAt(baseURL string, httpClient *http.Client) *AIApiClient {
	return &AIApiClient{base: NewBaseClient(baseURL, WithHTTPClient(httpClient))}
}

// NewAIApiClientAccessKey calls AI routes under /openapi with SenseCAP Access Key Basic auth.
// Host is SENSECRAFT_AI_HOST when set, otherwise the SenseCraft AI site for the remote type.
func NewAIApiClientAccessKey(cfg config.AccessApiConfig, sandbox bool) *AIApiClient {
	host, hostErr := enum.EffectiveAIHost(cfg.RemoteType)
	if hostErr != nil {
		host = enum.ResolveAIServerHost(cfg.RemoteType)
	}
	keyID := strings.TrimSpace(cfg.KeyId)
	accessKey := strings.TrimSpace(cfg.AccessKey)
	base := NewBaseClient(
		fmt.Sprintf("%s/openapi", strings.TrimSuffix(host, "/")),
		WithSandbox(sandbox),
		WithDefaultRequestOptions(WithBasicAuth(keyID, accessKey)),
	)
	return &AIApiClient{
		base:        base,
		basicAuth:   true,
		accessKeyOK: keyID != "" && accessKey != "",
		sandbox:     sandbox,
		hostErr:     hostErr,
	}
}

func (c *AIApiClient) ensureAccessKey() error {
	if c.hostErr != nil {
		return c.hostErr
	}
	if !c.basicAuth {
		return nil
	}
	if c.sandbox {
		return nil
	}
	if c.accessKeyOK {
		return nil
	}
	return fmt.Errorf("this command uses Access Key HTTP Basic → {ai-host}/openapi; run: sensecraft-cli auth init --kind sensecap")
}

func (c *AIApiClient) Base() *BaseClient {
	return c.base
}

func (c *AIApiClient) Get(path string, query url.Values) (json.RawMessage, error) {
	if err := c.ensureAccessKey(); err != nil {
		return nil, err
	}
	raw, _, err := c.base.Get(path, query)
	if err != nil {
		return nil, err
	}
	return parseAIEnvelope(raw)
}

func (c *AIApiClient) Post(path string, body any) (json.RawMessage, error) {
	if err := c.ensureAccessKey(); err != nil {
		return nil, err
	}
	raw, _, err := c.base.Do(http.MethodPost, path, nil, body)
	if err != nil {
		return nil, err
	}
	return parseAIEnvelope(raw)
}

func parseAIEnvelope(raw []byte) (json.RawMessage, error) {
	var envelope struct {
		Code json.RawMessage `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return raw, nil
	}
	code, ok := parseFlexibleCode(envelope.Code)
	if ok && code != 0 {
		return nil, AIAPIError{Code: code, Msg: envelope.Msg}
	}
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		return envelope.Data, nil
	}
	return raw, nil
}

func parseFlexibleCode(raw json.RawMessage) (int, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		n, err := strconv.Atoi(s)
		if err == nil {
			return n, true
		}
	}
	return 0, false
}
