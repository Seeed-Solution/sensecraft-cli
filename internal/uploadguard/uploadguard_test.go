package uploadguard

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
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

func TestAllowUploadBurstAndSustain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rate.json")
	SetStatePathForTest(path)

	for i := 0; i < BurstMax; i++ {
		if err := AllowUpload("test"); err != nil {
			t.Fatalf("burst slot %d: %v", i, err)
		}
	}
	if err := AllowUpload("test"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("want burst limit, got %v", err)
	}

	// Seed sustain-max events all older than the burst window.
	old := time.Now().Add(-(BurstWindow + time.Second)).UnixMilli()
	events := make([]int64, SustainMax)
	for i := range events {
		events[i] = old
	}
	if err := saveLedger(path, rateLedger{Events: events}); err != nil {
		t.Fatal(err)
	}
	if err := AllowUpload("test"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("want sustain limit, got %v", err)
	}
}
