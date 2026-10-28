package auth

import (
	"bytes"
	"strings"
	"testing"
)

func TestAuthInitHelpDocumentsSplitFlow(t *testing.T) {
	cmd := newInitCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	help := buf.String()
	for _, want := range []string{"--no-wait", "--device-code", "--key-id", "--label", "--site", "--access-level"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help missing %s:\n%s", want, help)
		}
	}
}
