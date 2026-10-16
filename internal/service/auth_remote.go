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

func portalAPIBase(remoteType string) string {
	if portalAPIBaseOverride != "" {
		return portalAPIBaseOverride
	}
	switch normalizeRemoteType(remoteType) {
	case enum.RemoteTypeChina:
		return enum.AuthAPIBaseChina
	default:
		return enum.AuthAPIBaseGlobal
	}
}

func getDeviceAuthURL(remoteType string) string {
	switch normalizeRemoteType(remoteType) {
	case enum.RemoteTypeChina:
		return enum.DeviceFlowOauth2China
	default:
		return enum.DeviceFlowOauth2Global
	}
}

func getDeviceTokenURL(remoteType string) string {
	switch normalizeRemoteType(remoteType) {
	case enum.RemoteTypeChina:
		return enum.DeviceFlowOauthTokenChina
	default:
		return enum.DeviceFlowOauthTokenGlobal
	}
}
