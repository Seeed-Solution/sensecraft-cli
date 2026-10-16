package task

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/enum"
	"sensecraft-cloud-terminal/internal/service"
	"sensecraft-cloud-terminal/internal/ui"
)

// AddAccessOptions 手动添加 Access Key 的输入。
type AddAccessOptions struct {
	KeyID     string
	AccessKey string
	Site      string // 空则默认 Global
}

// RunAddAccessFlow 校验用户提供的 Access Key，写入本地配置并设为当前使用。
func RunAddAccessFlow(ctx context.Context, console *ui.Console, opts AddAccessOptions) error {
	_ = ctx

	keyID := strings.TrimSpace(opts.KeyID)
	accessKey := strings.TrimSpace(opts.AccessKey)
	site := strings.TrimSpace(opts.Site)

	if keyID == "" || accessKey == "" {
		if !console.IsInteractive() {
			return fmt.Errorf("key-id and access-key are required (or run in an interactive terminal)")
		}
		var err error
		if keyID == "" {
			keyID, err = console.Input("Key ID", "")
			if err != nil {
				return err
			}
			keyID = strings.TrimSpace(keyID)
		}
		if accessKey == "" {
			accessKey, err = console.Input("Access Key", "")
			if err != nil {
				return err
			}
			accessKey = strings.TrimSpace(accessKey)
		}
		if site == "" {
			idx, err := console.Select("Select the SenseCAP site to connect to", []string{
				"Global (International)",
				"China (China site)",
			}, 0)
			if err != nil {
				return err
			}
			site = remoteTypeFromSelectIndex(idx)
		}
	}

	if keyID == "" {
		return fmt.Errorf("key-id cannot be empty")
	}
	if accessKey == "" {
		return fmt.Errorf("access-key cannot be empty")
	}

	remoteType, err := normalizeSite(site)
	if err != nil {
		return err
	}

	configPath, err := config.ResolveConfigPath()
	if err != nil {
		return err
	}

	console.Println("Verifying Access Key against OpenAPI ...")
	org, err := service.ViewOrganization(config.AccessApiConfig{
		RemoteType: remoteType,
		KeyId:      keyID,
		AccessKey:  accessKey,
	})
	if err != nil {
		console.PrintError(err.Error())
		return err
	}
	console.PrintSuccess("Access Key verified")
	console.Printf("  Site: %s\n", remoteType)
	console.Printf("  Organization ID: %d\n", org.OrgID)
	console.Printf("  Key ID: %s\n", keyID)

	profile := config.BuildAccessProfile(remoteType, strconv.FormatInt(org.OrgID, 10), keyID, accessKey)
	if err := config.UpsertAccessProfile(profile, true); err != nil {
		return err
	}

	console.Println("")
	console.PrintSuccess("Access config saved and set as active")
	console.Printf("  Config file: %s\n", configPath)
	return nil
}

func remoteTypeFromSelectIndex(idx int) string {
	if idx == 1 {
		return enum.RemoteTypeChina
	}
	return enum.RemoteTypeGlobal
}

func normalizeSite(site string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(site)) {
	case "", "global", "international":
		return enum.RemoteTypeGlobal, nil
	case "china", "cn":
		return enum.RemoteTypeChina, nil
	default:
		return "", fmt.Errorf("invalid site %q; use Global or China", site)
	}
}
