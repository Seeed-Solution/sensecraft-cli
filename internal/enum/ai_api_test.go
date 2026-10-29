package enum

import "testing"

func TestResolveAIServerHost(t *testing.T) {
	if got := ResolveAIServerHost(RemoteTypeGlobal); got != DomainNameAIServerGlobal {
		t.Fatalf("Global=%s", got)
	}
	if got := ResolveAIServerHost(RemoteTypeChina); got != DomainNameAIServerChina {
		t.Fatalf("China=%s", got)
	}
	if got := ResolveAIServerHost(""); got != DomainNameAIServerGlobal {
		t.Fatalf("default=%s", got)
	}
	if got := ResolveAIServerHost(RemoteTypeDevelop); got != DomainNameAIServerDevelop {
		t.Fatalf("Develop=%s", got)
	}
	if DomainNameAIServerGlobal != "https://api-gateway.seeed.cc" || DomainNameAIServerChina != "https://api-gateway.seeed.cc" {
		t.Fatalf("AI API host global=%s china=%s", DomainNameAIServerGlobal, DomainNameAIServerChina)
	}
	if DomainNameAIConsole != "https://sensecraft.seeed.cc" {
		t.Fatalf("console host=%s", DomainNameAIConsole)
	}
}
