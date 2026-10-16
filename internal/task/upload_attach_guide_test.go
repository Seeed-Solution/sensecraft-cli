package task

import (
	"errors"
	"strings"
	"testing"

	"sensecraft-cloud-terminal/internal/service"
	"sensecraft-cloud-terminal/internal/ui"
)

func TestResolveModelUploadAssocRequiresIDOrUnlinked(t *testing.T) {
	// Non-interactive console (strings.Reader is not a TTY).
	console := ui.NewConsole(strings.NewReader(""), nil, nil)
	_, err := ResolveModelUploadAssoc(console, "", false, false)
	if !errors.Is(err, service.ErrAttachRequired) {
		t.Fatalf("want ErrAttachRequired, got %v", err)
	}

	got, err := ResolveModelUploadAssoc(console, "m1", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.ModelID != "m1" || !got.Yes || got.Unlinked {
		t.Fatalf("unexpected assoc: %+v", got)
	}

	got, err = ResolveModelUploadAssoc(console, "", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Unlinked || got.ModelID != "" {
		t.Fatalf("unexpected unlinked assoc: %+v", got)
	}

	_, err = ResolveModelUploadAssoc(console, "m1", true, true)
	if err == nil {
		t.Fatal("expected conflict between --id and --unlinked")
	}
}

func TestResolveFileUploadAssocGatesType1And2(t *testing.T) {
	console := ui.NewConsole(strings.NewReader(""), nil, nil)

	_, err := ResolveFileUploadAssoc(console, service.FileTypeFile, "", "", false, false, false)
	if !errors.Is(err, service.ErrAttachRequired) {
		t.Fatalf("type 2 without assoc: want ErrAttachRequired, got %v", err)
	}

	_, err = ResolveFileUploadAssoc(console, service.FileTypeImage, "", "", false, false, false)
	if !errors.Is(err, service.ErrAttachRequired) {
		t.Fatalf("type 1 without assoc: want ErrAttachRequired, got %v", err)
	}

	got, err := ResolveFileUploadAssoc(console, service.FileTypeFirmware, "", "", false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Unlinked {
		t.Fatalf("firmware should bypass attach gate: %+v", got)
	}

	got, err = ResolveFileUploadAssoc(console, service.FileTypeImage, "", "", true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !got.ForApp {
		t.Fatalf("want for-app: %+v", got)
	}
}
