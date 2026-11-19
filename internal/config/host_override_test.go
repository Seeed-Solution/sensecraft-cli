package config

import (
	"testing"

	"sensecraft-cloud-terminal/internal/enum"
)

func TestResolveAPIHostUsesOverride(t *testing.T) {
	t.Setenv(enum.EnvNameDataHost, "https://sensecap-test.seeed.cc/openapi/")
	got, err := WithResolvedHost(AccessApiConfig{RemoteType: enum.RemoteTypeGlobal})
	if err != nil {
		t.Fatal(err)
	}
	if got.Host == nil || *got.Host != "https://sensecap-test.seeed.cc" {
		t.Fatalf("host=%v", got.Host)
	}
}

func TestResolveAPIHostRejectsUnofficialOverride(t *testing.T) {
	t.Setenv(enum.EnvNameDataHost, "https://cap.test.example")
	if _, err := WithResolvedHost(AccessApiConfig{RemoteType: enum.RemoteTypeGlobal}); err == nil {
		t.Fatal("expected foreign host rejection")
	}
	t.Setenv(enum.EnvNameDataHost, "http://sensecap.seeed.cc")
	if _, err := WithResolvedHost(AccessApiConfig{RemoteType: enum.RemoteTypeGlobal}); err == nil {
		t.Fatal("expected http rejection")
	}
}

func TestDevelopProfileUsesGlobalDataHost(t *testing.T) {
	t.Setenv(enum.EnvNameDataHost, "")
	got, err := WithResolvedHost(AccessApiConfig{RemoteType: "Develop"})
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
