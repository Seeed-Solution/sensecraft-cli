package service

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/textproto"
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
	FileTypeImage    = 1 // image
	FileTypeFile     = 2 // model binary / generic file
	FileTypeApp      = 3
	FileTypeFirmware = 4
	FileTypeDoc      = 5 // static resource; requires flow_id
	FileTypeArchive  = 6 // zip
)

// OpenAPI upload caps. Every image is 5MB; types 2–6 are 100MB.
const (
	MaxAppImageBytes       = uploadguard.MaxAppImageBytes
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
	// FlowID is required for file_type 5 (static resource).
	FlowID string
	// DeviceID is optional for type-4 single-file uploads: fills muse hints
	// (flash_address / muse_gadget / full_flash / build_artifact) when the
	// parent folder is not a matched device directory.
	DeviceID string
}

type FileUploadResult struct {
	FileName string          `json:"file_name"`
	FileType int             `json:"file_type"`
	FileURL  string          `json:"file_url"`
	Size     int64           `json:"size"`
	Checksum string          `json:"checksum,omitempty"`
	Muse     *MuseUploadInfo `json:"muse,omitempty"`
}

// ErrAttachRequired is returned when an upload would create an orphan OSS
// object without an explicit association or --unlinked opt-in. Agents must
// ask the user for a model/app id (or confirm brand-new create) before upload.
var ErrAttachRequired = errors.New(
	"attach required before upload to avoid orphan OSS objects: pass --id <model_id> --yes (model upload), --model-id/--app-id --yes (file upload), or --unlinked only for brand-new create / pic_url / images[] JSON. Ask the user which applies; do not upload first",
)

// UnlinkedUploadHint is returned when the user explicitly opted into --unlinked.
// Prefer attaching with --id / --model-id / --app-id whenever a record exists.
const UnlinkedUploadHint = "unlinked upload (--unlinked): put file_url into model create / update JSON (or app images[]) soon; prefer --id/--model-id/--app-id --yes when replacing an existing record"

// FileAttachResult is an upload plus optional model/app association.
type FileAttachResult struct {
	FileName        string          `json:"file_name"`
	FileType        int             `json:"file_type"`
	FileURL         string          `json:"file_url"`
	Size            int64           `json:"size"`
	Checksum        string          `json:"checksum,omitempty"`
	Attached        bool            `json:"attached"`
	ModelID         string          `json:"model_id,omitempty"`
	AppID           string          `json:"app_id,omitempty"`
	Version         string          `json:"version,omitempty"`
	PreviousVersion string          `json:"previous_version,omitempty"`
	BePublic        int             `json:"be_public,omitempty"`
	ReReview        bool            `json:"re_review,omitempty"`
	Hint            string          `json:"hint,omitempty"`
	Muse            *MuseUploadInfo `json:"muse,omitempty"`
}

func FileAttachFromUpload(up FileUploadResult) FileAttachResult {
	return FileAttachResult{
		FileName: up.FileName,
		FileType: up.FileType,
		FileURL:  up.FileURL,
		Size:     up.Size,
		Checksum: up.Checksum,
		Muse:     up.Muse,
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
	flowID := strings.TrimSpace(p.FlowID)
	if fileType == FileTypeDoc && flowID == "" {
		return FileUploadResult{}, fmt.Errorf("--flow-id is required when --type 5 (static resource)")
	}

	var muse *MuseUploadInfo
	deviceID := strings.TrimSpace(p.DeviceID)
	if fileType == FileTypeFirmware {
		prepared, err := prepareMuseFirmwareUpload(path)
		if err != nil {
			return FileUploadResult{}, err
		}
		if prepared != nil {
			path = prepared.uploadPath
			info := prepared.info
			if deviceID != "" && !deviceIDInList(deviceID, info.DeviceIDs) && deviceID != info.DeviceID {
				return FileUploadResult{}, fmt.Errorf(
					"--device-id %s conflicts with folder-matched device_id %s (%s). Omit --device-id to keep the folder match, or upload a bare file with the intended --device-id",
					deviceID, info.DeviceID, info.DeviceName,
				)
			}
			muse = &info
		} else {
			hint, err := singleFileFirmwareHint(path, deviceID)
			if err != nil {
				return FileUploadResult{}, err
			}
			muse = hint
		}
	} else if deviceID != "" {
		return FileUploadResult{}, fmt.Errorf("--device-id is only valid with --type 4 (firmware)")
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
	if err := validateUploadFile(fileType, path, info.Size()); err != nil {
		return FileUploadResult{}, err
	}
	if fileType == FileTypeFirmware {
		enrichFirmwareUploadGuidance(muse)
		if err := rejectTinyMergedFirmwareUpload(path, info.Size(), muse); err != nil {
			return FileUploadResult{}, err
		}
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
		res.Muse = muse
		return res, nil
	}

	q := url.Values{}
	q.Set("file_name", fileName)
	q.Set("file_type", strconv.Itoa(fileType))
	if fileType == FileTypeDoc {
		q.Set("flow_id", flowID)
	}
	raw, err := s.client.Get(enum.AIApiPathAcquireFile, q)
	if err != nil {
		return FileUploadResult{}, annotateAcquireError(err)
	}

	cred, err := parseAcquireFileURL(raw)
	if err != nil {
		return FileUploadResult{}, err
	}
	if fileType == FileTypeImage && cred.ContentType == "" {
		return FileUploadResult{}, fmt.Errorf("acquire_file_url missing content_type")
	}
	if info.Size() > cred.MaxBytes {
		return FileUploadResult{}, fmt.Errorf(
			"file exceeds acquire_file_url max_bytes (%d bytes); got %d bytes",
			cred.MaxBytes, info.Size(),
		)
	}

	f, err := os.Open(path)
	if err != nil {
		return FileUploadResult{}, err
	}
	defer f.Close()

	if err := postOSSFile(cred, fileName, f, info.Size()); err != nil {
		return FileUploadResult{}, err
	}

	return FileUploadResult{
		FileName: fileName,
		FileType: fileType,
		FileURL:  cred.FileURL,
		Size:     info.Size(),
		Checksum: checksum,
		Muse:     muse,
	}, nil
}

// acquiredFile is the OSS POST policy returned by acquire_file_url.
type acquiredFile struct {
	Host           string
	Policy         string
	Signature      string
	OSSAccessKeyID string
	SecurityToken  string
	Key            string
	FileURL        string
	MaxBytes       int64
	ContentType    string
}

func parseAcquireFileURL(raw json.RawMessage) (acquiredFile, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return acquiredFile{}, fmt.Errorf("parse acquire_file_url: %w", err)
	}
	host, err := requireAcquireString(m, "host")
	if err != nil {
		return acquiredFile{}, err
	}
	u, err := url.Parse(host)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return acquiredFile{}, fmt.Errorf("acquire_file_url host is not an http(s) URL")
	}
	policy, err := requireAcquireString(m, "policy")
	if err != nil {
		return acquiredFile{}, err
	}
	signature, err := requireAcquireString(m, "signature")
	if err != nil {
		return acquiredFile{}, err
	}
	accessKeyID, err := requireAcquireString(m, "OSSAccessKeyId")
	if err != nil {
		return acquiredFile{}, err
	}
	token, err := requireAcquireString(m, "security_token")
	if err != nil {
		return acquiredFile{}, err
	}
	key, err := requireAcquireString(m, "key")
	if err != nil {
		return acquiredFile{}, err
	}
	fileURL, err := requireAcquireString(m, "file_url")
	if err != nil {
		return acquiredFile{}, err
	}
	maxBytes, err := acquireMaxBytes(m["max_bytes"])
	if err != nil {
		return acquiredFile{}, err
	}
	contentType, _ := stringField(m, "content_type")
	return acquiredFile{
		Host:           host,
		Policy:         policy,
		Signature:      signature,
		OSSAccessKeyID: accessKeyID,
		SecurityToken:  token,
		Key:            key,
		FileURL:        fileURL,
		MaxBytes:       maxBytes,
		ContentType:    contentType,
	}, nil
}

