package uploadguard

import (
	"errors"
	"testing"
)

func TestValidateOSSUploadSizeRejectsEmptyAndOversize(t *testing.T) {
	if err := ValidateOSSUploadSize(2, 0, false); !errors.Is(err, ErrEmptyUpload) {
		t.Fatalf("want ErrEmptyUpload, got %v", err)
	}
	if err := ValidateOSSUploadSize(2, MaxNonImageUploadBytes, false); err != nil {
		t.Fatal(err)
	}
	if err := ValidateOSSUploadSize(2, MaxNonImageUploadBytes+1, false); err == nil {
		t.Fatal("want oversize error")
	}
	if err := ValidateOSSUploadSize(1, MaxModelImageBytes+1, false); err == nil {
		t.Fatal("want model image oversize")
	}
	if err := ValidateOSSUploadSize(1, MaxAppImageBytes, true); err != nil {
		t.Fatal(err)
	}
}
