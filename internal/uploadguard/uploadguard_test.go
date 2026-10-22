package uploadguard

import (
	"errors"
	"strings"
	"testing"
)

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
