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

// OfficialHostOverride reads a credential-bearing origin.
// The value must be https, and the hostname must be seeed.cc, seeed.cn, or a subdomain.
func OfficialHostOverride(envName string) (origin string, ok bool, err error) {
	origin, ok, err = HostOverride(envName)
	if err != nil || !ok {
		return origin, ok, err
	}
	if err = requireOfficialOrigin(origin); err != nil {
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

func requireOfficialOrigin(origin string) error {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || !officialSeeedHost(u.Hostname()) {
		return fmt.Errorf("only https origins under seeed.cc or seeed.cn are allowed, got %q", origin)
	}
	return nil
}

func officialSeeedHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == "seeed.cc" || host == "seeed.cn" {
		return true
	}
	return strings.HasSuffix(host, ".seeed.cc") || strings.HasSuffix(host, ".seeed.cn")
}

// EffectiveAIHost returns SENSECRAFT_AI_HOST when set, otherwise the production AI origin.
func EffectiveAIHost(remoteType string) (string, error) {
	if origin, ok, err := OfficialHostOverride(EnvNameAIHost); ok || err != nil {
		return origin, err
	}
	return ResolveAIServerHost(remoteType), nil
}

// EffectiveConsoleOrigin returns SENSECRAFT_CONSOLE_HOST when set, otherwise the
// public SenseCraft AI website. Model links append /ai.
func EffectiveConsoleOrigin() (string, error) {
	if origin, ok, err := OfficialHostOverride(EnvNameConsoleHost); ok || err != nil {
		return origin, err
	}
	return DomainNameAIConsole, nil
}

// EffectiveAppSiteOrigin returns SENSECRAFT_APP_SITE when set, otherwise the
// public CodeCraft Hub. The value may include a path. A trailing slash is removed.
func EffectiveAppSiteOrigin() (string, error) {
	raw := strings.TrimSpace(os.Getenv(EnvNameAppSite))
	if raw == "" {
		return DomainNameAppHub, nil
	}
	origin, err := NormalizeSiteURL(raw)
	if err != nil {
		return "", fmt.Errorf("%s: %w", EnvNameAppSite, err)
	}
	if err = requireOfficialOrigin(origin); err != nil {
		return "", fmt.Errorf("%s: %w", EnvNameAppSite, err)
	}
	return origin, nil
}

// NormalizeSiteURL accepts an http(s) URL with an optional path.
// A trailing slash is removed. Query, fragment, and userinfo are rejected.
func NormalizeSiteURL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimSuffix(s, "/")
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return "", fmt.Errorf("invalid site %q; use an http(s) URL such as https://codecraft.seeed.cc/hub", raw)
	}
	path := strings.TrimSuffix(u.EscapedPath(), "/")
	return u.Scheme + "://" + u.Host + path, nil
}

// EffectiveAuthOrigin returns SENSECRAFT_AUTH_HOST when set, otherwise the
// production account center for remoteType (China or Global). The value is an
// origin; callers append /authapi.
func EffectiveAuthOrigin(remoteType string) (string, error) {
	if origin, ok, err := OfficialHostOverride(EnvNameAuthHost); ok || err != nil {
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
