package service

import (
	"strings"
	"testing"
)

func TestValidateModelName(t *testing.T) {
	if err := validateModelName(""); err == nil {
		t.Fatal("empty")
	}
	if err := validateModelName("  ok  "); err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("中", 65)
	if err := validateModelName(long); err == nil || !strings.Contains(err.Error(), "64") {
		t.Fatalf("want max 64, got %v", err)
	}
}

func TestValidateAppName(t *testing.T) {
	if err := validateAppName(""); err == nil {
		t.Fatal("empty")
	}
	if err := validateAppName("ok"); err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("a", 61)
	if err := validateAppName(long); err == nil || !strings.Contains(err.Error(), "60") {
		t.Fatalf("want max 60, got %v", err)
	}
}
