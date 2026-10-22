package config

import (
	"fmt"
	"strings"
	"time"
)

// ResetConfigCache clears the in-memory config cache. Call it after a save.
func ResetConfigCache() {
	cfg = nil
}

// UpsertAccessProfile writes or updates one SenseCAP API profile and can make it active.
func UpsertAccessProfile(profile AccessApiConfig, setActive bool) error {
	cli, err := LoadConfig()
	if err != nil {
		return err
	}
	if cli == nil {
		cli = &CliConfig{}
	}

	updated := false
	for i := range cli.AppConfigList {
		if cli.AppConfigList[i].KeyId == profile.KeyId {
			cli.AppConfigList[i] = profile
			updated = true
			break
		}
	}
	if !updated {
		cli.AppConfigList = append(cli.AppConfigList, profile)
	}
	if setActive {
		cli.UseApi = profile.KeyId
	}
	if err := SaveCliConfig(cli); err != nil {
		return err
	}
	ResetConfigCache()
	return nil
}

// BuildAccessProfile builds a local SenseCAP profile from a completed login.
func BuildAccessProfile(remoteType, orgID, keyID, accessKey string) AccessApiConfig {
	return AccessApiConfig{
		RemoteType: remoteType,
		OrgId:      orgID,
		KeyId:      keyID,
		AccessKey:  accessKey,
		CreateTime: time.Now().Format(time.RFC3339),
	}
}

// FormatCliConfigInfo formats the config summary printed by auth info.
func FormatCliConfigInfo(cli *CliConfig, configPath string) string {
	if cli == nil || len(cli.AppConfigList) == 0 {
		return fmt.Sprintf("Config file: %s\n\nNo profiles configured yet.\n  Run: sensecraft-cli auth init --kind sensecap\n", configPath)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Config file: %s\n\n", configPath)
	b.WriteString("=== Access Key (SenseCAP and SenseCraft AI) ===\n")
	if cli.UseApi != "" {
		fmt.Fprintf(&b, "Active Key ID: %s\n", cli.UseApi)
	} else {
		b.WriteString("Active Key ID: (not set)\n")
	}
	b.WriteString("\n")
	for i, item := range cli.AppConfigList {
		if i > 0 {
			b.WriteString("\n")
		}
		active := cli.UseApi != "" && cli.UseApi == item.KeyId
		if active {
			b.WriteString("[active]\n")
		} else {
			b.WriteString("[inactive]\n")
		}
		b.WriteString(formatAccessProfile(item))
	}
	return b.String()
}

func formatAccessProfile(item AccessApiConfig) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  Kind: %s\n", ProfileKindSenseCAP)
	fmt.Fprintf(&b, "  Site: %s\n", formatRemoteType(item.RemoteType))
	fmt.Fprintf(&b, "  Organization ID: %s\n", displayOrDash(item.OrgId))
	fmt.Fprintf(&b, "  Key ID: %s\n", displayOrDash(item.KeyId))
	fmt.Fprintf(&b, "  Created at: %s\n", displayOrDash(item.CreateTime))
	if item.Host != nil && strings.TrimSpace(*item.Host) != "" {
		fmt.Fprintf(&b, "  Host: %s\n", strings.TrimSpace(*item.Host))
	}
	return b.String()
}

func formatRemoteType(remoteType string) string {
	switch remoteType {
	case "Global", "":
		return "Global (International)"
	case "China":
		return "China (China site)"
	case "Private":
		return "Private (Self-hosted)"
	default:
		if remoteType == "" {
			return "Global (International)"
		}
		return remoteType
	}
}

func displayOrDash(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return v
}
