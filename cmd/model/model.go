package model

import (
	"encoding/json"
	"fmt"

	cmdctx "sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/service"
	"sensecraft-cloud-terminal/internal/task"
	"sensecraft-cloud-terminal/internal/ui"

	"github.com/spf13/cobra"
)

func NewModelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "SenseCraft model library (requires Access Key /openapi)",
	}
	cmd.AddCommand(newModelListCmd())
	cmd.AddCommand(newModelViewCmd())
	cmd.AddCommand(newModelDownloadCmd())
	cmd.AddCommand(newModelUploadCmd())
	cmd.AddCommand(newModelConstantsCmd())
	cmd.AddCommand(newModelCreateCmd())
	cmd.AddCommand(newModelUpdateCmd())
	cmd.AddCommand(newModelPublishCmd())
	cmd.AddCommand(newModelUnpublishCmd())
	cmd.AddCommand(newModelDeleteCmd())
	cmd.AddCommand(newModelRecameraProCmd())
	return cmd
}

func newModelRecameraProCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recamera-pro",
		Short: "reCamera Pro model compatibility helpers",
	}
	cmd.AddCommand(newModelRecameraProResolveCmd())
	return cmd
}

func newModelRecameraProResolveCmd() *cobra.Command {
	var (
		task        int
		aiFramework string
	)
	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Validate task/framework and resolve device algorithm",
		Long: `Resolve SenseCraft model task + ai_framework to the reCamera Pro device algorithm.

This is a local validation helper and does not require authentication or contact a device.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.ValidateRecameraProAlgorithm(task, aiFramework)
			raw, marshalErr := json.MarshalIndent(result, "", "  ")
			if marshalErr != nil {
				return marshalErr
			}
			if _, writeErr := fmt.Fprintln(cmd.OutOrStdout(), string(raw)); writeErr != nil {
				return writeErr
			}
			return err
		},
	}
	cmd.Flags().IntVar(&task, "task", 0, "model_task_array id (required)")
	cmd.Flags().StringVar(&aiFramework, "ai-framework", "", "ai_framework_array id (required for detection/pose)")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}

func newModelListCmd() *cobra.Command {
	var (
		public      bool
		mine        bool
		name        string
		task        string
		uniformType string
		adapteds    string
		page        int
		length      int
		sandbox     bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List models (public library or my models)",
		Long: `List SenseCraft models via {ai-host}/openapi (Access Key Basic).

  --public         Public model library (list_model); default if neither flag set
  --mine           My models (list_custom_model)
  --name           Search keyword (API query "search", same as Console model library)
  --task           Task id from get_constant model_task_array
  --uniform-type   Device type (list_model uniform_type: reCamera=40, XIAO ESP32S3=32)
  --adapteds       Device id (Console cardList adapteds; optional SKU-level filter)
  --page / --length  Pagination (default 1 / 20)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			if !public && !mine {
				public = true
			}
			if public && mine {
				return fmt.Errorf("use only one of --public or --mine")
			}
			data, err := service.GetModelService(ctx).ListModels(service.ModelListParams{
				Public:      public,
				Mine:        mine,
				Name:        name,
				Task:        task,
				UniformType: uniformType,
				Adapteds:    adapteds,
				Page:        page,
				Length:      length,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(service.RedactPublicModelArtifactURLs(data, public))
		},
	}
	cmd.Flags().BoolVar(&public, "public", false, "List public model library")
	cmd.Flags().BoolVar(&mine, "mine", false, "List my models")
	cmd.Flags().StringVar(&name, "name", "", "Search keyword (API param search; same as Console model library)")
	cmd.Flags().StringVar(&task, "task", "", "Filter by task id")
	cmd.Flags().StringVar(&uniformType, "uniform-type", "", "Filter by device type (uniform_type: 40=reCamera, 32=XIAO ESP32S3, 53=reCamera Pro)")
	cmd.Flags().StringVar(&adapteds, "adapteds", "", "Filter by device id (list_model adapteds, same as Console)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&length, "length", 20, "Page size")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	return cmd
}

func newModelViewCmd() *cobra.Command {
	var (
		id      string
		sandbox bool
	)
	cmd := &cobra.Command{
		Use:   "view",
		Short: "View model detail",
		Long: `GET /model/view_model — returns name, task, format, classes, devices, etc.

Public library models omit file_url. Artifacts are pulled as firmware at deploy time.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetModelService(ctx).ViewModel(service.ModelViewParams{ModelID: id})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(service.RedactPublicModelArtifactURLs(data, false))
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Model id (required)")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newModelDownloadCmd() *cobra.Command {
	var (
		id      string
		out     string
		apply   bool
		urlOnly bool
		sandbox bool
	)
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download private model artifact (file_url or apply_model)",
		Long: `Download a private (My Models) SenseCraft model binary.

  Public library models cannot be downloaded. Deploy instead so the device
  pulls firmware at apply time (cloud deploy or Console workspace).

  Default source: view_model.file_url (owner library download)
  --apply:        use apply_model → model_snapshot.arguments.url (private deploy snapshot)
  --url-only:     print JSON with url only; do not write a file (private models only)
  --out:          output file or directory (default: basename from URL)

  File downloads omit url from JSON unless --url-only is set.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			result, err := service.GetModelService(ctx).DownloadModel(service.ModelDownloadParams{
				ModelID: id,
				Out:     out,
				Apply:   apply,
				URLOnly: urlOnly,
			})
			if err != nil {
				return err
			}
			return ctx.PrintJsonData(result)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Model id (required)")
	cmd.Flags().StringVar(&out, "out", "", "Output file or directory")
	cmd.Flags().BoolVar(&apply, "apply", false, "Use apply_model deploy snapshot URL instead of file_url")
	cmd.Flags().BoolVar(&urlOnly, "url-only", false, "Return download URL JSON without writing a file")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; write sample bytes / return sample URL")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newModelUploadCmd() *cobra.Command {
	var (
		path     string
		id       string
		yes      bool
		unlinked bool
		sandbox  bool
	)
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a model binary; requires --id attach or --unlinked for create",
		Long: `Uploads file_type=2 with checksum (same as file upload --type 2 --checksum).

Association is required before any OSS PUT (avoids orphan static files):

  # Prefer: attach to an existing private model (bumps patch version)
  sensecraft-cli model upload --path ./model.tflite --id <model_id> --yes

  # Brand-new model only: explicit opt-in, then put file_url into model create
  sensecraft-cli model upload --path ./model.tflite --unlinked

Without --id or --unlinked: interactive TTY guides association; agents (non-TTY)
get ErrAttachRequired and must ask the user first — do not upload then attach.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			assoc, err := task.ResolveModelUploadAssoc(ui.DefaultConsole(), id, unlinked, yes)
			if err != nil {
				return err
			}
			result, err := service.GetModelService(ctx).UploadAndAttach(service.ModelUploadParams{
				Path:     path,
				ModelID:  assoc.ModelID,
				Yes:      assoc.Yes,
				Unlinked: assoc.Unlinked,
			})
			if err != nil {
				return err
			}
			if !result.Attached && result.Hint != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), result.Hint)
			}
			return ctx.PrintJsonData(result)
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "Local model file path (required)")
	cmd.Flags().StringVar(&id, "id", "", "Existing model id to attach this file to (bumps version)")
	cmd.Flags().BoolVar(&unlinked, "unlinked", false, "Allow orphan OSS object for brand-new model create (put file_url into create JSON)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 attach/update when --id is set")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample file_url")
	_ = cmd.MarkFlagRequired("path")
	return cmd
}

func newModelConstantsCmd() *cobra.Command {
	var sandbox bool
	cmd := &cobra.Command{
		Use:   "constants",
		Short: "Dump get_constant (tasks/formats/frameworks/scenarios)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetModelService(ctx).Constants()
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode")
	return cmd
}

func newModelCreateCmd() *cobra.Command {
	var (
		fromJSON string
		target   string
		yes      bool
		sandbox  bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a model from JSON (L2: --yes)",
		Long: `POST /model/create_model

Requires a complete Console-compatible model payload. The CLI loads get_constant and
validates required fields, labels, version, device families, model format, and file extension.
Do not set be_public=1; use model publish --yes after confirmation.
Public library models cannot be cloned: do not pass another user's public model id / model_id / from_id.
reCamera Pro rules activate automatically when adapteds resolves to uniform_type=53.
Optional --target recamera-pro additionally asserts that the payload actually targets Pro.
On success, stdout includes model_id and console_url (Console model detail page).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetModelService(ctx).CreateModel(service.ModelCreateParams{
				FromJSON: fromJSON,
				Target:   target,
				Yes:      yes,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&fromJSON, "from-json", "", "JSON file path or - for stdin (required)")
	cmd.Flags().StringVar(&target, "target", "", "Optional target assertion (recamera-pro; Pro is auto-detected)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 create after user approval")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("from-json")
	return cmd
}

func newModelUpdateCmd() *cobra.Command {
	var (
		id       string
		fromJSON string
		path     string
		target   string
		yes      bool
		sandbox  bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a model from JSON and/or a new file (L2: --yes)",
		Long: `POST /model/edit_model — requires --id and at least one of --from-json / --path.

Same as Console model edit (add.vue):
  - --from-json is an incremental patch, merged with view_model then validated.
  - --path uploads the binary (type=2 + checksum) and writes file_url/checksum/model_size.
  - If labels or file_url change and version was left unchanged, patch version auto-increments
    (incrementVersion: 1.0.0 to 1.0.1).
  - If the model is already public (be_public=1), the save re-enters review
    (Console extra confirm + edit_model be_public=1). --yes is that confirm.
  - Someone else's public-library id is refused. Delete is only for private models.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetModelService(ctx).UpdateModel(service.ModelUpdateParams{
				ModelID:  id,
				FromJSON: fromJSON,
				Path:     path,
				Target:   target,
				Yes:      yes,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Model id (required)")
	cmd.Flags().StringVar(&fromJSON, "from-json", "", "JSON patch file path or - for stdin")
	cmd.Flags().StringVar(&path, "path", "", "Local model file to upload and attach (bumps version like Console)")
	cmd.Flags().StringVar(&target, "target", "", "Optional target assertion (recamera-pro; Pro is auto-detected)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 update after user approval")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newModelPublishCmd() *cobra.Command {
	var (
		id      string
		yes     bool
		sandbox bool
	)
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish model to public library (status=1, L2: --yes)",
		Long: `POST /model/publish_model with status=1.
Uses model API status (not app privacy). Confirm with user before --yes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetModelService(ctx).PublishModel(service.ModelPublishParams{
				ModelID: id,
				Status:  1,
				Yes:     yes,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Model id (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 publish after user approval")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newModelUnpublishCmd() *cobra.Command {
	var (
		id      string
		yes     bool
		sandbox bool
	)
	cmd := &cobra.Command{
		Use:   "unpublish",
		Short: "Privatize model (status=0, L2: --yes)",
		Long:  `POST /model/publish_model with status=0.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetModelService(ctx).PublishModel(service.ModelPublishParams{
				ModelID: id,
				Status:  0,
				Yes:     yes,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Model id (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 unpublish after user approval")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newModelDeleteCmd() *cobra.Command {
	var (
		id      string
		yes     bool
		sandbox bool
	)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a model (L2: requires --yes)",
		Long: `POST /model/delete_model — owner private models only (be_public=0).
Published or under-review models must be unpublished first (same as Console).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetModelService(ctx).DeleteModel(service.ModelDeleteParams{
				ModelID: id,
				Yes:     yes,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Model id (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 delete after user approval")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}
