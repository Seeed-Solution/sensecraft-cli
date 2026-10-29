package cmdtest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/enum"

	"github.com/spf13/cobra"
)

// RequestSpec describes the HTTP request a mock OpenAPI server should receive.
type RequestSpec struct {
	Method     string
	PathSuffix string
	Query      map[string]string
	Body       map[string]any
}

// SetupConfig writes a temporary config that points at the mock server.
func SetupConfig(t *testing.T, serverURL string) {
	t.Helper()

	dir := t.TempDir()
	t.Setenv(enum.EnvNameConfigPath, dir)
	t.Setenv(enum.EnvNameConfigFile, "")
	config.ResetConfigCache()

	host := strings.TrimSuffix(serverURL, "/")
	cfg := &config.CliConfig{
		UseApi: "test-key-id",
		AppConfigList: []config.AccessApiConfig{
			{
				RemoteType: enum.RemoteTypePrivate,
				Host:       &host,
				KeyId:      "test-key-id",
				AccessKey:  "test-access-key",
			},
		},
	}
	path, err := config.ResolveConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := config.SaveCliConfigTo(path, cfg); err != nil {
		t.Fatal(err)
	}
	config.ResetConfigCache()
}

// NewMockServer starts a test HTTP server that checks the request and returns an OpenAPI envelope.
func NewMockServer(t *testing.T, spec RequestSpec, responseData any) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if spec.Method != "" && r.Method != spec.Method {
			t.Errorf("HTTP method: got %s, want %s", r.Method, spec.Method)
		}

		wantPath := "/openapi" + spec.PathSuffix
		if r.URL.Path != wantPath {
			t.Errorf("HTTP path: got %s, want %s", r.URL.Path, wantPath)
		}

		for key, want := range spec.Query {
			if got := r.URL.Query().Get(key); got != want {
				t.Errorf("query %s: got %q, want %q", key, got, want)
			}
		}

		if spec.Body != nil {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			for key, want := range spec.Body {
				got, ok := body[key]
				if !ok {
					t.Errorf("request body missing key %q", key)
					continue
				}
				if !jsonValueEqual(got, want) {
					t.Errorf("request body %s: got %#v, want %#v", key, got, want)
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "0",
			"data": responseData,
		})
	}))
}

func jsonValueEqual(got, want any) bool {
	gb, err1 := json.Marshal(got)
	wb, err2 := json.Marshal(want)
	if err1 != nil || err2 != nil {
		return got == want
	}
	return string(gb) == string(wb)
}

// Execute runs a cobra command with stdout replaced and returns what it printed.
func Execute(t *testing.T, root *cobra.Command, args []string) (stdout string, err error) {
	t.Helper()

	root.SetArgs(args)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatal(pipeErr)
	}
	os.Stdout = w

	execErr := root.Execute()

	if closeErr := w.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatal(copyErr)
	}
	return buf.String(), execErr
}

// AssertJSONOutput checks that stdout JSON contains the given field values.
func AssertJSONOutput(t *testing.T, stdout string, checks map[string]any) {
	t.Helper()

	var out map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &out); err != nil {
		t.Fatalf("parse stdout JSON: %v\nstdout: %s", err, stdout)
	}
	for key, want := range checks {
		got, ok := out[key]
		if !ok {
			t.Fatalf("stdout JSON missing key %q, got: %s", key, stdout)
		}
		if !jsonValueEqual(got, want) {
			t.Fatalf("stdout JSON %s: got %#v, want %#v", key, got, want)
		}
	}
}
