package service

import (
	"fmt"
	"sensecraft-cloud-terminal/internal/definition"

	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/enum"
)

type OrganizationInfo struct {
	OrgID int64 `json:"orgId"`
}

// ViewOrganization 使用 Access Key 调用 OpenAPI，校验凭证并返回组织信息。
func ViewOrganization(cfg config.AccessApiConfig) (*definition.ViewOrgInfoResponse, error) {
	resolved, err := config.WithResolvedHost(cfg)
	if err != nil {
		return nil, err
	}

	api := client.NewOpenApiClient(resolved, false)
	body, err := api.GetBody(enum.OpenApiUrlViewOrganization, nil)
	if err != nil {
		return nil, fmt.Errorf("verify access key: %w", err)
	}

	// GetBody already unwraps {code,msg,data}. The organization object is that data.
	var info definition.ViewOrgInfoResponse
	if err := body.Unmarshal(&info); err != nil {
		return nil, err
	}
	if info.OrgID == 0 {
		return nil, fmt.Errorf("organization id missing from API response")
	}
	return &info, nil
}
