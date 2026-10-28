package config

import (
	"testing"

	"sensecraft-cloud-terminal/internal/enum"
)

func TestResolveAPIHostUsesOverride(t *testing.T) {
	t.Setenv(enum.EnvNameDataHost, "https://cap.test.example/openapi/")
	got, err := WithResolvedHost(AccessApiConfig{RemoteType: enum.RemoteTypeGlobal})
	if err != nil {
		t.Fatal(err)
	}
	if got.Host == nil || *got.Host != "https://cap.test.example" {
		t.Fatalf("host=%v", got.Host)
	}
}

func TestDevelopProfileUsesGlobalDataHost(t *testing.T) {
	t.Setenv(enum.EnvNameDataHost, "")
	got, err := WithResolvedHost(AccessApiConfig{RemoteType: enum.RemoteTypeDevelop})
	if err != nil {
		t.Fatal(err)
	}
	if got.Host == nil || *got.Host != enum.DomainNameOpenApiGlobal {
		t.Fatalf("host=%v", got.Host)
	}
}

func TestResolveAPIHostRejectsBadOverride(t *testing.T) {
	t.Setenv(enum.EnvNameDataHost, "not a host")
	_, err := WithResolvedHost(AccessApiConfig{RemoteType: enum.RemoteTypeChina})
	if err == nil {
		t.Fatal("expected invalid host error")
	}
}
