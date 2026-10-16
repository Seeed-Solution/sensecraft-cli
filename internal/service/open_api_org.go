package service

import (
	"encoding/json"
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
	data, err := api.Get(enum.OpenApiUrlViewOrganization, nil)
	if err != nil {
		return nil, fmt.Errorf("verify access key: %w", err)
	}

	var info definition.OpenApiIntResponse[definition.ViewOrgInfoResponse]
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parse organization info: %w", err)
	}
	if info.Data.OrgID == 0 {
		return nil, fmt.Errorf("organization id missing from API response")
	}
	return &info.Data, nil
}
