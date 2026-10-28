package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sensecraft-cloud-terminal/internal/authhint"
	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/enum"
	"sensecraft-cloud-terminal/internal/service"
	"sensecraft-cloud-terminal/internal/ui"
	"sensecraft-cloud-terminal/internal/utils"
)

const pendingAuthFile = "sensecraft-auth-pending.json"

// SplitAuthOptions runs auth init as separate commands instead of the wizard.
// With no split flags, RunAuthInit keeps the interactive wizard.
type SplitAuthOptions struct {
	Sandbox     bool
	Site        string
	NoWait      bool
	JSON        bool
	DeviceCode  string
	KeyID       string
	Label       string
	AccessLevel string

	flow authFlow
}

type authFlow struct {
	start  func(ctx context.Context, remoteType string, sandbox bool) (*service.DeviceAuthSession, error)
	poll   func(ctx context.Context, remoteType string, session *service.DeviceAuthSession, sandbox bool) (*service.DeviceTokenResult, error)
	org    func(ctx context.Context, remoteType, accessToken string, sandbox bool) (*service.UserOrgInfo, error)
	list   func(ctx context.Context, remoteType, accessToken string, sandbox bool) ([]service.AccessKeyItem, error)
	create func(ctx context.Context, remoteType, accessToken, label, accessLevel string, sandbox bool) (*service.AccessKeyItem, error)
}

type pendingAuth struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
	RemoteType              string `json:"remote_type"`
	Sandbox                 bool   `json:"sandbox"`
	AccessToken             string `json:"access_token,omitempty"`
	RefreshToken            string `json:"refresh_token,omitempty"`
	TokenExpiry             string `json:"token_expiry,omitempty"`
}

type authStartResult struct {
	VerificationURL string `json:"verification_url"`
	UserCode        string `json:"user_code"`
	DeviceCode      string `json:"device_code"`
	ExpiresIn       int    `json:"expires_in"`
	Site            string `json:"site"`
	Note            string `json:"note,omitempty"`
	Hint            string `json:"hint"`
}

type authKeyResult struct {
	Label       string `json:"label"`
	AccessID    string `json:"access_id"`
	AccessLevel string `json:"access_level"`
}

type authListResult struct {
	DeviceCode   string          `json:"device_code"`
	Site         string          `json:"site"`
	Account      string          `json:"account"`
	Organization string          `json:"organization"`
	OrgID        string          `json:"org_id"`
	Keys         []authKeyResult `json:"keys"`
	Hint         string          `json:"hint"`
}

type authSaveResult struct {
	Site         string `json:"site"`
	Account      string `json:"account"`
	Organization string `json:"organization"`
	OrgID        string `json:"org_id"`
	KeyID        string `json:"key_id"`
	Label        string `json:"label"`
	AccessLevel  string `json:"access_level"`
	ConfigFile   string `json:"config_file"`
	Hint         string `json:"hint"`
}

// RunAuthInit runs either the interactive wizard or one step of the split flow.
func RunAuthInit(ctx context.Context, console *ui.Console, opts SplitAuthOptions) error {
	if opts.split() {
		return runSplitAuth(ctx, console, opts)
	}
	if !console.IsInteractive() && !opts.Sandbox {
		return errAuthInitNonInteractive()
	}
	return RunInitFlow(ctx, console, opts.Sandbox)
}

func (o SplitAuthOptions) split() bool {
	return o.NoWait ||
		strings.TrimSpace(o.DeviceCode) != "" ||
		strings.TrimSpace(o.KeyID) != "" ||
		strings.TrimSpace(o.Label) != "" ||
		strings.TrimSpace(o.AccessLevel) != ""
}

func errAuthInitNonInteractive() error {
	return fmt.Errorf("%s\nTo save a key after that:\n  sensecraft-cli auth init --device-code <device_code> --key-id <access_id>\n  sensecraft-cli auth init --device-code <device_code> --label <name> --access-level read-write", authhint.NeedLogin())
}

