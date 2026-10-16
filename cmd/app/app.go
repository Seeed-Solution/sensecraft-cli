package app

import (
	"fmt"
	"strings"

	cmdctx "sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/service"

	"github.com/spf13/cobra"
)

func NewAppCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "SenseCraft Application Plaza (requires Access Key /openapi)",
	}
	cmd.AddCommand(newAppListCmd())
	cmd.AddCommand(newAppViewCmd())
	cmd.AddCommand(newAppFieldsCmd())
	cmd.AddCommand(newAppBlocksCmd())
	cmd.AddCommand(newAppCloneCmd())
	cmd.AddCommand(newAppCreateCmd())
	cmd.AddCommand(newAppUpdateCmd())
	cmd.AddCommand(newAppPublishCmd())
	cmd.AddCommand(newAppUnpublishCmd())
	cmd.AddCommand(newAppDeleteCmd())
	cmd.AddCommand(newAppBOMCmd())
	return cmd
}

func newAppListCmd() *cobra.Command {
	var (
		public      bool
		mine        bool
		name        string
		scene       string
		uniformType string
		page        int
		length      int
		markType    string
		sandbox     bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List applications (plaza or my apps)",
		Long: `List SenseCraft apps via {ai-host}/openapi (Access Key Basic).

  --public         Application Plaza (be_public=1)
  --mine           My apps (omit be_public; default if neither flag set)
  --name           Filter by app_name
  --scene          Scene id
  --uniform-type   Device type (list_app uniform_type: reCamera=40, XIAO ESP32S3=32)
  --page / --length  Pagination (default 1 / 20)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			if !public && !mine {
				mine = true
			}
			if public && mine {
				return fmt.Errorf("use only one of --public or --mine")
			}
			data, err := service.GetAppService(ctx).ListApps(service.AppListParams{
				Public:      public,
				Mine:        mine,
				Name:        name,
				Scene:       scene,
				UniformType: uniformType,
				Page:        page,
				Length:      length,
				MarkType:    markType,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().BoolVar(&public, "public", false, "List public Application Plaza apps (be_public=1)")
	cmd.Flags().BoolVar(&mine, "mine", false, "List my apps")
	cmd.Flags().StringVar(&name, "name", "", "Filter by app name")
	cmd.Flags().StringVar(&scene, "scene", "", "Filter by scene id")
	cmd.Flags().StringVar(&uniformType, "uniform-type", "", "Filter by device type (uniform_type: 40=reCamera, 32=XIAO ESP32S3, 53=reCamera Pro)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&length, "length", 20, "Page size")
	cmd.Flags().StringVar(&markType, "mark-type", "", "mark_type filter (like/star when supported)")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	return cmd
}

func newAppViewCmd() *cobra.Command {
	var (
		id      string
		sketch  bool
		sandbox bool
	)
	cmd := &cobra.Command{
		Use:   "view",
		Short: "View application detail (v3)",
		Long: `GET /flow/app/v3/view_app

  --id       app_id (required)
  --sketch   view_sketch=1 (draft); default published (0)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetAppService(ctx).ViewApp(service.AppViewParams{
				AppID:      id,
				ViewSketch: sketch,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Application id (required)")
	cmd.Flags().BoolVar(&sketch, "sketch", false, "View draft (view_sketch=1)")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newAppFieldsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fields",
		Short: "Print app info + BOM field schema (for Agents)",
		Long:  `Static schema aligned with Console custom-app v3 fields. No network required.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			return ctx.PrintRawData(service.AppFieldsSchema())
		},
	}
}

func newAppBlocksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "blocks",
		Short: "Print deploy doc + tool-block catalog (for Agents)",
		Long:  `Static catalog of text/tool blocks and when to use them. No network required.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			return ctx.PrintRawData(service.AppBlocksCatalog())
		},
	}
}

func newAppCloneCmd() *cobra.Command {
	var (
		id              string
		name            string
		yes             bool
		agreeDisclaimer bool
		sandbox         bool
	)
	cmd := &cobra.Command{
		Use:   "clone",
		Short: "Clone an app into My Apps (L2: --agree-disclaimer --yes)",
		Long: `POST /flow/app/v3/clone_app

Matches Console 「克隆应用」dialog: user must agree to the Disclaimer for Use
(https://sensecraft.seeed.cc/legal/en/disclaimer) via --agree-disclaimer,
then confirm with --yes. Agreement is client-side only (API body is app_id + app_name).
Default name is Clone_<source app_name> (max 60 with prefix).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetAppService(ctx).CloneApp(service.AppCloneParams{
				AppID:           id,
				Name:            name,
				Yes:             yes,
				AgreeDisclaimer: agreeDisclaimer,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Source app_id (required)")
	cmd.Flags().StringVar(&name, "name", "", "Cloned app_name (default Clone_<source>)")
	cmd.Flags().BoolVar(&agreeDisclaimer, "agree-disclaimer", false, "Confirm user read and agreed to the Disclaimer for Use (Console disclaimerAccepted)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 clone after user approval")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newAppCreateCmd() *cobra.Command {
	var (
		name     string
		fromJSON string
		sandbox  bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an app (--name and/or --from-json)",
		Long: `Create a new app. Prefer --from-json with a full payload when ready (no --yes).

  --name          required unless provided in --from-json.app_name
  --from-json     JSON path or "-" (stdin). create_app(app_name) then, if extra
                  fields remain, update_app on the **new** app_id only.
  Not L2: create never requires --yes (unlike app update on an existing app).
  Never include be_public=1 / privacy=1; use app publish --agree-declaration --yes after confirmation.
  On success, stdout includes app_id, console_url (edit page), and detail_url.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			if strings.TrimSpace(name) == "" && strings.TrimSpace(fromJSON) == "" {
				return fmt.Errorf("provide --name and/or --from-json")
			}
			data, err := service.GetAppService(ctx).CreateApp(service.AppCreateParams{
				Name:     name,
				FromJSON: fromJSON,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "app_name (max 60)")
	cmd.Flags().StringVar(&fromJSON, "from-json", "", "JSON file path or - for stdin")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	return cmd
}

func newAppUpdateCmd() *cobra.Command {
	var (
		id       string
		fromJSON string
		yes      bool
		sandbox  bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an app from JSON (L2: --yes)",
		Long: `POST /flow/app/v3/update_app

Requires --id and --from-json. Injects app_id. Rejects user JSON be_public=1 / privacy=1.

Lifecycle (matches Console edit.vue / list.vue):
  owner only (user_id == profile) — others: clone first
  be_public=0  private — edit/delete/publish allowed
  be_public=1  published — owner may edit (saved as sketch; CLI sends be_public=1). Plaza unchanged until app publish (draft).
  be_public=2  under review — locked; no update/cover/BOM. Unpublish to unlock.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetAppService(ctx).UpdateApp(service.AppUpdateParams{
				AppID:    id,
				FromJSON: fromJSON,
				Yes:      yes,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "app_id (required)")
	cmd.Flags().StringVar(&fromJSON, "from-json", "", "JSON file path or - for stdin (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 update after user approval")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("from-json")
	return cmd
}

func newAppPublishCmd() *cobra.Command {
	var (
		id               string
		yes              bool
		agreeDeclaration bool
		sandbox          bool
	)
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Submit app for public review (privacy=2, L2: --agree-declaration --yes)",
		Long: `POST /flow/app/v3/edit_privacy_app with privacy=2 (under review).
Never sends privacy=1 (ops-approved public).

Matches Console 「发布确认」: user must agree to the Publish Declaration
(https://sensecraft.seeed.cc/legal/en/declaration) via --agree-declaration.
Submitting locks editing until review finishes (or unpublish).

Private apps: submit for first public listing.
Already published (be_public=1) with a sketch: same API as Console 「发布草稿」 / publishDraft.
Under review (be_public=2): refused until unpublish.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetAppService(ctx).SetAppPrivacy(service.AppPrivacyParams{
				AppID:            id,
				Privacy:          2,
				Yes:              yes,
				AgreeDeclaration: agreeDeclaration,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "app_id (required)")
	cmd.Flags().BoolVar(&agreeDeclaration, "agree-declaration", false, "Confirm user read and agreed to the Publish Declaration (Console warrantyAccepted)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 publish after user approval")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newAppUnpublishCmd() *cobra.Command {
	var (
		id      string
		yes     bool
		sandbox bool
	)
	cmd := &cobra.Command{
		Use:   "unpublish",
		Short: "Make app private (privacy=0, L2: --yes)",
		Long:  `POST /flow/app/v3/edit_privacy_app with privacy=0.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetAppService(ctx).SetAppPrivacy(service.AppPrivacyParams{
				AppID:   id,
				Privacy: 0,
				Yes:     yes,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "app_id (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 unpublish after user approval")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newAppDeleteCmd() *cobra.Command {
	var (
		id      string
		yes     bool
		sandbox bool
	)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a private app (L2: requires --yes)",
		Long:  `POST /flow/app/v2/delete_app. Only be_public=0 (private). Published / under-review: unpublish first.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetAppService(ctx).DeleteApp(service.AppDeleteParams{
				AppID: id,
				Yes:   yes,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "app_id (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 delete after user approval")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newAppBOMCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bom",
		Short: "Bill of Materials (Seeed catalog + app materials)",
	}
	cmd.AddCommand(newAppBOMSearchCmd())
	cmd.AddCommand(newAppBOMListCmd())
	cmd.AddCommand(newAppBOMSetCmd())
	return cmd
}

func newAppBOMSearchCmd() *cobra.Command {
	var (
		name    string
		sandbox bool
	)
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search Seeed material_options (local filter on get_production_options)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetAppService(ctx).SearchBOMProducts(service.AppBOMSearchParams{Name: name})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Filter keyword (name/sku/classify)")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode")
	return cmd
}

func newAppBOMListCmd() *cobra.Command {
	var (
		id      string
		sketch  bool
		sandbox bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List materials on an app (from view_app)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetAppService(ctx).ListBOM(id, sketch)
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "app_id (required)")
	cmd.Flags().BoolVar(&sketch, "sketch", false, "view_sketch=1")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newAppBOMSetCmd() *cobra.Command {
	var (
		id       string
		fromJSON string
		yes      bool
		sandbox  bool
	)
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Replace app materials via update_app (L2: --yes)",
		Long: `POST update_app with materials[]. Refused when the app is under review (be_public=2).

Seeed row: id(=SKU), name, quantity, note, buy_url
Custom row: omit id; name, quantity, note, buy_url`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetAppService(ctx).SetBOM(service.AppBOMSetParams{
				AppID:    id,
				FromJSON: fromJSON,
				Yes:      yes,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "app_id (required)")
	cmd.Flags().StringVar(&fromJSON, "from-json", "", "JSON object with materials[] (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 BOM update")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("from-json")
	return cmd
}
