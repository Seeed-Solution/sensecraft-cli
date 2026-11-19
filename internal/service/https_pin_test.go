package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDownloadRejectsHTTP(t *testing.T) {
	_, err := downloadURLToFile("http://cdn.example/a.bin", filepath.Join(t.TempDir(), "a.bin"))
	if err == nil || !strings.Contains(err.Error(), "only https") {
		t.Fatalf("expected https rejection, got %v", err)
	}
}

func TestDownloadFollowsSameHostRedirect(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start.bin" {
			http.Redirect(w, r, srv.URL+"/final.bin", http.StatusTemporaryRedirect)
			return
		}
		_, _ = w.Write([]byte("model-bytes"))
	}))
	t.Cleanup(srv.Close)
	withTestHTTPSClient(t, srv)

	out := filepath.Join(t.TempDir(), "model.bin")
	n, err := downloadURLToFile(srv.URL+"/start.bin", out)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len("model-bytes")) {
		t.Fatalf("bytes %d", n)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "model-bytes" {
		t.Fatalf("content %q", got)
	}
}

func TestDownloadRefusesCrossHostRedirect(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://example.invalid/x.bin", http.StatusTemporaryRedirect)
	}))
	t.Cleanup(srv.Close)
	withTestHTTPSClient(t, srv)

	out := filepath.Join(t.TempDir(), "model.bin")
	_, err := downloadURLToFile(srv.URL+"/model.bin", out)
	if err == nil || !strings.Contains(err.Error(), "refusing redirect") {
		t.Fatalf("expected redirect rejection, got %v", err)
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Fatalf("redirect wrote %s", out)
	}
}

func TestRefuseCrossHostRedirect(t *testing.T) {
	orig, err := http.NewRequest(http.MethodGet, "https://oss.example/a", nil)
	if err != nil {
		t.Fatal(err)
	}
	other, err := http.NewRequest(http.MethodGet, "https://evil.example/a", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := refuseCrossHostRedirect(other, []*http.Request{orig}); err == nil {
		t.Fatal("expected a different host to be refused")
	}
	same, err := http.NewRequest(http.MethodGet, "https://oss.example:443/b", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := refuseCrossHostRedirect(same, []*http.Request{orig}); err != nil {
		t.Fatal(err)
	}
	plain, err := http.NewRequest(http.MethodGet, "http://oss.example/b", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := refuseCrossHostRedirect(plain, []*http.Request{orig}); err == nil {
		t.Fatal("expected http redirect to be refused")
	}
}

func withTestHTTPSClient(t *testing.T, srv *httptest.Server) {
	t.Helper()
	prev := externalHTTPClient
	externalHTTPClient = func(timeout time.Duration) *http.Client {
		c := srv.Client()
		c.Timeout = timeout
		c.CheckRedirect = refuseCrossHostRedirect
		return c
	}
	t.Cleanup(func() { externalHTTPClient = prev })
}
