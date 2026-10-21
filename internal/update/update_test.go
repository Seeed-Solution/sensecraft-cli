package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"0.1.5", "0.1.4", true},
		{"0.1.4", "0.1.4", false},
		{"0.1.4", "0.2.0", false},
		{"v0.2.0", "0.1.9", true},
		{"0.1.5", "dev", true},
		{"dev", "0.1.4", false},
		{"0.1.4", "0.1.4-beta.1", true},
		{"0.1.4-beta.2", "0.1.4-beta.1", true},
		{"0.1.4-beta.1", "0.1.4", false},
		{"", "0.1.4", false},
	}
	for _, tc := range cases {
		if got := IsNewer(tc.latest, tc.current); got != tc.want {
			t.Errorf("IsNewer(%q, %q)=%v, want %v", tc.latest, tc.current, got, tc.want)
		}
	}
}

func TestShouldSkip(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("BUILD_NUMBER", "")
	t.Setenv("RUN_ID", "")
	t.Setenv("SENSECRAFT_CLI_NO_UPDATE_NOTIFIER", "")

	if ShouldSkip("0.1.4") {
		t.Fatal("release version should be checked")
	}
	if ShouldSkip("0.1.4-beta.1") {
		t.Fatal("npm prerelease should be checked")
	}
	for _, v := range []string{"", "dev", "DEV", "0.1.4-3-gabcdef1", "0.1.4-dirty"} {
		if !ShouldSkip(v) {
			t.Fatalf("ShouldSkip(%q)=false", v)
		}
	}

	t.Setenv("SENSECRAFT_CLI_NO_UPDATE_NOTIFIER", "1")
	if !ShouldSkip("0.1.4") {
		t.Fatal("opt-out should skip")
	}
	t.Setenv("SENSECRAFT_CLI_NO_UPDATE_NOTIFIER", "")
	t.Setenv("CI", "true")
	if !ShouldSkip("0.1.4") {
		t.Fatal("CI should skip")
	}
}

func TestFetchLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept=%q", r.Header.Get("Accept"))
		}
		_, _ = w.Write([]byte(`{"version":"0.2.0","name":"@seeed-studio/sensecraft-cli"}`))
	}))
	defer srv.Close()

	got, err := fetchLatest(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.2.0" {
		t.Fatalf("version=%q", got)
	}
}

func TestFetchLatestRejectsEmptyAndHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()
	if _, err := fetchLatest(context.Background(), srv.URL); err == nil {
		t.Fatal("expected HTTP error")
	}

	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"name":"x"}`))
	}))
	defer empty.Close()
	if _, err := fetchLatest(context.Background(), empty.URL); err == nil {
		t.Fatal("expected empty version error")
	}
}

func TestCachedNotice(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SENSECRAFT_CLI_PATH", dir)
	t.Setenv("CI", "")
	t.Setenv("BUILD_NUMBER", "")
	t.Setenv("RUN_ID", "")
	t.Setenv("SENSECRAFT_CLI_NO_UPDATE_NOTIFIER", "")

	if got := CachedNotice("0.1.4"); got != "" {
		t.Fatalf("missing cache notice=%q", got)
	}
	if err := rememberAt("0.1.5", time.Now()); err != nil {
		t.Fatal(err)
	}
	got := CachedNotice("0.1.4")
	want := Notice("0.1.4", "0.1.5")
	if got != want {
		t.Fatalf("notice=%q", got)
	}
	if CachedNotice("0.1.5") != "" {
		t.Fatal("current release should not notify")
	}
	if CachedNotice("dev") != "" {
		t.Fatal("dev build should not notify")
	}
	if _, err := os.Stat(filepath.Join(dir, "update-state.json")); err != nil {
		t.Fatal(err)
	}
	if err := markShownAt(time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := CachedNotice("0.1.4"); got != "" {
		t.Fatalf("same version should stay quiet for 24h, got %q", got)
	}
	if err := markShownAt(time.Now().Add(-25 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got := CachedNotice("0.1.4"); got != want {
		t.Fatalf("after 24h notice=%q", got)
	}
	if err := markShownAt(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := rememberAt("0.1.6", time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := CachedNotice("0.1.4"); got != Notice("0.1.4", "0.1.6") {
		t.Fatalf("newer release should notify again, got %q", got)
	}
}
