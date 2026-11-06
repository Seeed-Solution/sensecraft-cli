package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"sensecraft-cloud-terminal/internal/version"
)

var ErrSandboxMode = errors.New("sandbox mode: no real request sent")

// HTTPError is a non-2xx HTTP response. Error() keeps the historical
// "HTTP <status>: <body>" form used by callers and tests.
type HTTPError struct {
	StatusCode int
	Body       []byte
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, string(e.Body))
}

// RequestOption adds per-request configuration.
type RequestOption func(*http.Request)

func WithBasicAuth(username, password string) RequestOption {
	return func(req *http.Request) {
		req.SetBasicAuth(username, password)
		if id := strings.TrimSpace(username); id != "" {
			req.Header.Set("sensecraft-cli-accessid", id)
		}
	}
}

func WithHeader(key, value string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// BaseClient is the shared HTTP client: it joins URLs, sends requests, and reads raw responses.
type BaseClient struct {
	baseURL     string
	httpClient  *http.Client
	sandbox     bool
	defaultOpts []RequestOption
}

type BaseClientOption func(*BaseClient)

func WithHTTPClient(c *http.Client) BaseClientOption {
	return func(bc *BaseClient) {
		if c != nil {
			bc.httpClient = c
		}
	}
}

func WithSandbox(enabled bool) BaseClientOption {
	return func(bc *BaseClient) {
		bc.sandbox = enabled
	}
}

func WithDefaultRequestOptions(opts ...RequestOption) BaseClientOption {
	return func(bc *BaseClient) {
		bc.defaultOpts = append(bc.defaultOpts, opts...)
	}
}

func NewBaseClient(baseURL string, opts ...BaseClientOption) *BaseClient {
	c := &BaseClient{
		baseURL: strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *BaseClient) BaseURL() string {
	return c.baseURL
}

func (c *BaseClient) Do(method, path string, query url.Values, body any, opts ...RequestOption) ([]byte, int, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, 0, err
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	if c.sandbox {
		_, _ = fmt.Fprintln(os.Stderr, fmt.Sprintf("sandBox====> method=%s url=%s body=%#v", method, u.String(), body))
		return nil, 0, ErrSandboxMode
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, u.String(), bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("sensecraft-cli-version", cliVersion())

	for _, opt := range c.defaultOpts {
		opt(req)
	}
	for _, opt := range opts {
		opt(req)
	}

	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, HTTPError{StatusCode: resp.StatusCode, Body: raw}
	}
	return raw, resp.StatusCode, nil
}

func (c *BaseClient) Get(path string, query url.Values, opts ...RequestOption) ([]byte, int, error) {
	return c.Do(http.MethodGet, path, query, nil, opts...)
}

func (c *BaseClient) Post(path string, body any, opts ...RequestOption) ([]byte, int, error) {
	return c.Do(http.MethodPost, path, nil, body, opts...)
}

func (c *BaseClient) PostQuery(path string, query url.Values, opts ...RequestOption) ([]byte, int, error) {
	return c.Do(http.MethodPost, path, query, nil, opts...)
}

func cliVersion() string {
	v := strings.TrimSpace(version.Version)
	if v == "" {
		v = "dev"
	}
	return v
}
