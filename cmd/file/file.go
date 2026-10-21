package file

import (
	"fmt"
	"strings"

	cmdctx "sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/service"
	"sensecraft-cloud-terminal/internal/task"
	"sensecraft-cloud-terminal/internal/ui"

	"github.com/spf13/cobra"
)

func NewFileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file",
		Short: "SenseCraft file upload (acquire_file_url + PUT)",
	}
	cmd.AddCommand(newFileUploadCmd())
	return cmd
}

func newFileUploadCmd() *cobra.Command {
	var (
		path     string
		fileType int
		checksum bool
		modelID  string
		appID    string
		forApp   bool
		unlinked bool
		yes      bool
		sandbox  bool
	)
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a local file; type 1/2 require attach or --unlinked",
		Long: `GET /user/acquire_file_url then PUT bytes to upload_url.

file_type (Console upload.vue):
  1=image  2=file/model (default)  3=app  4=firmware  5=doc  6=archive/zip

Size limits (same as Console + CLI security gates):
  type 1 default / model pic_url: ≤2 MB (model/add.vue max-size=2)
  type 1 --for-app or --app-id:   ≤5 MB (infoForm max-size=5; max 16 app images)
  type 2–6:                       ≤100 MB
  empty (0 B) files are rejected
  Formats for type 1: JPG/PNG/GIF/WebP (no SVG)

Association (types 1–2) is required before any OSS PUT:
  --model-id <id> --yes   type=2 + checksum; writes file onto the model and bumps version
  --app-id <id> --yes     type=1 cover image; prepends images[] (max 16 total; 5 MB)
  --for-app               type=1 unlinked for later app images[] JSON (5 MB)
  --unlinked              explicit orphan for later create/update JSON (pic_url / model create)

Types 3–6 have no model/app attach path and upload directly.
Without association flags: interactive TTY guides; agents must ask first.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cmdctx.GetAICmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			if strings.TrimSpace(path) == "" {
				return fmt.Errorf("--path is required")
			}
			assoc, err := task.ResolveFileUploadAssoc(
				ui.DefaultConsole(), fileType, modelID, appID, forApp, unlinked, yes,
			)
			if err != nil {
				return err
			}
			modelID = assoc.ModelID
			appID = assoc.AppID
			forApp = assoc.ForApp
			unlinked = assoc.Unlinked
			yes = assoc.Yes

			if modelID != "" {
				result, err := service.GetModelService(ctx).UploadAndAttach(service.ModelUploadParams{
					Path:    path,
					ModelID: modelID,
					Yes:     yes,
				})
				if err != nil {
					return err
				}
				return ctx.PrintJsonData(result)
			}
			if appID != "" {
				if fileType != 0 && fileType != service.FileTypeImage {
					return fmt.Errorf("attaching to --app-id currently supports --type 1 (cover image)")
				}
				if err := service.RequireYes(yes); err != nil {
					return fmt.Errorf("%w; --app-id attaches the image as the app cover", err)
				}
				up, err := service.GetFileService(ctx).UploadFile(service.FileUploadParams{
					Path:        path,
					FileType:    service.FileTypeImage,
					Checksum:    checksum,
					ForAppImage: true,
				})
				if err != nil {
					return err
				}
				if _, err := service.GetAppService(ctx).AttachCoverImage(appID, up.FileURL, true); err != nil {
					return err
				}
				result := service.FileAttachFromUpload(up)
				result.Attached = true
				result.AppID = appID
				return ctx.PrintJsonData(result)
			}
			if forApp {
				if fileType != 0 && fileType != service.FileTypeImage {
					return fmt.Errorf("--for-app requires --type 1 (app image)")
				}
				fileType = service.FileTypeImage
			}
			result, err := service.GetFileService(ctx).UploadFile(service.FileUploadParams{
				Path:        path,
				FileType:    fileType,
				Checksum:    checksum,
				ForAppImage: forApp,
			})
			if err != nil {
				return err
			}
			out := service.FileAttachFromUpload(result)
			// Types 1–2 reach here only via --unlinked / --for-app (or type 3–6).
			if fileType == 0 || fileType == service.FileTypeFile || fileType == service.FileTypeImage {
				out.Hint = service.UnlinkedUploadHint
				fmt.Fprintln(cmd.ErrOrStderr(), out.Hint)
			}
			return ctx.PrintJsonData(out)
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "Local file path (required)")
	cmd.Flags().IntVar(&fileType, "type", service.FileTypeFile, "Upload file_type (default 2=model/file)")
	cmd.Flags().BoolVar(&checksum, "checksum", false, "Compute local MD5 checksum")
	cmd.Flags().StringVar(&modelID, "model-id", "", "Attach uploaded file to this model (bumps version; implies type=2 --checksum)")
	cmd.Flags().StringVar(&appID, "app-id", "", "Attach uploaded image as this app's cover (implies type=1, 5 MB app image limit)")
	cmd.Flags().BoolVar(&forApp, "for-app", false, "Type-1 image for app images[] (5 MB); default type-1 is 2 MB for model pic_url")
	cmd.Flags().BoolVar(&unlinked, "unlinked", false, "Allow orphan OSS object for later create/update JSON (types 1–2)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 attach when --model-id or --app-id is set")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample file_url")
	_ = cmd.MarkFlagRequired("path")
	return cmd
}
