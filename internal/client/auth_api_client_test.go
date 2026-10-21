package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"sensecraft-cloud-terminal/internal/version"
)

func TestAuthAPIClientSendsCLIVersion(t *testing.T) {
	var sawVersion, sawAccessID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawVersion = r.Header.Get("sensecraft-cli-version")
		sawAccessID = r.Header.Get("sensecraft-cli-accessid")
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewAuthAPIClient(srv.URL)
	c.httpClient = srv.Client()
	if _, err := c.PostJSON("/device/code", map[string]any{"a": 1}, "", nil); err != nil {
		t.Fatal(err)
	}
	if sawVersion != version.Version {
		t.Fatalf("sensecraft-cli-version=%q", sawVersion)
	}
	if sawAccessID != "" {
		t.Fatalf("auth request has no access id yet, got %q", sawAccessID)
	}
}
