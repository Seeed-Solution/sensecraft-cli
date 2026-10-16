package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/enum"
	"sensecraft-cloud-terminal/internal/uploadguard"
)

// File upload types aligned with Console upload.vue uploadType.
const (
	FileTypeImage    = 1 // app cover / pics
	FileTypeFile     = 2 // model binary / generic file
	FileTypeApp      = 3
	FileTypeFirmware = 4
	FileTypeDoc      = 5
	FileTypeArchive  = 6 // zip
)

// Console upload.vue / infoForm.vue / model/add.vue limits (frontend-enforced there; CLI mirrors).
const (
	MaxAppImageBytes       = uploadguard.MaxAppImageBytes
	MaxModelImageBytes     = uploadguard.MaxModelImageBytes
	MaxAppImageCount       = 16
	MaxNonImageUploadBytes = uploadguard.MaxNonImageUploadBytes
)

type FileService struct {
	client  *client.AIApiClient
	sandbox bool
}

func GetFileService(ctx app.AICmdContext) FileService {
	return FileService{client: newAIHTTPClient(ctx), sandbox: ctx.SandBox}
}

type FileUploadParams struct {
	Path     string
	FileType int
	// Checksum computes local MD5 (Console with-checksum).
	Checksum bool
	// ForAppImage selects Console app image cap (5MB). When false, type-1 uses
	// model example / default image cap (2MB). Ignored for non-image types.
	ForAppImage bool
}

type FileUploadResult struct {
	FileName string `json:"file_name"`
	FileType int    `json:"file_type"`
	FileURL  string `json:"file_url"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum,omitempty"`
}

// ErrAttachRequired is returned when an upload would create an orphan OSS
// object without an explicit association or --unlinked opt-in. Agents must
// ask the user for a model/app id (or confirm brand-new create) before upload.
var ErrAttachRequired = errors.New(
	"attach required before upload to avoid orphan OSS objects: pass --id <model_id> --yes (model upload), --model-id/--app-id --yes (file upload), or --unlinked only for brand-new create / pic_url / images[] JSON. Ask the user which applies; do not PUT first",
)

// UnlinkedUploadHint is returned when the user explicitly opted into --unlinked.
// Prefer attaching with --id / --model-id / --app-id whenever a record exists.
const UnlinkedUploadHint = "unlinked upload (--unlinked): put file_url into model create / update JSON (or app images[]) soon; prefer --id/--model-id/--app-id --yes when replacing an existing record"

// FileAttachResult is an upload plus optional model/app association.
type FileAttachResult struct {
	FileName        string `json:"file_name"`
	FileType        int    `json:"file_type"`
	FileURL         string `json:"file_url"`
	Size            int64  `json:"size"`
	Checksum        string `json:"checksum,omitempty"`
	Attached        bool   `json:"attached"`
	ModelID         string `json:"model_id,omitempty"`
	AppID           string `json:"app_id,omitempty"`
	Version         string `json:"version,omitempty"`
	PreviousVersion string `json:"previous_version,omitempty"`
	BePublic        int    `json:"be_public,omitempty"`
	ReReview        bool   `json:"re_review,omitempty"`
	Hint            string `json:"hint,omitempty"`
}

func FileAttachFromUpload(up FileUploadResult) FileAttachResult {
	return FileAttachResult{
		FileName: up.FileName,
		FileType: up.FileType,
		FileURL:  up.FileURL,
		Size:     up.Size,
		Checksum: up.Checksum,
	}
}

