package task

import (
	"context"
	"fmt"

	"sensecraft-cloud-terminal/internal/config"
	"sensecraft-cloud-terminal/internal/enum"
	"sensecraft-cloud-terminal/internal/service"
	"sensecraft-cloud-terminal/internal/ui"
)

const totalSteps = 5

// InitContext 向导共享状态。
type InitContext struct {
	UI         *ui.Console
	Ctx        context.Context
	Sandbox    bool
	RemoteType string
	Session    *service.DeviceAuthSession
	Token      *service.DeviceTokenResult
	OrgInfo    *service.UserOrgInfo
	AccessKey  *service.AccessKeyItem
	ConfigPath string
}

type step func(*InitContext) (step, error)

// RunInitFlow 执行 auth init 向导。向导只保存 Access Key。
func RunInitFlow(ctx context.Context, console *ui.Console, sandbox bool) error {
	if !console.IsInteractive() && !sandbox {
		return fmt.Errorf("auth init requires an interactive terminal, or use --sandbox to try the flow")
	}

	configPath, err := config.ResolveConfigPath()
	if err != nil {
		return err
	}

	initCtx := &InitContext{
		UI:         console,
		Ctx:        ctx,
		Sandbox:    sandbox,
		ConfigPath: configPath,
	}

	console.Println("SenseCraft CLI Setup Wizard")
	console.Println("Sign in via OAuth Device Flow and save a local API config.")
	if sandbox {
		console.Println("(Sandbox mode: no real network requests will be sent)")
	}

	var current step = stepSelectRemote
	for current != nil {
		next, err := current(initCtx)
		if err != nil {
			initCtx.UI.PrintError(err.Error())
			return err
		}
		current = next
	}
	return nil
}

func stepSelectRemote(ctx *InitContext) (step, error) {
	ctx.UI.PrintStep(1, totalSteps, "Select site")
	idx, err := ctx.UI.Select("Select the SenseCAP site to connect to", []string{
		"Global (International)",
		"China (China site)",
	}, 0)
	if err != nil {
		return nil, err
	}
	if idx == 1 {
		ctx.RemoteType = enum.RemoteTypeChina
	} else {
		ctx.RemoteType = enum.RemoteTypeGlobal
	}
	ctx.UI.PrintSuccess(fmt.Sprintf("Selected %s", ctx.RemoteType))
	return stepRequestDeviceCode, nil
}

func stepRequestDeviceCode(ctx *InitContext) (step, error) {
	ctx.UI.PrintStep(2, totalSteps, "Request device authorization code")
	ctx.UI.Println("Requesting user_code from the user center ...")

	session, err := service.StartDeviceFlow(ctx.Ctx, ctx.RemoteType, ctx.Sandbox)
	if err != nil {
		return nil, err
	}
	ctx.Session = session
	ctx.UI.PrintSuccess("Received user_code")
	return stepShowAuthorization, nil
}

func stepShowAuthorization(ctx *InitContext) (step, error) {
	ctx.UI.PrintStep(3, totalSteps, "Complete authorization in the browser")
	authURL := ctx.Session.VerificationURIComplete
	if authURL == "" {
		authURL = ctx.Session.VerificationURI
	}

	ctx.UI.Println("")
	ctx.UI.Println("Copy the link below into your browser and authorize with the primary account:")
	ctx.UI.Println("")
	ctx.UI.Printf("  %s\n", authURL)
	if ctx.Session.UserCode != "" {
		ctx.UI.Println("")
		ctx.UI.Printf("  Verification code: %s\n", ctx.Session.UserCode)
	}
	ctx.UI.Println("")
	ctx.UI.Println("The CLI will continue automatically after you complete browser authorization.")
	return stepPollToken, nil
}

