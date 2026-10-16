package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// AuthAPIClient 用户中心 authapi 客户端（Device Flow + 需 Bearer 的用户 API）。
type AuthAPIClient struct {
	base       *BaseClient
	httpClient *http.Client
}

func NewAuthAPIClient(baseURL string) *AuthAPIClient {
	return &AuthAPIClient{
		base: NewBaseClient(baseURL),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type authEnvelope[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type AuthAPIError struct {
	Code int
	Msg  string
}

func (e AuthAPIError) Error() string {
	return fmt.Sprintf("authapi error code=%d: %s", e.Code, e.Msg)
}

func (c *AuthAPIClient) PostJSON(path string, body any, bearerToken string, out any) (int, error) {
	return c.doJSON(http.MethodPost, path, nil, body, bearerToken, out)
}

func (c *AuthAPIClient) GetJSON(path string, query url.Values, bearerToken string, out any) (int, error) {
	return c.doJSON(http.MethodGet, path, query, nil, bearerToken, out)
}

func (c *AuthAPIClient) PostAbsolute(urlStr string, body any, bearerToken string, out any) (int, error) {
	return c.doAbsolute(http.MethodPost, urlStr, body, bearerToken, out)
}

func (c *AuthAPIClient) doJSON(method, path string, query url.Values, body any, bearerToken string, out any) (int, error) {
	u, err := url.Parse(c.base.BaseURL() + path)
	if err != nil {
		return 0, err
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}
	return c.doAbsolute(method, u.String(), body, bearerToken, out)
}

func (c *AuthAPIClient) doAbsolute(method, urlStr string, body any, bearerToken string, out any) (int, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, urlStr, bodyReader)
	if err != nil {
		return 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", bearerToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var envelope authEnvelope[json.RawMessage]
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return 0, fmt.Errorf("parse response: %w", err)
	}
	if envelope.Code != 0 {
		return envelope.Code, AuthAPIError{Code: envelope.Code, Msg: envelope.Msg}
	}
	if out != nil && len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return 0, fmt.Errorf("parse data: %w", err)
		}
	}
	return 0, nil
}
