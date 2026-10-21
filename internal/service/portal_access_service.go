package service

import (
	"context"
	"errors"

	"sensecraft-cloud-terminal/internal/client"
)

type UserOrgInfo struct {
	Account      string `json:"account"`
	Organization string `json:"organization"`
	OrgID        string `json:"org_id"`
	UserID       string `json:"user_id"`
	Nickname     string `json:"nickname"`
}

type AccessKeyItem struct {
	ID          int    `json:"id"`
	AccessID    string `json:"access_id"`
	AccessKey   string `json:"access_key"`
	AccessLevel string `json:"access_level"`
	Label       string `json:"label"`
}

func GetUserOrgInfo(ctx context.Context, remoteType, accessToken string, sandbox bool) (*UserOrgInfo, error) {
	if sandbox {
		return &UserOrgInfo{
			Account:      "sandbox@example.com",
			Organization: "Sandbox Org",
			OrgID:        "org-sandbox",
			UserID:       "user-sandbox",
			Nickname:     "Sandbox User",
		}, nil
	}

	api, err := newPortalAPIClient(remoteType)
	if err != nil {
		return nil, err
	}
	var info UserOrgInfo
	if _, err := api.GetJSON("/api/v1/user/getUserOrgInfo", nil, accessToken, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func ListAccessKeys(ctx context.Context, remoteType, accessToken string, sandbox bool) ([]AccessKeyItem, error) {
	if sandbox {
		return []AccessKeyItem{}, nil
	}

	api, err := newPortalAPIClient(remoteType)
	if err != nil {
		return nil, err
	}
	var list []AccessKeyItem
	if _, err := api.GetJSON("/api/v1/user/access/getAccessList", nil, accessToken, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func CreateAccessKey(ctx context.Context, remoteType, accessToken, label, accessLevel string, sandbox bool) (*AccessKeyItem, error) {
	if sandbox {
		return &AccessKeyItem{
			ID:          1,
			AccessID:    "sandbox-key-id",
			AccessKey:   "sandbox-access-key",
			AccessLevel: accessLevel,
			Label:       label,
		}, nil
	}

	api, err := newPortalAPIClient(remoteType)
	if err != nil {
		return nil, err
	}
	body := map[string]string{
		"label":       label,
		"accessLevel": accessLevel,
	}
	if _, err := api.PostJSON("/api/v1/user/access/createAccessKey", body, accessToken, nil); err != nil {
		return nil, err
	}

	keys, err := ListAccessKeys(ctx, remoteType, accessToken, sandbox)
	if err != nil {
		return nil, err
	}
	for i := len(keys) - 1; i >= 0; i-- {
		if keys[i].Label == label {
			return &keys[i], nil
		}
	}
	if len(keys) > 0 {
		return &keys[len(keys)-1], nil
	}
	return nil, errors.New("Access Key was created but not found in the list; please try again later")
}

func newPortalAPIClient(remoteType string) (*client.AuthAPIClient, error) {
	base, err := portalAPIBase(remoteType)
	if err != nil {
		return nil, err
	}
	return client.NewAuthAPIClient(base), nil
}