func (s FileService) UploadFile(p FileUploadParams) (FileUploadResult, error) {
	path := strings.TrimSpace(p.Path)
	if path == "" {
		return FileUploadResult{}, fmt.Errorf("--path is required")
	}
	fileType := p.FileType
	if fileType <= 0 {
		fileType = FileTypeFile
	}
	info, err := os.Stat(path)
	if err != nil {
		return FileUploadResult{}, fmt.Errorf("stat file: %w", err)
	}
	if info.IsDir() {
		return FileUploadResult{}, fmt.Errorf("path is a directory: %s", path)
	}
	fileName := strings.ReplaceAll(filepath.Base(path), " ", "")
	if fileName == "" {
		return FileUploadResult{}, fmt.Errorf("invalid file name")
	}
	if err := validateUploadFile(fileType, path, info.Size(), p.ForAppImage); err != nil {
		return FileUploadResult{}, err
	}

	var checksum string
	if p.Checksum {
		sum, _, err := md5File(path)
		if err != nil {
			return FileUploadResult{}, err
		}
		checksum = sum
	}

	if s.sandbox {
		res := sandboxFileUpload(fileName, fileType, info.Size(), p.Checksum)
		if checksum != "" {
			res.Checksum = checksum
		}
		return res, nil
	}

	q := url.Values{}
	q.Set("file_name", fileName)
	q.Set("file_type", strconv.Itoa(fileType))
	raw, err := s.client.Get(enum.AIApiPathAcquireFile, q)
	if err != nil {
		return FileUploadResult{}, err
	}

	uploadURL, fileURL, err := parseAcquireFileURL(raw)
	if err != nil {
		return FileUploadResult{}, err
	}

	f, err := os.Open(path)
	if err != nil {
		return FileUploadResult{}, err
	}
	defer f.Close()

	if err := putFile(uploadURL, f, info.Size()); err != nil {
		return FileUploadResult{}, err
	}

	return FileUploadResult{
		FileName: fileName,
		FileType: fileType,
		FileURL:  fileURL,
		Size:     info.Size(),
		Checksum: checksum,
	}, nil
}

func parseAcquireFileURL(raw json.RawMessage) (uploadURL, fileURL string, err error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", "", fmt.Errorf("parse acquire_file_url: %w", err)
	}
	uploadURL, _ = stringField(m, "upload_url")
	fileURL, _ = stringField(m, "file_url")
	if uploadURL == "" {
		return "", "", fmt.Errorf("acquire_file_url missing upload_url")
	}
	if fileURL == "" {
		return "", "", fmt.Errorf("acquire_file_url missing file_url")
	}
	return uploadURL, fileURL, nil
}

func putFile(uploadURL string, body io.Reader, size int64) error {
	req, err := http.NewRequest(http.MethodPut, uploadURL, body)
	if err != nil {
		return fmt.Errorf("build PUT: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if size > 0 {
		req.ContentLength = size
	}
	httpClient := &http.Client{Timeout: 30 * time.Minute}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("PUT upload: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("PUT upload HTTP %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func md5File(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := md5.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, fmt.Errorf("checksum: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func sandboxFileUpload(fileName string, fileType int, size int64, withChecksum bool) FileUploadResult {
	res := FileUploadResult{
		FileName: fileName,
		FileType: fileType,
		FileURL:  "https://example.invalid/sandbox/" + fileName,
		Size:     size,
	}
	if withChecksum {
		res.Checksum = "sandboxmd5checksum000000000000000"
	}
	return res
}

func maxUploadBytes(fileType int, forAppImage bool) int64 {
	if fileType == FileTypeImage {
		if forAppImage {
			return MaxAppImageBytes
		}
		return MaxModelImageBytes
	}
	return MaxNonImageUploadBytes
}

// validateUploadFile mirrors Console upload.vue size (+ image extension) checks.
func validateUploadFile(fileType int, path string, size int64, forAppImage bool) error {
	if fileType == FileTypeImage {
		if err := validateImageExtension(path); err != nil {
			return err
		}
	}
	if err := uploadguard.ValidateOSSUploadSize(fileType, size, forAppImage); err != nil {
		if errors.Is(err, uploadguard.ErrEmptyUpload) {
			return err
		}
		if fileType == FileTypeImage && forAppImage && size > MaxAppImageBytes {
			return fmt.Errorf(
				"%w (infoForm max-size=5; max %d images)",
				err, MaxAppImageCount,
			)
		}
		return err
	}
	return nil
}

func validateImageExtension(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return nil
	case ".svg":
		return fmt.Errorf("SVG images are not allowed (Console upload.vue rejects image/svg+xml)")
	default:
		return fmt.Errorf(
			"unsupported image extension %q (Console accepts JPG/JPEG/PNG/GIF/WebP)",
			ext,
		)
	}
}

// validateAppImageCount mirrors Console infoForm maxImageCount=16.
func validateAppImageCount(images []string) error {
	if len(images) > MaxAppImageCount {
		return fmt.Errorf(
			"app images exceed Console maxImageCount=%d (got %d); remove extras before update/attach",
			MaxAppImageCount, len(images),
		)
	}
	return nil
}
