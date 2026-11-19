package config

import (
	"errors"
	"fmt"
	"strings"

	"sensecraft-cloud-terminal/internal/authhint"
	"sensecraft-cloud-terminal/internal/enum"
	"sensecraft-cloud-terminal/internal/utils"
)

// legacyRemoteTypeDevelop is an old saved remote_type. It is not a public site.
// Data and AI hosts match Global. Override them with SENSECRAFT_DATA_HOST and
// SENSECRAFT_AI_HOST.
const legacyRemoteTypeDevelop = "Develop"

// RemoteTypeOf is the selected profile's remote_type.
// No config, no selection, or an unknown key returns "" so callers use the default host.
func RemoteTypeOf(cli *CliConfig) string {
	if cli == nil {
		return ""
	}
	id := strings.TrimSpace(cli.UseApi)
	if id == "" {
		return ""
	}
	for _, item := range cli.AppConfigList {
		if item.KeyId == id {
			return strings.TrimSpace(item.RemoteType)
		}
	}
	return ""
}

// ResolveActiveAPIConfig 从 CLI 配置中解析当前使用的 SenseCAP OpenAPI 凭证。
func ResolveActiveAPIConfig(cli *CliConfig) (AccessApiConfig, error) {
	if cli == nil || len(cli.AppConfigList) == 0 {
		return AccessApiConfig{}, errors.New(authhint.NeedLogin())
	}

	if cli.UseApi == "" {
		return AccessApiConfig{}, errors.New(authhint.NeedLogin())
	}

	var selected = AccessApiConfig{}

	for _, apiConfig := range cli.AppConfigList {
		if cli.UseApi == apiConfig.KeyId {
			selected = apiConfig
			break
		}
	}

	err := accessConfigCheck(selected)
	if err != nil {
		return AccessApiConfig{}, err
	}

	return WithResolvedHost(selected)
}

// WithResolvedHost 根据 remote_type（或 Private 的 host）填充 Host，供 OpenAPI 客户端使用。
func WithResolvedHost(cfg AccessApiConfig) (AccessApiConfig, error) {
	host, err := resolveAPIHost(cfg)
	if err != nil {
		return AccessApiConfig{}, err
	}
	cfg.Host = utils.String2Ptr(host)
	return cfg, nil
}

func accessConfigCheck(cfg AccessApiConfig) error {
	if cfg.KeyId == "" || cfg.AccessKey == "" {
		return fmt.Errorf("current API config is missing key_id or access_key")
	}
	return nil
}

func resolveAPIHost(cfg AccessApiConfig) (string, error) {
	if origin, ok, err := enum.OfficialHostOverride(enum.EnvNameDataHost); ok || err != nil {
		return origin, err
	}
	switch cfg.RemoteType {
	case enum.RemoteTypeGlobal, legacyRemoteTypeDevelop, "":
		// legacyRemoteTypeDevelop is an old config label on a Global Access Key.
		// It uses the same public data host. Test hosts are SENSECRAFT_DATA_HOST,
		// SENSECRAFT_AI_HOST, and SENSECRAFT_AUTH_HOST.
		return enum.DomainNameOpenApiGlobal, nil
	case enum.RemoteTypeChina:
		return enum.DomainNameOpenApiChina, nil
	default:
		if cfg.RemoteType == enum.RemoteTypePrivate && cfg.Host != nil {
			return strings.TrimSuffix(*cfg.Host, "/"), nil
		}
		return "", fmt.Errorf("unable to resolve API host; set host or remote_type")
	}
}