func runSplitAuth(ctx context.Context, console *ui.Console, opts SplitAuthOptions) error {
	deviceCode := strings.TrimSpace(opts.DeviceCode)
	keyID := strings.TrimSpace(opts.KeyID)
	label := strings.TrimSpace(opts.Label)
	level := strings.TrimSpace(opts.AccessLevel)

	if opts.NoWait && deviceCode != "" {
		return fmt.Errorf("--no-wait and --device-code cannot be used together")
	}
	if opts.NoWait && (keyID != "" || label != "" || level != "") {
		return fmt.Errorf("--no-wait only requests a device code; choose a key after authorization")
	}
	if keyID != "" && label != "" {
		return fmt.Errorf("use either --key-id or --label")
	}
	if level != "" && label == "" {
		return fmt.Errorf("--access-level requires --label")
	}
	if strings.TrimSpace(opts.Label) == "" && opts.Label != "" {
		return fmt.Errorf("label cannot be empty")
	}
	if !opts.NoWait && deviceCode == "" {
		return fmt.Errorf("the split flow needs --no-wait or --device-code")
	}

	opts.DeviceCode = deviceCode
	opts.KeyID = keyID
	opts.Label = label
	opts.AccessLevel = level
	if opts.NoWait {
		return startSplitAuth(ctx, console, opts)
	}
	if keyID != "" || label != "" {
		return finishSplitAuth(ctx, console, opts)
	}
	return listSplitAuth(ctx, console, opts)
}

func startSplitAuth(ctx context.Context, console *ui.Console, opts SplitAuthOptions) error {
	site, err := normalizeAuthSite(opts.Site)
	if err != nil {
		return err
	}
	session, err := opts.flow.startDevice(ctx, site, opts.Sandbox)
	if err != nil {
		return err
	}
	if session == nil || strings.TrimSpace(session.DeviceCode) == "" {
		return fmt.Errorf("device authorization did not return a device code")
	}
	verificationURL := session.VerificationURIComplete
	if verificationURL == "" {
		verificationURL = session.VerificationURI
	}
	if verificationURL == "" {
		return fmt.Errorf("device authorization did not return a verification url")
	}

	pending := &pendingAuth{
		DeviceCode:              session.DeviceCode,
		UserCode:                session.UserCode,
		VerificationURI:         session.VerificationURI,
		VerificationURIComplete: session.VerificationURIComplete,
		ExpiresIn:               session.ExpiresIn,
		Interval:                session.Interval,
		RemoteType:              site,
		Sandbox:                 opts.Sandbox,
	}
	if err := savePendingAuth(pending); err != nil {
		return err
	}

	result := authStartResult{
		VerificationURL: verificationURL,
		UserCode:        session.UserCode,
		DeviceCode:      session.DeviceCode,
		ExpiresIn:       session.ExpiresIn,
		Site:            site,
		Hint:            authhint.DeviceCodeIssued(session.DeviceCode),
	}
	if site == enum.RemoteTypeChina {
		result.Note = "The China site supports SenseCraft Data only. Use Global for SenseCraft AI and CodeCraft Hub."
	}
	return writeAuthJSON(console, result)
}

func listSplitAuth(ctx context.Context, console *ui.Console, opts SplitAuthOptions) error {
	pending, err := loadPendingForDevice(opts.DeviceCode)
	if err != nil {
		return err
	}
	if err := ensurePendingToken(ctx, opts.flow, pending, opts.Sandbox); err != nil {
		return err
	}
	org, keys, err := loadOrgAndKeys(ctx, opts.flow, pending, opts.Sandbox)
	if err != nil {
		return err
	}
	views := make([]authKeyResult, 0, len(keys))
	for _, key := range keys {
		views = append(views, authKeyResult{
			Label:       key.Label,
			AccessID:    key.AccessID,
			AccessLevel: key.AccessLevel,
		})
	}
	return writeAuthJSON(console, authListResult{
		DeviceCode:   pending.DeviceCode,
		Site:         pending.RemoteType,
		Account:      org.Account,
		Organization: org.Organization,
		OrgID:        org.OrgID,
		Keys:         views,
		Hint:         authhint.ChooseAccessKey(pending.DeviceCode),
	})
}

