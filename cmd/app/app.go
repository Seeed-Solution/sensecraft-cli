package app

import (
	"encoding/json"
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
	cmd.AddCommand(newAppActivityCmd())
	cmd.AddCommand(newAppViewCmd())
	cmd.AddCommand(newAppFieldsCmd())
	cmd.AddCommand(newAppBlocksCmd())
	cmd.AddCommand(newAppCreateCmd())
	cmd.AddCommand(newAppUpdateCmd())
	cmd.AddCommand(newAppPublishCmd())
	cmd.AddCommand(newAppUnpublishCmd())
	cmd.AddCommand(newAppWithdrawCmd())
	cmd.AddCommand(newAppDeleteCmd())
	cmd.AddCommand(newAppLikeCmd())
	cmd.AddCommand(newAppUnlikeCmd())
	cmd.AddCommand(newAppCollectCmd())
	cmd.AddCommand(newAppUncollectCmd())
	cmd.AddCommand(newAppMarkDeployCmd())
	cmd.AddCommand(newAppCloneCmd())
	cmd.AddCommand(newAppBOMCmd())
	return cmd
}

func newAppListCmd() *cobra.Command {
	var (
		public             bool
		mine               bool
		name               string
		scene              string
		uniformType        string
		page               int
		length             int
		markType           string
		sort               string
		priorityActivityID string
		tagID              string
		likes              bool
		collections        bool
		sandbox            bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List applications (plaza or my apps)",
		Long: `List SenseCraft apps via {ai-host}/openapi (Access Key Basic).

  Plaza:            --public
  My apps:          --mine, or omit both visibility flags
  Plaza hot sort:   --public --sort popular
  --name            Filter by app_name
  --scene           Scene id
  --uniform-type    Device type (list_app uniform_type: reCamera=40, XIAO ESP32S3=32)
  --sort            popular | newest | likes. Omitted: do not send sort_by
  --priority-activity-id
                    Only with --sort popular. Omitted on popular: first ongoing activity
  --tag-id          Competition filter (activity tag id)
  --mark-type       1 = my likes, 0 = my collections
  --likes           mark_type=1 and be_public=1. Not with --mine
  --collections     mark_type=0 and be_public=1. Not with --mine
  --page / --length Pagination (default 1 / 20)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			if !public && !mine && !likes && !collections {
				mine = true
			}
			if public && mine {
				return fmt.Errorf("use only one of --public or --mine")
			}
			data, err := service.GetAppService(ctx).ListApps(service.AppListParams{
				Public:             public,
				Mine:               mine,
				Name:               name,
				Scene:              scene,
				UniformType:        uniformType,
				Page:               page,
				Length:             length,
				MarkType:           markType,
				Sort:               sort,
				PriorityActivityID: priorityActivityID,
				TagID:              tagID,
				Likes:              likes,
				Collections:        collections,
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
	cmd.Flags().StringVar(&markType, "mark-type", "", "1 = my likes, 0 = my collections")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort: popular, newest, or likes")
	cmd.Flags().StringVar(&priorityActivityID, "priority-activity-id", "", "With --sort popular, pin this activity first (positive integer)")
	cmd.Flags().StringVar(&tagID, "tag-id", "", "Competition tag id (list_app tag_id)")
	cmd.Flags().BoolVar(&likes, "likes", false, "My likes (mark_type=1, be_public=1). Not with --mine")
	cmd.Flags().BoolVar(&collections, "collections", false, "My collections (mark_type=0, be_public=1). Not with --mine")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	return cmd
}

func newAppActivityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activity",
		Short: "Ongoing SenseCraft activities",
	}
	cmd.AddCommand(newAppActivityListCmd())
	return cmd
}

func newAppActivityListCmd() *cobra.Command {
	var sandbox bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List ongoing activities (read-only)",
		Long: `GET /activity/list with no time filter.

Prints { "list": [ { activity_id, name, tag_ids, start_time, end_time } ] }.
Rows missing a positive activity_id or a name are dropped. No --yes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetAppService(ctx).ListActivities()
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
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
		id         string
		fromJSON   string
		activityID string
		yes        bool
		sandbox    bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an app from JSON (L2: --yes)",
		Long: `POST /flow/app/v3/update_app

Requires --id and at least one of --from-json / --activity-id. Injects app_id.
Rejects user JSON be_public=1 / privacy=1.

--activity-id joins an activity (positive integer, sent as a JSON number).
Omit it, or pass an empty value, to leave existing activity tags unchanged.
If the activity is not in app activity list, update_app is not sent.
When --activity-id is set and the JSON omits allow_clone, CLI sets allow_clone
to 1 unless the sketch is already 1.
tags in JSON is ignored and is not sent.

Lifecycle (matches Console edit.vue / list.vue):
  owner only (user_id == profile) — another user's app cannot be edited
  be_public=0  private — edit/delete/publish allowed
  be_public=1  published — owner may edit (saved as sketch; CLI sends be_public=1). Plaza unchanged until app publish (draft).
  be_public=2  under review — locked; no update/cover/BOM. Unpublish to unlock.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			if strings.TrimSpace(fromJSON) == "" && !cmd.Flags().Changed("activity-id") {
				return fmt.Errorf("provide --from-json and/or --activity-id")
			}
			data, err := service.GetAppService(ctx).UpdateApp(service.AppUpdateParams{
				AppID:       id,
				FromJSON:    fromJSON,
				Yes:         yes,
				ActivitySet: cmd.Flags().Changed("activity-id"),
				ActivityID:  activityID,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "app_id (required)")
	cmd.Flags().StringVar(&fromJSON, "from-json", "", "JSON file path or - for stdin")
	cmd.Flags().StringVar(&activityID, "activity-id", "", "Join this activity (positive integer). Empty leaves current activity tags unchanged")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 update after user approval")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("id")
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

Matches the Console publish confirmation: user must agree to the Publish Declaration
(https://sensecraft.seeed.cc/legal/en/declaration) via --agree-declaration.
Submitting locks editing until review finishes (or unpublish).

Private apps: submit for first public listing.
Already published (be_public=1) with a sketch: same API as the Console publish-draft action.
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
		Short: "Make a public app private (privacy=0, L2: --yes)",
		Long: `POST /flow/app/v3/edit_privacy_app with privacy=0.

Use this for an already public app (be_public=1). It does not withdraw a
review. An app under review (be_public=2) uses: app withdraw --id <id> --yes`,
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

func newAppWithdrawCmd() *cobra.Command {
	return newAppIDYesCmd("withdraw", "Withdraw an app that is under review (L2: --yes)",
		`POST /flow/app/v3/callback_sketch with {app_id} only.

Refuses unless be_public=2. A public app uses app unpublish. A private app
has nothing to withdraw.`,
		func(ctx cmdctx.AICmdContext, id string, yes bool) (json.RawMessage, error) {
			return service.GetAppService(ctx).WithdrawApp(service.AppDeleteParams{AppID: id, Yes: yes})
		})
}

func newAppLikeCmd() *cobra.Command {
	return newAppMarkCmd("like", "Like an app (op_type=1, L2: --yes)", 1)
}

func newAppUnlikeCmd() *cobra.Command {
	return newAppMarkCmd("unlike", "Remove a like (op_type=0, L2: --yes)", 0)
}

func newAppCollectCmd() *cobra.Command {
	return newAppMarkCmd("collect", "Collect an app (op_type=3, L2: --yes)", 3)
}

func newAppUncollectCmd() *cobra.Command {
	return newAppMarkCmd("uncollect", "Remove a collection (op_type=2, L2: --yes)", 2)
}

func newAppMarkDeployCmd() *cobra.Command {
	return newAppMarkCmd("mark-deploy", "Record one public-detail deploy (op_type=4, L2: --yes)", 4)
}

func newAppMarkCmd(use, short string, op int) *cobra.Command {
	return newAppIDYesCmd(use, short,
		`POST /flow/app/v2/mark_app. Does not flash a device.
stdout includes app_id and op_type even when the API body is empty.`,
		func(ctx cmdctx.AICmdContext, id string, yes bool) (json.RawMessage, error) {
			return service.GetAppService(ctx).MarkApp(id, op, yes)
		})
}

func newAppCloneCmd() *cobra.Command {
	var (
		id      string
		name    string
		yes     bool
		sandbox bool
	)
	cmd := &cobra.Command{
		Use:   "clone",
		Short: "Clone an app into my apps (L2: --yes)",
		Long: `POST /flow/app/v3/clone_app.

Refuses when the source allow_clone is 0. Without --name the new name is
Clone_ plus the first 54 characters of the source name.
Does not copy be_public and does not publish.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetAppService(ctx).CloneApp(service.AppCloneParams{
				AppID: id,
				Name:  name,
				Yes:   yes,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Source app_id (required)")
	cmd.Flags().StringVar(&name, "name", "", "New app name (default Clone_ + source name)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 clone after user approval")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newAppIDYesCmd(use, short, long string, run func(ctx cmdctx.AICmdContext, id string, yes bool) (json.RawMessage, error)) *cobra.Command {
	var (
		id      string
		yes     bool
		sandbox bool
	)
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := run(ctx, id, yes)
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "app_id (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 write after user approval")
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
