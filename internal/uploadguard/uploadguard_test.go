package uploadguard

import (
	"errors"
	"strings"
	"testing"
)

func TestStoredFileName(t *testing.T) {
	for _, tt := range []struct {
		name string
		want string
	}{
		{"", "muse-gadget.bin"},
		{"snake-muse-gadget.bin", "snake-muse-gadget.bin"},
		{"my image.bin", "my image.bin"},
		{"../firmware.bin", ""},
		{`folder\firmware.bin`, ""},
		{"/firmware.bin", ""},
		{"..", ""},
		{" ", ""},
		{"file\n.bin", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StoredFileName("/build/muse-gadget.bin", tt.name)
			if tt.want == "" {
				if err == nil {
					t.Fatal("expected invalid stored name")
				}
			} else if err != nil || got != tt.want {
				t.Fatalf("got %q, error=%v; want %q", got, err, tt.want)
			}
		})
	}
	if got, err := StoredFileName("/build/my firmware.bin", ""); err != nil || got != "myfirmware.bin" {
		t.Fatalf("default naming changed: %q, %v", got, err)
	}
}

func TestValidateOSSUploadSizeRejectsEmptyAndOversize(t *testing.T) {
	if err := ValidateOSSUploadSize(2, 0); !errors.Is(err, ErrEmptyUpload) {
		t.Fatalf("want ErrEmptyUpload, got %v", err)
	}
	if err := ValidateOSSUploadSize(2, MaxNonImageUploadBytes); err != nil {
		t.Fatal(err)
	}
	if err := ValidateOSSUploadSize(2, MaxNonImageUploadBytes+1); err == nil {
		t.Fatal("want oversize error")
	}
	if err := ValidateOSSUploadSize(1, 3*1024*1024); err != nil {
		t.Fatal(err)
	}
	if err := ValidateOSSUploadSize(1, MaxAppImageBytes); err != nil {
		t.Fatal(err)
	}
	err := ValidateOSSUploadSize(1, MaxAppImageBytes+1)
	if err == nil || !strings.Contains(err.Error(), "5 MB") {
		t.Fatalf("want 5 MB image cap, got %v", err)
	}
}