func finishSplitAuth(ctx context.Context, console *ui.Console, opts SplitAuthOptions) error {
	pending, err := loadPendingForDevice(opts.DeviceCode)
	if err != nil {
		return err
	}
	if err := ensurePendingToken(ctx, opts.flow, pending, opts.Sandbox); err != nil {
		return err
	}
	org, keys, err := loadOrgAndKeys(ctx, opts.flow, pending, opts.Sandbox)
	if err != nil {
		return err
	}

	var selected *service.AccessKeyItem
	if opts.KeyID != "" {
		selected, err = findAccessKey(keys, opts.KeyID)
	} else {
		level, levelErr := normalizeAccessLevel(opts.AccessLevel)
		if levelErr != nil {
			return levelErr
		}
		selected, err = opts.flow.createKey(ctx, pending.RemoteType, pending.AccessToken, opts.Label, level, pending.Sandbox || opts.Sandbox)
	}
	if err != nil {
		return err
	}
	if selected == nil || strings.TrimSpace(selected.AccessID) == "" || strings.TrimSpace(selected.AccessKey) == "" {
		return fmt.Errorf("Access Key secret is empty")
	}

	profile := config.BuildAccessProfile(pending.RemoteType, org.OrgID, selected.AccessID, selected.AccessKey)
	if err := config.UpsertAccessProfile(profile, true); err != nil {
		return err
	}
	if err := deletePendingAuth(); err != nil {
		return err
	}
	configPath, err := config.ResolveConfigPath()
	if err != nil {
		return err
	}
	return writeAuthJSON(console, authSaveResult{
		Site:         pending.RemoteType,
		Account:      org.Account,
		Organization: org.Organization,
		OrgID:        org.OrgID,
		KeyID:        selected.AccessID,
		Label:        selected.Label,
		AccessLevel:  selected.AccessLevel,
		ConfigFile:   configPath,
		Hint:         "Access Key saved. You can now run: sensecraft-cli device list",
	})
}

func loadOrgAndKeys(ctx context.Context, flow authFlow, pending *pendingAuth, sandbox bool) (*service.UserOrgInfo, []service.AccessKeyItem, error) {
	useSandbox := pending.Sandbox || sandbox
	org, err := flow.userOrg(ctx, pending.RemoteType, pending.AccessToken, useSandbox)
	if err != nil {
		return nil, nil, err
	}
	if org == nil || strings.TrimSpace(org.OrgID) == "" {
		return nil, nil, fmt.Errorf("authorization did not return an organization id")
	}
	keys, err := flow.accessKeys(ctx, pending.RemoteType, pending.AccessToken, useSandbox)
	if err != nil {
		return nil, nil, err
	}
	return org, keys, nil
}

func ensurePendingToken(ctx context.Context, flow authFlow, pending *pendingAuth, sandbox bool) error {
	if strings.TrimSpace(pending.AccessToken) != "" {
		return nil
	}
	token, err := flow.pollToken(ctx, pending.RemoteType, pending.session(), pending.Sandbox || sandbox)
	if err != nil {
		return err
	}
	if token == nil || strings.TrimSpace(token.AccessToken) == "" {
		return fmt.Errorf("authorization returned an empty access token")
	}
	pending.AccessToken = token.AccessToken
	pending.RefreshToken = token.RefreshToken
	pending.TokenExpiry = token.Expiry
	return savePendingAuth(pending)
}

func findAccessKey(keys []service.AccessKeyItem, accessID string) (*service.AccessKeyItem, error) {
	for i := range keys {
		if keys[i].AccessID == accessID {
			selected := keys[i]
			return &selected, nil
		}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no Access Key %s; this account has none yet; rerun with --label", accessID)
	}
	ids := make([]string, 0, len(keys))
	for _, key := range keys {
		if key.AccessID != "" {
			ids = append(ids, key.AccessID)
		}
	}
	return nil, fmt.Errorf("no Access Key %s; available access_id values: %s", accessID, strings.Join(ids, ", "))
}

