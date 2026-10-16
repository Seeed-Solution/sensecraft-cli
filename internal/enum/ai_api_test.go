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
}
