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

// ViewOrganization calls OpenAPI with the Access Key, checks the credential, and returns the organization.
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

	var info definition.OpenApiIntResponse[definition.ViewOrgInfoResponse]
	if err := body.Unmarshal(&info); err != nil {
		return nil, err
	}
	if info.Data.OrgID == 0 {
		return nil, fmt.Errorf("organization id missing from API response")
	}
	return &info.Data, nil
}