func stepPollToken(ctx *InitContext) (step, error) {
	ctx.UI.PrintStep(4, totalSteps, "Wait for authorization")

	var token *service.DeviceTokenResult
	err := ctx.UI.WaitSpinner("Waiting for authorization", func() error {
		var pollErr error
		token, pollErr = service.PollDeviceToken(ctx.Ctx, ctx.RemoteType, ctx.Session, ctx.Sandbox)
		return pollErr
	})
	if err != nil {
		return nil, err
	}
	ctx.Token = token
	ctx.UI.PrintSuccess("Authorization succeeded, access_token obtained")
	return stepAccessKeyAndSave, nil
}

func stepAccessKeyAndSave(ctx *InitContext) (step, error) {
	ctx.UI.PrintStep(5, totalSteps, "Configure Access Key and save")

	org, err := service.GetUserOrgInfo(ctx.Ctx, ctx.RemoteType, ctx.Token.AccessToken, ctx.Sandbox)
	if err != nil {
		return nil, err
	}
	ctx.OrgInfo = org
	ctx.UI.Printf("  Account: %s\n", org.Account)
	ctx.UI.Printf("  Organization: %s (%s)\n", org.Organization, org.OrgID)
	ctx.UI.Println("")

	keys, err := service.ListAccessKeys(ctx.Ctx, ctx.RemoteType, ctx.Token.AccessToken, ctx.Sandbox)
	if err != nil {
		return nil, err
	}

	selected, err := selectOrCreateAccessKey(ctx, keys)
	if err != nil {
		return nil, err
	}
	ctx.AccessKey = selected

	profile := config.BuildAccessProfile(
		ctx.RemoteType,
		org.OrgID,
		selected.AccessID,
		selected.AccessKey,
	)
	if err := config.UpsertAccessProfile(profile, true); err != nil {
		return nil, err
	}

	ctx.UI.Println("")
	ctx.UI.PrintSuccess("SenseCAP setup complete")
	ctx.UI.Printf("  Key ID: %s\n", selected.AccessID)
	ctx.UI.Printf("  Organization ID: %s\n", org.OrgID)
	ctx.UI.Printf("  Config file: %s\n", ctx.ConfigPath)
	ctx.UI.Println("")
	ctx.UI.Println("You can now run: sensecraft-cli device list")
	return nil, nil
}

func selectOrCreateAccessKey(ctx *InitContext, keys []service.AccessKeyItem) (*service.AccessKeyItem, error) {
	if len(keys) == 0 {
		ctx.UI.Println("No Access Keys found for this account; a new one will be created.")
		return promptCreateAccessKey(ctx)
	}

	ctx.UI.Println("The following Access Keys were found:")
	options := make([]string, 0, len(keys)+1)
	for _, k := range keys {
		options = append(options, formatAccessKeyOption(k))
	}
	options = append(options, "Create a new Access Key")

	idx, err := ctx.UI.Select("Select an existing Access Key, or create a new one", options, 0)
	if err != nil {
		return nil, err
	}
	if idx < len(keys) {
		selected := keys[idx]
		return &selected, nil
	}
	return promptCreateAccessKey(ctx)
}

func promptCreateAccessKey(ctx *InitContext) (*service.AccessKeyItem, error) {
	label, err := ctx.UI.Input("Enter a label for the new Access Key", "")
	if err != nil {
		return nil, err
	}
	if label == "" {
		return nil, fmt.Errorf("label cannot be empty")
	}

	levelIdx, err := ctx.UI.Select("Select permission level", []string{
		"read-write",
		"only-read",
	}, 0)
	if err != nil {
		return nil, err
	}
	level := enum.ApiAccessLevelRW
	if levelIdx == 1 {
		level = enum.ApiAccessLevelOR
	}

	ctx.UI.Println("Creating Access Key ...")
	return service.CreateAccessKey(ctx.Ctx, ctx.RemoteType, ctx.Token.AccessToken, label, level, ctx.Sandbox)
}

func formatAccessKeyOption(k service.AccessKeyItem) string {
	label := k.Label
	if label == "" {
		label = "(no label)"
	}
	return fmt.Sprintf("%s  access_id=%s  level=%s", label, k.AccessID, k.AccessLevel)
}
