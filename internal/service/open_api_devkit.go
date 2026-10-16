package service

import (
	"encoding/json"
	"fmt"
	"slices"

	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/definition"
	"sensecraft-cloud-terminal/internal/enum"
)

type DevKitService struct {
	client *client.OpenApiClient
}

func GetDevKitService(ctx app.CmdContext) DevKitService {
	return DevKitService{client: client.NewOpenApiClient(ctx.Config, ctx.SandBox)}
}

func (s DevKitService) CreateDevelopmentKit(sku, devName string, deviceInfo any) (definition.CreateDevelopmentKitResponse, error) {
	if !slices.Contains(enum.DevKitSKUList, sku) {
		return definition.CreateDevelopmentKitResponse{}, fmt.Errorf(
			"invalid sku %q, allowed values: %v", sku, enum.DevKitSKUList,
		)
	}

	data, err := s.client.CreateDevelopmentKit(sku, devName, deviceInfo)
	if err != nil {
		return definition.CreateDevelopmentKitResponse{}, err
	}

	resp := definition.CreateDevelopmentKitResponse{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return definition.CreateDevelopmentKitResponse{}, err
	}
	return resp, nil
}

func (s DevKitService) ViewNodeKey(eui string) (definition.ViewNodeKeyResponse, error) {
	if len(eui) != 16 {
		return definition.ViewNodeKeyResponse{}, fmt.Errorf("EUI must be 16 characters, got length: %d", len(eui))
	}

	data, err := s.client.ViewNodeKey(eui)
	if err != nil {
		return definition.ViewNodeKeyResponse{}, err
	}

	resp := definition.ViewNodeKeyResponse{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return definition.ViewNodeKeyResponse{}, err
	}
	return resp, nil
}
