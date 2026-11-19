package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/enum"
	"sensecraft-cloud-terminal/internal/version"
)

func TestAIApiClientAccessKeyUsesBasicWithoutAppID(t *testing.T) {
	var sawAuth string
	var sawAppID string
	var sawVersion string
	var sawAccessID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		sawAppID = r.URL.Query().Get("appid")
		sawVersion = r.Header.Get("sensecraft-cli-version")
		sawAccessID = r.Header.Get("sensecraft-cli-accessid")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"ok": true},
		})
	}))
	t.Cleanup(srv.Close)

	c := &AIApiClient{
		base: NewBaseClient(srv.URL, WithHTTPClient(srv.Client()), WithDefaultRequestOptions(
			WithBasicAuth("kid", "secret"),
		)),
		basicAuth:   true,
		accessKeyOK: true,
	}
	raw, err := c.Get("/model/list_custom_model", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"ok":true`) {
		t.Fatalf("raw=%s", raw)
	}
	if !strings.HasPrefix(sawAuth, "Basic ") {
		t.Fatalf("auth=%q", sawAuth)
	}
	if sawAppID != "" {
		t.Fatalf("appid should be empty, got %q", sawAppID)
	}
	if sawVersion != version.Version {
		t.Fatalf("sensecraft-cli-version=%q", sawVersion)
	}
	if sawAccessID != "kid" {
		t.Fatalf("sensecraft-cli-accessid=%q", sawAccessID)
	}
}

func TestAIApiClientAccessKeyRejectsMissingCredentials(t *testing.T) {
	c := NewAIApiClientAccessKey(config.AccessApiConfig{RemoteType: "Global"}, false)
	_, err := c.Get("/model/list_model", nil)
	if err == nil || !strings.Contains(err.Error(), "Access Key") {
		t.Fatalf("want access key error, got %v", err)
	}
}

func TestAIApiClientPublicOmitsAuthorization(t *testing.T) {
	var sawAuth string
	var sawPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		sawPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "0",
			"data": map[string]any{"list": []any{}},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewAIApiClientAt(srv.URL+"/openapi", srv.Client())
	raw, err := c.Get("/activity/list", nil)
	if err != nil {
		t.Fatal(err)
	}
	if sawAuth != "" {
		t.Fatalf("auth=%q", sawAuth)
	}
	if sawPath != "/openapi/activity/list" {
		t.Fatalf("path=%s", sawPath)
	}
	if !strings.Contains(string(raw), `"list"`) {
		t.Fatalf("raw=%s", raw)
	}
}

func TestAIApiClientAccessKeyUsesHostOverride(t *testing.T) {
	t.Setenv(enum.EnvNameAIHost, "https://test-api-gateway.seeed.cc")
	c := NewAIApiClientAccessKey(config.AccessApiConfig{
		RemoteType: "Global",
		KeyId:      "kid",
		AccessKey:  "secret",
	}, false)
	if c.base.baseURL != "https://test-api-gateway.seeed.cc/openapi" {
		t.Fatalf("base=%s", c.base.baseURL)
	}
	if c.hostErr != nil {
		t.Fatal(c.hostErr)
	}
}

func TestAIApiClientAccessKeyRejectsUnofficialHost(t *testing.T) {
	t.Setenv(enum.EnvNameAIHost, "https://ai.test.example")
	c := NewAIApiClientAccessKey(config.AccessApiConfig{
		RemoteType: "Global",
		KeyId:      "kid",
		AccessKey:  "secret",
	}, false)
	if c.hostErr == nil {
		t.Fatal("expected host error")
	}
	if _, err := c.Get("/model/list_model", nil); err == nil {
		t.Fatal("expected request to fail before using the key")
	}
}
