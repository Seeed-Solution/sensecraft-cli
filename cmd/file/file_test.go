package file

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/cmdtest"
)

func TestFileUploadStoredNameAndWarning(t *testing.T) {
	cmdtest.SetupConfig(t, "https://unused.invalid")
	path := filepath.Join(t.TempDir(), "muse-gadget.bin")
	if err := os.WriteFile(path, []byte("application-image"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		fileType string
		name     string
		want     string
	}{
		{"4", "", "muse-gadget.bin"},
		{"4", "snake-muse-gadget.bin", "snake-muse-gadget.bin"},
		{"3", "project.bin", "project.bin"},
	} {
		t.Run(tt.fileType+tt.name, func(t *testing.T) {
			args := []string{"--path", path, "--type", tt.fileType, "--sandbox"}
			if tt.name != "" {
				args = append(args, "--name", tt.name)
			}
			cmd := newFileUploadCmd()
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)
			cmd.SetArgs(args)
			oldStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			os.Stdout = w
			execErr := cmd.Execute()
			os.Stdout = oldStdout
			_ = w.Close()
			defer r.Close()
			var result map[string]any
			if err := json.NewDecoder(r).Decode(&result); err != nil {
				t.Fatal(err)
			}
			if execErr != nil || result["file_name"] != tt.want {
				t.Fatalf("file_name=%v, error=%v", result["file_name"], execErr)
			}
			warning := stderr.String()
			if tt.fileType == "4" && tt.name == "" {
				for _, text := range []string{"Warning:", "replace", "same name", "organisation", "--name <project>-muse-gadget.bin"} {
					if !strings.Contains(warning, text) {
						t.Fatalf("warning missing %q", text)
					}
				}
			} else if warning != "" {
				t.Fatalf("unexpected warning: %s", warning)
			}
		})
	}
}
