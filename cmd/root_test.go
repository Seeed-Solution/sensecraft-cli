package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestPublicRootOmitsTrainAndFleet(t *testing.T) {
	root := GetRootCmd()
	foundRecamera := false
	for _, c := range root.Commands() {
		if c.Name() == "recamera" {
			foundRecamera = true
			break
		}
	}
	if !foundRecamera {
		t.Fatal("public CLI must register recamera")
	}
	for _, name := range []string{"train", "fleet"} {
		for _, c := range root.Commands() {
			if c.Name() == name {
				t.Fatalf("public CLI must not register %q", name)
			}
		}
	}

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	help := buf.String()
	if strings.Contains(help, "\n  train ") || strings.Contains(help, "\n  fleet ") {
		t.Fatalf("public --help must not list train/fleet:\n%s", help)
	}
}

func TestTrainAndFleetAreUnknownCommands(t *testing.T) {
	for _, name := range []string{"train", "fleet"} {
		root := GetRootCmd()
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetArgs([]string{name, "--help"})
		if err := root.Execute(); err == nil {
			t.Fatalf("expected unknown command for %q", name)
		}
	}
}
