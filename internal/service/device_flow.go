package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"sensecraft-cloud-terminal/internal/enum"

	"golang.org/x/oauth2"
)

type DeviceAuthSession struct {
	DeviceCode              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               int
	Interval                int
}

type DeviceTokenResult struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Expiry       string
}

func oauth2Config(remoteType string) (*oauth2.Config, error) {
	deviceAuthURL, err := getDeviceAuthURL(remoteType)
	if err != nil {
		return nil, err
	}
	tokenURL, err := getDeviceTokenURL(remoteType)
	if err != nil {
		return nil, err
	}
	return &oauth2.Config{
		ClientID: enum.OAuthClientID,
		Endpoint: oauth2.Endpoint{
			DeviceAuthURL: deviceAuthURL,
			TokenURL:      tokenURL,
			AuthStyle:     oauth2.AuthStyleInParams,
		},
	}, nil
}

func StartDeviceFlow(ctx context.Context, remoteType string, sandbox bool) (*DeviceAuthSession, error) {
	if sandbox {
		return &DeviceAuthSession{
			DeviceCode:              "sandbox-device-code",
			UserCode:                "ABCD-2345",
			VerificationURI:         "https://account.seeed.cc/device",
			VerificationURIComplete: "https://account.seeed.cc/device?user_code=ABCD-2345",
			ExpiresIn:               900,
			Interval:                2,
		}, nil
	}

	cfg, err := oauth2Config(remoteType)
	if err != nil {
		return nil, err
	}
	resp, err := cfg.DeviceAuth(ctx)
	if err != nil {
		return nil, fmt.Errorf("request device authorization code: %w", mapDeviceFlowError(err))
	}

	interval := int(resp.Interval)
	if interval <= 0 {
		interval = 5
	}
	expiresIn := int(resp.Expiry.Sub(time.Now()).Seconds())
	if expiresIn <= 0 && !resp.Expiry.IsZero() {
		expiresIn = 1
	}
	if expiresIn <= 0 {
		expiresIn = 900
	}

	return &DeviceAuthSession{
		DeviceCode:              resp.DeviceCode,
		UserCode:                resp.UserCode,
		VerificationURI:         resp.VerificationURI,
		VerificationURIComplete: resp.VerificationURIComplete,
		ExpiresIn:               expiresIn,
		Interval:                interval,
	}, nil
}

func PollDeviceToken(ctx context.Context, remoteType string, session *DeviceAuthSession, sandbox bool) (*DeviceTokenResult, error) {
	if sandbox {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
			return &DeviceTokenResult{
				AccessToken:  "sandbox-access-token",
				RefreshToken: "sandbox-refresh-token",
				TokenType:    "Bearer",
				Expiry:       time.Now().Add(time.Hour).Format(time.RFC3339),
			}, nil
		}
	}

	da := &oauth2.DeviceAuthResponse{
		DeviceCode:              session.DeviceCode,
		UserCode:                session.UserCode,
		VerificationURI:         session.VerificationURI,
		VerificationURIComplete: session.VerificationURIComplete,
		Interval:                int64(session.Interval),
	}
	if session.ExpiresIn > 0 {
		da.Expiry = time.Now().Add(time.Duration(session.ExpiresIn) * time.Second)
	}

	cfg, err := oauth2Config(remoteType)
	if err != nil {
		return nil, err
	}
	token, err := cfg.DeviceAccessToken(ctx, da)
	if err != nil {
		return nil, mapDeviceFlowError(err)
	}

	expiry := token.Expiry.Format(time.RFC3339)
	if token.Expiry.IsZero() {
		expiry = ""
	}

	return &DeviceTokenResult{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       expiry,
	}, nil
}

func mapDeviceFlowError(err error) error {
	var re *oauth2.RetrieveError
	if errors.As(err, &re) {
		switch re.ErrorCode {
		case "access_denied":
			return errors.New("user denied authorization")
		case "expired_token":
			return errors.New("authorization expired, please run auth init again")
		}
	}
	return err
}
