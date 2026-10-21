package service

import (
	"testing"

	"sensecraft-cloud-terminal/internal/enum"
)

func TestDeviceAuthURLUsesAuthHost(t *testing.T) {
	t.Setenv(enum.EnvNameAuthHost, "https://account.test.example")
	got, err := getDeviceAuthURL(enum.RemoteTypeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://account.test.example/authapi/api/v1/auth/oauth/device"
	if got != want {
		t.Fatalf("device=%s", got)
	}
	token, err := getDeviceTokenURL(enum.RemoteTypeChina)
	if err != nil {
		t.Fatal(err)
	}
	wantToken := "https://account.test.example/authapi/api/v1/auth/oauth/device/token"
	if token != wantToken {
		t.Fatalf("token=%s", token)
	}
	base, err := portalAPIBase(enum.RemoteTypeChina)
	if err != nil {
		t.Fatal(err)
	}
	if base != "https://account.test.example/authapi" {
		t.Fatalf("portal=%s", base)
	}
}

func TestDeviceAuthURLDefaults(t *testing.T) {
	t.Setenv(enum.EnvNameAuthHost, "")
	got, err := getDeviceAuthURL(enum.RemoteTypeChina)
	if err != nil {
		t.Fatal(err)
	}
	if got != enum.DeviceFlowOauth2China {
		t.Fatalf("got %s", got)
	}
	got, err = getDeviceAuthURL(enum.RemoteTypeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	if got != enum.DeviceFlowOauth2Global {
		t.Fatalf("got %s", got)
	}
}

func TestDeviceAuthURLRejectsBadHost(t *testing.T) {
	t.Setenv(enum.EnvNameAuthHost, "ftp://account.test.example")
	if _, err := getDeviceAuthURL(enum.RemoteTypeGlobal); err == nil {
		t.Fatal("expected invalid host error")
	}
}
