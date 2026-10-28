package client

import (
	"net/http"
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
	if !strings.Contains(got, "auth init") {
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

func TestAPIError_AccessKeyRejected(t *testing.T) {
	err := APIError{Code: "11104", Msg: "AccessId or accessKey error"}
	got := err.Error()
	if strings.Contains(got, "cannot unmarshal") {
		t.Fatalf("leaked decode error: %q", got)
	}
	for _, want := range []string{"11104", "Access Key", "AccessId or accessKey error", "auth init"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}

func TestParseOpenAPIEnvelope_NumericAccessKeyError(t *testing.T) {
	raw := []byte(`{"serviceIp":"10.0.0.1","msg":"AccessId or accessKey error","code":11104,"serviceName":"sensecap"}`)
	_, err := parseOpenAPIEnvelope(http.StatusOK, raw)
	if err == nil {
		t.Fatal("expected error")
	}
	got := err.Error()
	if strings.Contains(got, "cannot unmarshal") || strings.Contains(got, "ListDeviceResponse") {
		t.Fatalf("leaked decode error: %s", got)
	}
	if !strings.Contains(got, "11104") || !strings.Contains(got, "Access Key") {
		t.Fatalf("got %s", got)
	}
}

func TestParseOpenAPIEnvelope_NumericSuccessReturnsData(t *testing.T) {
	raw := []byte(`{"code":0,"data":[{"device_eui":"abc"}]}`)
	body, err := parseOpenAPIEnvelope(http.StatusOK, raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body.Payload()), "abc") {
		t.Fatalf("data=%s", body.Payload())
	}
}

func TestParseOpenAPIEnvelope_StringErrorUnchanged(t *testing.T) {
	raw := []byte(`{"code":"11202","msg":"Invalid request parameters"}`)
	_, err := parseOpenAPIEnvelope(http.StatusOK, raw)
	if err == nil || err.Error() != "API error code=11202: Invalid request parameters" {
		t.Fatalf("got %v", err)
	}
}

func TestParseOpenAPIEnvelope_UnreadableBodyIncludesHTTPStatusAndOriginal(t *testing.T) {
	raw := []byte(`not-json`)
	_, err := parseOpenAPIEnvelope(http.StatusOK, raw)
	if err == nil {
		t.Fatal("expected error")
	}
	got := err.Error()
	if !strings.Contains(got, "HTTP 200") || !strings.Contains(got, "not-json") {
		t.Fatalf("got %s", got)
	}
	if strings.Contains(got, "cannot unmarshal") {
		t.Fatalf("leaked decode error: %s", got)
	}
}

func TestAPIBodyUnmarshalIncludesOriginalResponse(t *testing.T) {
	raw := []byte(`{"code":"0","data":{"unexpected":true}}`)
	body, err := parseOpenAPIEnvelope(http.StatusOK, raw)
	if err != nil {
		t.Fatal(err)
	}
	var list []struct{}
	err = body.Unmarshal(&list)
	if err == nil {
		t.Fatal("expected error")
	}
	got := err.Error()
	for _, want := range []string{"HTTP 200", `"code":"0"`, `"unexpected":true`} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, "cannot unmarshal") || strings.Contains(got, "ListDeviceResponse") {
		t.Fatalf("leaked decode error: %s", got)
	}
}

func TestAIAPIError_PermissionDeniedMessage(t *testing.T) {
	err := AIAPIError{Code: 11201, Msg: "You have no options permission"}
	got := err.Error()
	if strings.Contains(got, "You have no options permission") {
		t.Fatalf("want friendly message, got raw API msg: %q", got)
	}
	if !strings.Contains(got, "auth init") {
		t.Fatalf("want auth init hint, got %q", got)
	}
}
