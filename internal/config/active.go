package config

import (
	"fmt"
	"strings"

	"sensecraft-cloud-terminal/internal/enum"
	"sensecraft-cloud-terminal/internal/utils"
)

// ResolveActiveAPIConfig 从 CLI 配置中解析当前使用的 SenseCAP OpenAPI 凭证。
func ResolveActiveAPIConfig(cli *CliConfig) (AccessApiConfig, error) {
	if cli == nil || len(cli.AppConfigList) == 0 {
		return AccessApiConfig{}, fmt.Errorf("no SenseCAP API config found; run: sensecraft-cli auth init")
	}

	if cli.UseApi == "" {
		return AccessApiConfig{}, fmt.Errorf("no SenseCAP API config found; run: sensecraft-cli auth init")
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
	if origin, ok, err := enum.HostOverride(enum.EnvNameDataHost); ok || err != nil {
		return origin, err
	}
	switch cfg.RemoteType {
	case enum.RemoteTypeGlobal, "":
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
