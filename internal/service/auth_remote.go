package service

import (
	"strings"

	"sensecraft-cloud-terminal/internal/enum"
)

func normalizeRemoteType(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return enum.RemoteTypeGlobal
	}
	return v
}

// portalAPIBaseOverride, when non-empty, replaces the resolved authapi base.
// Used by tests (httptest) only; production code must leave it empty.
var portalAPIBaseOverride string

func portalAPIBase(remoteType string) (string, error) {
	if portalAPIBaseOverride != "" {
		return portalAPIBaseOverride, nil
	}
	origin, err := enum.EffectiveAuthOrigin(normalizeRemoteType(remoteType))
	if err != nil {
		return "", err
	}
	return origin + "/authapi", nil
}

func getDeviceAuthURL(remoteType string) (string, error) {
	origin, err := enum.EffectiveAuthOrigin(normalizeRemoteType(remoteType))
	if err != nil {
		return "", err
	}
	return origin + "/authapi/api/v1/auth/oauth/device", nil
}

func getDeviceTokenURL(remoteType string) (string, error) {
	origin, err := enum.EffectiveAuthOrigin(normalizeRemoteType(remoteType))
	if err != nil {
		return "", err
	}
	return origin + "/authapi/api/v1/auth/oauth/device/token", nil
}
