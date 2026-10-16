package service

import (
	"encoding/json"
	"errors"
	"net/url"

	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/enum"
)

type GroupService struct {
	client *client.OpenApiClient
}

func GetGroupService(ctx app.CmdContext) GroupService {
	return GroupService{client: client.NewOpenApiClient(ctx.Config, ctx.SandBox)}
}

func (s GroupService) ListGroups() (json.RawMessage, error) {
	data, err := s.client.Get(enum.OpenApiUrlGroupList, nil)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			raw, _ := json.Marshal([]map[string]any{
				{"group_uuid": "sandbox-group-1", "name": "Sandbox Group"},
			})
			return raw, nil
		}
		return nil, err
	}
	return data, nil
}

func (s GroupService) CreateGroup(name string) (json.RawMessage, error) {
	return s.client.Post(enum.OpenApiUrlGroupCreate, map[string]string{"name": name})
}

func (s GroupService) UpdateGroup(groupUUID, name string) (json.RawMessage, error) {
	return s.client.Post(enum.OpenApiUrlGroupUpdate, map[string]string{
		"group_uuid": groupUUID,
		"name":       name,
	})
}

func (s GroupService) DeleteGroup(groupUUID string, yes bool) (json.RawMessage, error) {
	if err := requireYes(yes); err != nil {
		return nil, err
	}
	return s.client.PostQuery(enum.OpenApiUrlGroupDelete, url.Values{
		"group_uuid": {groupUUID},
	})
}

func (s GroupService) MoveDevicesToGroup(groupUUID string, devices []string, yes bool) (json.RawMessage, error) {
	if err := requireYes(yes); err != nil {
		return nil, err
	}
	return s.client.Post(enum.OpenApiUrlGroupMoveDevices, map[string]any{
		"group_uuid": groupUUID,
		"devices":    devices,
	})
}
