package enum

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// HostOverride reads an origin from the environment.
// ok is false when the variable is unset or blank.
func HostOverride(envName string) (origin string, ok bool, err error) {
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return "", false, nil
	}
	origin, err = NormalizeOrigin(raw)
	if err != nil {
		return "", true, fmt.Errorf("%s: %w", envName, err)
	}
	return origin, true, nil
}

// NormalizeOrigin accepts an http(s) origin. A trailing slash, /openapi, or /authapi is removed.
func NormalizeOrigin(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	for i := 0; i < 2; i++ {
		s = strings.TrimSuffix(s, "/")
		s = stripOriginSuffix(s, "/openapi")
		s = stripOriginSuffix(s, "/authapi")
	}
	s = strings.TrimSuffix(s, "/")
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" || u.RawQuery != "" || u.User != nil {
		return "", fmt.Errorf("invalid host %q; use an http(s) origin such as https://example.com", raw)
	}
	return u.Scheme + "://" + u.Host, nil
}

// EffectiveAIHost returns SENSECRAFT_AI_HOST when set, otherwise the production AI origin.
func EffectiveAIHost(remoteType string) (string, error) {
	if origin, ok, err := HostOverride(EnvNameAIHost); ok || err != nil {
		return origin, err
	}
	return ResolveAIServerHost(remoteType), nil
}

// EffectiveAuthOrigin returns SENSECRAFT_AUTH_HOST when set, otherwise the
// production account center for remoteType (China or Global). The value is an
// origin; callers append /authapi.
func EffectiveAuthOrigin(remoteType string) (string, error) {
	if origin, ok, err := HostOverride(EnvNameAuthHost); ok || err != nil {
		return origin, err
	}
	switch strings.TrimSpace(remoteType) {
	case RemoteTypeChina:
		return "https://account.seeed.cn", nil
	default:
		return "https://account.seeed.cc", nil
	}
}

func stripOriginSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && strings.EqualFold(s[len(s)-len(suffix):], suffix) {
		return s[:len(s)-len(suffix)]
	}
	return s
}
