package enum

import "testing"

func TestNormalizeOrigin(t *testing.T) {
	got, err := NormalizeOrigin("https://ai.test.example/openapi/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://ai.test.example" {
		t.Fatalf("got %s", got)
	}
	got, err = NormalizeOrigin("https://account.test.example/authapi/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://account.test.example" {
		t.Fatalf("authapi got %s", got)
	}
	if _, err := NormalizeOrigin("ftp://ai.test.example"); err == nil {
		t.Fatal("expected scheme rejection")
	}
	if _, err := NormalizeOrigin("https://ai.test.example/extra"); err == nil {
		t.Fatal("expected path rejection")
	}
}

func TestHostOverrideBlank(t *testing.T) {
	t.Setenv(EnvNameAIHost, "  ")
	origin, ok, err := HostOverride(EnvNameAIHost)
	if err != nil || ok || origin != "" {
		t.Fatalf("origin=%q ok=%v err=%v", origin, ok, err)
	}
}

func TestEffectiveAuthOriginOverride(t *testing.T) {
	t.Setenv(EnvNameAuthHost, "https://account.test.example/authapi/")
	got, err := EffectiveAuthOrigin(RemoteTypeChina)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://account.test.example" {
		t.Fatalf("got %s", got)
	}
}

func TestEffectiveAuthOriginDefaults(t *testing.T) {
	t.Setenv(EnvNameAuthHost, "")
	got, err := EffectiveAuthOrigin(RemoteTypeChina)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://account.seeed.cn" {
		t.Fatalf("china got %s", got)
	}
	got, err = EffectiveAuthOrigin(RemoteTypeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://account.seeed.cc" {
		t.Fatalf("global got %s", got)
	}
}

func TestEffectiveAIHostOverride(t *testing.T) {
	t.Setenv(EnvNameAIHost, "https://ai.test.example")
	got, err := EffectiveAIHost(RemoteTypeChina)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://ai.test.example" {
		t.Fatalf("got %s", got)
	}
}
