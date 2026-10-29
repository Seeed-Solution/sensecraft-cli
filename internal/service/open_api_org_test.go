package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/enum"
)

func TestViewOrganizationReadsOrgIDFromEnvelope(t *testing.T) {
	const want int64 = 346466133785600
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openapi/view_organization" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"msg":"Success","data":{"orgId":346466133785600}}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv(enum.EnvNameDataHost, srv.URL)

	org, err := ViewOrganization(config.AccessApiConfig{
		RemoteType: enum.RemoteTypeGlobal,
		KeyId:      "key",
		AccessKey:  "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if org.OrgID != want {
		t.Fatalf("org id = %d, want %d", org.OrgID, want)
	}
}
