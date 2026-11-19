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
	t.Setenv(EnvNameAuthHost, "https://account-test.seeed.cc/authapi/")
	got, err := EffectiveAuthOrigin(RemoteTypeChina)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://account-test.seeed.cc" {
		t.Fatalf("got %s", got)
	}
}

func TestEffectiveAuthOriginRejectsUnofficialHost(t *testing.T) {
	t.Setenv(EnvNameAuthHost, "http://account.seeed.cc")
	if _, err := EffectiveAuthOrigin(RemoteTypeGlobal); err == nil {
		t.Fatal("expected http rejection")
	}
	t.Setenv(EnvNameAuthHost, "https://account.seeed.cc.evil.example")
	if _, err := EffectiveAuthOrigin(RemoteTypeGlobal); err == nil {
		t.Fatal("expected suffix rejection")
	}
	t.Setenv(EnvNameAuthHost, "https://evil.example")
	if _, err := EffectiveAuthOrigin(RemoteTypeGlobal); err == nil {
		t.Fatal("expected foreign host rejection")
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

func TestEffectiveConsoleOrigin(t *testing.T) {
	t.Setenv(EnvNameConsoleHost, "")
	got, err := EffectiveConsoleOrigin()
	if err != nil {
		t.Fatal(err)
	}
	if got != DomainNameAIConsole {
		t.Fatalf("default=%s", got)
	}
	t.Setenv(EnvNameConsoleHost, "https://console.seeed.cc/openapi/")
	got, err = EffectiveConsoleOrigin()
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://console.seeed.cc" {
		t.Fatalf("got %s", got)
	}
	t.Setenv(EnvNameConsoleHost, "https://console.example")
	if _, err := EffectiveConsoleOrigin(); err == nil {
		t.Fatal("expected foreign console host rejection")
	}
	t.Setenv(EnvNameConsoleHost, "http://sensecraft.seeed.cc")
	if _, err := EffectiveConsoleOrigin(); err == nil {
		t.Fatal("expected http console host rejection")
	}
}

func TestEffectiveAppSiteOrigin(t *testing.T) {
	t.Setenv(EnvNameAppSite, "")
	got, err := EffectiveAppSiteOrigin()
	if err != nil {
		t.Fatal(err)
	}
	if got != DomainNameAppHub {
		t.Fatalf("default=%s", got)
	}
	t.Setenv(EnvNameAppSite, "https://test-codecraft.seeed.cc/hub/")
	got, err = EffectiveAppSiteOrigin()
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://test-codecraft.seeed.cc/hub" {
		t.Fatalf("got %s", got)
	}
	t.Setenv(EnvNameAppSite, "https://test-codecraft.seeed.cc/hub?q=1")
	if _, err := EffectiveAppSiteOrigin(); err == nil {
		t.Fatal("expected query rejection")
	}
	t.Setenv(EnvNameAppSite, "https://hub.example/hub")
	if _, err := EffectiveAppSiteOrigin(); err == nil {
		t.Fatal("expected foreign app site rejection")
	}
	t.Setenv(EnvNameAppSite, "http://codecraft.seeed.cc/hub")
	if _, err := EffectiveAppSiteOrigin(); err == nil {
		t.Fatal("expected http app site rejection")
	}
}

func TestEffectiveAIHostOverride(t *testing.T) {
	t.Setenv(EnvNameAIHost, "https://test-api-gateway.seeed.cc/openapi/")
	got, err := EffectiveAIHost(RemoteTypeChina)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://test-api-gateway.seeed.cc" {
		t.Fatalf("got %s", got)
	}
}

func TestEffectiveAIHostRejectsUnofficialHost(t *testing.T) {
	t.Setenv(EnvNameAIHost, "https://ai.test.example")
	if _, err := EffectiveAIHost(RemoteTypeGlobal); err == nil {
		t.Fatal("expected foreign host rejection")
	}
	t.Setenv(EnvNameAIHost, "http://test-api-gateway.seeed.cc")
	if _, err := EffectiveAIHost(RemoteTypeGlobal); err == nil {
		t.Fatal("expected http rejection")
	}
}
