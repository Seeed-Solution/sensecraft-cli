package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/config"
)

func TestAIApiClientAccessKeyUsesBasicWithoutAppID(t *testing.T) {
	var sawAuth string
	var sawAppID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		sawAppID = r.URL.Query().Get("appid")
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
}

func TestAIApiClientAccessKeyRejectsMissingCredentials(t *testing.T) {
	c := NewAIApiClientAccessKey(config.AccessApiConfig{RemoteType: "Global"}, false)
	_, err := c.Get("/model/list_model", nil)
	if err == nil || !strings.Contains(err.Error(), "Access Key") {
		t.Fatalf("want access key error, got %v", err)
	}
}
