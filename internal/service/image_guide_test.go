package service

import (
	"strings"
	"testing"
)

func TestBuildImagePromptAsksForBrief(t *testing.T) {
	got := BuildImagePrompt("", "")
	if !strings.Contains(got.Prompt, "Ask for the brief") {
		t.Fatalf("missing ask-for-brief:\n%s", got.Prompt)
	}
	if !strings.Contains(got.Prompt, "SVG") || !strings.Contains(got.Prompt, "not only a title") {
		t.Fatalf("missing svg instructions:\n%s", got.Prompt)
	}
	for _, step := range []string{"4:3", "1200×900", "convert it on this machine to PNG", "ask whether it meets the requirements", "Upload this image", "Generate again", "Use the user's own image file"} {
		if !strings.Contains(got.Prompt, step) {
			t.Fatalf("missing %q\n%s", step, got.Prompt)
		}
	}
	if got.LocalTool != "" || got.Brief != "" {
		t.Fatalf("unexpected fields: %+v", got)
	}
}

func TestBuildImagePromptIncludesBriefAndTool(t *testing.T) {
	got := BuildImagePrompt("  自行车骑鹈鹕  ", "  sd-cli  ")
	if got.Brief != "自行车骑鹈鹕" || got.LocalTool != "sd-cli" {
		t.Fatalf("fields: %+v", got)
	}
	if !strings.Contains(got.Prompt, "Brief: 自行车骑鹈鹕") {
		t.Fatalf("brief missing:\n%s", got.Prompt)
	}
	if !strings.Contains(got.Prompt, "Local image tool: sd-cli") {
		t.Fatalf("tool missing:\n%s", got.Prompt)
	}
}