func requireAcquireString(m map[string]any, key string) (string, error) {
	s, ok := stringField(m, key)
	if !ok {
		return "", fmt.Errorf("acquire_file_url missing %s", key)
	}
	return s, nil
}

func acquireMaxBytes(v any) (int64, error) {
	switch n := v.(type) {
	case float64:
		if n <= 0 || n != math.Trunc(n) || n > float64(math.MaxInt64) {
			return 0, fmt.Errorf("acquire_file_url max_bytes is invalid")
		}
		return int64(n), nil
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		if err != nil || i <= 0 {
			return 0, fmt.Errorf("acquire_file_url max_bytes is invalid")
		}
		return i, nil
	default:
		return 0, fmt.Errorf("acquire_file_url missing max_bytes")
	}
}

func annotateAcquireError(err error) error {
	var api client.AIAPIError
	if !errors.As(err, &api) {
		return err
	}
	if api.Code == 11208 {
		return fmt.Errorf("%w; allowed image suffixes are jpg, jpeg, png, gif, webp", err)
	}
	return err
}

// postOSSFile POSTs multipart/form-data to the OSS host. The file part is last.
// Images also send Content-Type, and the file part uses that same MIME type.
// Success is HTTP 204. This request does not carry Access Key or CLI headers.
func postOSSFile(cred acquiredFile, fileName string, body io.Reader, size int64) error {
	var prefix bytes.Buffer
	w := multipart.NewWriter(&prefix)
	fields := [][2]string{
		{"key", cred.Key},
		{"policy", cred.Policy},
		{"OSSAccessKeyId", cred.OSSAccessKeyID},
		{"Signature", cred.Signature},
		{"x-oss-security-token", cred.SecurityToken},
	}
	if cred.ContentType != "" {
		fields = append(fields, [2]string{"Content-Type", cred.ContentType})
	}
	for _, field := range fields {
		if err := w.WriteField(field[0], field[1]); err != nil {
			return fmt.Errorf("build OSS form: %w", err)
		}
	}
	fileCT := "application/octet-stream"
	if cred.ContentType != "" {
		fileCT = cred.ContentType
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, escapeMultipart(fileName)))
	header.Set("Content-Type", fileCT)
	if _, err := w.CreatePart(header); err != nil {
		return fmt.Errorf("build OSS form: %w", err)
	}
	suffix := "\r\n--" + w.Boundary() + "--\r\n"
	req, err := http.NewRequest(http.MethodPost, cred.Host, io.MultiReader(bytes.NewReader(prefix.Bytes()), body, strings.NewReader(suffix)))
	if err != nil {
		return fmt.Errorf("build OSS POST: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.ContentLength = int64(prefix.Len()) + size + int64(len(suffix))

	httpClient := &http.Client{Timeout: 30 * time.Minute}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("OSS upload: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("OSS upload HTTP %d, want 204: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func escapeMultipart(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
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

// validateUploadFile checks image suffixes and the OpenAPI size caps.
func validateUploadFile(fileType int, path string, size int64) error {
	if fileType == FileTypeImage {
		if err := validateImageExtension(path); err != nil {
			return err
		}
	}
	return uploadguard.ValidateOSSUploadSize(fileType, size)
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
