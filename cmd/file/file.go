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
		Short: "SenseCraft file upload (acquire_file_url + OSS POST)",
	}
	cmd.AddCommand(newFileUploadCmd())
	return cmd
}

func newFileUploadCmd() *cobra.Command {
	var (
		path     string
		fileType int
		flowID   string
		deviceID string
		checksum bool
		name     string
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
		Long: `GET /user/acquire_file_url, then POST multipart/form-data to the returned host.

file_type:
  1=image  2=file/model (default)  3=app  4=firmware  5=static resource  6=archive/zip

Size limits:
  type 1:  ≤5 MB (jpg/jpeg/png/gif/webp; no SVG)
  type 2–6: ≤100 MB
  empty (0 B) files are rejected
  type 5 requires --flow-id

Association (types 1–2) is required before any OSS upload:
  --model-id <id> --yes   type=2 + checksum; writes file onto the model and bumps version
  --app-id <id> --yes     type=1 cover image; prepends images[] (max 16 total; 5 MB)
  --for-app               type=1 unlinked for later app images[] JSON
  --unlinked              explicit orphan for later create/update JSON (pic_url / model create)

Types 3–6 have no model/app attach path and upload directly.
Without association flags: interactive TTY guides; agents must ask first.

--name is the file_name sent with the upload. The server may rewrite part of
that name. Use the returned file_url; do not build a URL from --name.

Type 4 firmware guidance:
  - Pass a single .bin/.hex, or a matched device folder (e.g. reterminal-e1001, sensecap-watcher, "E1001设备").
  - For a bare file, pass --device-id <SKU> so the response includes muse hints (address/Muse/full_flash/build_artifact).
  - Muse SDK folders upload muse-gadget.bin only; companion segments stay local (app site supplies them).
  - Success JSON may include muse{device_id,flash_address,muse_gadget,full_flash,build_artifact,note}.
  - Muse ON → muse_gadget true, full_flash false, keep device default flash_address (not 0x20000).
  - Muse-capable board + normal firmware → muse_gadget false, default full_flash true.
  - E100x @ 0x90000 only; Wio @ 0x4000 only; Watcher/Sticky/Indicator/1.47 merged @ 0x0 only.
  - Generate firmware by address: 0x0 → merged image; 0x90000/0x4000/0x10000 → app-region; Muse → muse-gadget.bin.
  - muse.note always states when 0x0 needs a merged .bin; uploads that are far too small for a 0x0 merge are rejected before OSS.
  - Rejection messages name the conflicting field — fix that field before retrying.
  - See sensecraft-cli app blocks → flash-firmware.device_defaults (build_artifact).`,
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
					Name:    name,
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
					Path:     path,
					FileType: service.FileTypeImage,
					Checksum: checksum,
					Name:     name,
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
				Path:     path,
				FileType: fileType,
				Checksum: checksum,
				FlowID:   flowID,
				Name:     name,
				DeviceID: deviceID,
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
	cmd.Flags().StringVar(&name, "name", "", "file_name sent to the server; it may rewrite the stored name, so use the returned file_url")
	cmd.Flags().IntVar(&fileType, "type", service.FileTypeFile, "Upload file_type (default 2=model/file)")
	cmd.Flags().StringVar(&flowID, "flow-id", "", "Required when --type 5 (static resource)")
	cmd.Flags().StringVar(&deviceID, "device-id", "", "Type 4: material SKU for muse/address/build_artifact hints on a bare firmware file")
	cmd.Flags().BoolVar(&checksum, "checksum", false, "Compute local MD5 checksum")
	cmd.Flags().StringVar(&modelID, "model-id", "", "Attach uploaded file to this model (bumps version; implies type=2 --checksum)")
	cmd.Flags().StringVar(&appID, "app-id", "", "Attach uploaded image as this app's cover (implies type=1, 5 MB)")
	cmd.Flags().BoolVar(&forApp, "for-app", false, "Type-1 unlinked image for later app images[] JSON (5 MB)")
	cmd.Flags().BoolVar(&unlinked, "unlinked", false, "Allow orphan OSS object for later create/update JSON (types 1–2)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 attach when --model-id or --app-id is set")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample file_url")
	_ = cmd.MarkFlagRequired("path")
	return cmd
}
