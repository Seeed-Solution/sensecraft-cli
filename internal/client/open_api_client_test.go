package client

import (
	"strings"
	"testing"
)

func TestAPIError_PermissionDeniedMessage(t *testing.T) {
	err := APIError{Code: OpenAPIPermissionDenied, Msg: "You have no options permission"}
	got := err.Error()
	if strings.Contains(got, "You have no options permission") {
		t.Fatalf("want friendly message, got raw API msg: %q", got)
	}
	if !strings.Contains(got, "read-write") {
		t.Fatalf("want read-write guidance, got %q", got)
	}
	if !strings.Contains(got, "auth init --kind sensecap") {
		t.Fatalf("want auth init hint, got %q", got)
	}
	if !strings.Contains(got, "11201") {
		t.Fatalf("want code retained, got %q", got)
	}
}

func TestAPIError_OtherCodesUnchanged(t *testing.T) {
	err := APIError{Code: "11202", Msg: "Invalid request parameters"}
	got := err.Error()
	want := "API error code=11202: Invalid request parameters"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAIAPIError_PermissionDeniedMessage(t *testing.T) {
	err := AIAPIError{Code: 11201, Msg: "You have no options permission"}
	got := err.Error()
	if strings.Contains(got, "You have no options permission") {
		t.Fatalf("want friendly message, got raw API msg: %q", got)
	}
	if !strings.Contains(got, "auth init --kind sensecap") {
		t.Fatalf("want auth init hint, got %q", got)
	}
}