func normalizeAuthSite(site string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(site)) {
	case "global":
		return enum.RemoteTypeGlobal, nil
	case "china":
		return enum.RemoteTypeChina, nil
	case "":
		return "", fmt.Errorf("--site is required: Global or China")
	default:
		return "", fmt.Errorf("site must be Global or China")
	}
}

func normalizeAccessLevel(level string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "read-write", "write", "rw":
		return enum.ApiAccessLevelRW, nil
	case "only-read", "read-only", "read":
		return enum.ApiAccessLevelOR, nil
	default:
		return "", fmt.Errorf("access-level must be read-write or only-read")
	}
}

func (p *pendingAuth) session() *service.DeviceAuthSession {
	return &service.DeviceAuthSession{
		DeviceCode:              p.DeviceCode,
		UserCode:                p.UserCode,
		VerificationURI:         p.VerificationURI,
		VerificationURIComplete: p.VerificationURIComplete,
		ExpiresIn:               p.ExpiresIn,
		Interval:                p.Interval,
	}
}

func (f authFlow) startDevice(ctx context.Context, remoteType string, sandbox bool) (*service.DeviceAuthSession, error) {
	if f.start != nil {
		return f.start(ctx, remoteType, sandbox)
	}
	return service.StartDeviceFlow(ctx, remoteType, sandbox)
}

func (f authFlow) pollToken(ctx context.Context, remoteType string, session *service.DeviceAuthSession, sandbox bool) (*service.DeviceTokenResult, error) {
	if f.poll != nil {
		return f.poll(ctx, remoteType, session, sandbox)
	}
	return service.PollDeviceToken(ctx, remoteType, session, sandbox)
}

func (f authFlow) userOrg(ctx context.Context, remoteType, accessToken string, sandbox bool) (*service.UserOrgInfo, error) {
	if f.org != nil {
		return f.org(ctx, remoteType, accessToken, sandbox)
	}
	return service.GetUserOrgInfo(ctx, remoteType, accessToken, sandbox)
}

func (f authFlow) accessKeys(ctx context.Context, remoteType, accessToken string, sandbox bool) ([]service.AccessKeyItem, error) {
	if f.list != nil {
		return f.list(ctx, remoteType, accessToken, sandbox)
	}
	return service.ListAccessKeys(ctx, remoteType, accessToken, sandbox)
}

func (f authFlow) createKey(ctx context.Context, remoteType, accessToken, label, accessLevel string, sandbox bool) (*service.AccessKeyItem, error) {
	if f.create != nil {
		return f.create(ctx, remoteType, accessToken, label, accessLevel, sandbox)
	}
	return service.CreateAccessKey(ctx, remoteType, accessToken, label, accessLevel, sandbox)
}

func pendingAuthPath() (string, error) {
	configPath, err := config.ResolveConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(configPath), pendingAuthFile), nil
}

func loadPendingForDevice(deviceCode string) (*pendingAuth, error) {
	path, err := pendingAuthPath()
	if err != nil {
		return nil, err
	}
	if !utils.Exists(path) {
		return nil, errNoPendingAuth()
	}
	pending := &pendingAuth{}
	if err := utils.ReadJSON(path, pending); err != nil {
		return nil, err
	}
	if pending.DeviceCode != deviceCode {
		return nil, errNoPendingAuth()
	}
	return pending, nil
}

func errNoPendingAuth() error {
	return fmt.Errorf("no pending authorization for this device code. %s", authhint.NeedLogin())
}

func savePendingAuth(pending *pendingAuth) error {
	path, err := pendingAuthPath()
	if err != nil {
		return err
	}
	return utils.WriteJSON(path, pending, true)
}

func deletePendingAuth() error {
	path, err := pendingAuthPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func writeAuthJSON(console *ui.Console, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	console.Println(strings.TrimSuffix(buf.String(), "\n"))
	return nil
}
