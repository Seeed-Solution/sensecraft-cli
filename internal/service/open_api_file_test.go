package service

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sensecraft-cloud-terminal/internal/client"
)

func TestParseAcquireFileURL(t *testing.T) {
	got, err := parseAcquireFileURL([]byte(`{
		"host":"https://oss.example",
		"policy":"cG9s",
		"signature":"sig",
		"OSSAccessKeyId":"STS.id",
		"security_token":"tok",
		"key":"refer/pic/a.png",
		"file_url":"https://cdn.example/a.png",
		"max_bytes":"5242880",
		"content_type":"image/png"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != "https://oss.example" || got.FileURL != "https://cdn.example/a.png" || got.MaxBytes != 5242880 || got.ContentType != "image/png" {
		t.Fatalf("%+v", got)
	}
	if got.Key != "refer/pic/a.png" || got.Policy != "cG9s" || got.Signature != "sig" || got.OSSAccessKeyID != "STS.id" || got.SecurityToken != "tok" {
		t.Fatalf("%+v", got)
	}

	num, err := parseAcquireFileURL([]byte(`{
		"host":"https://oss.example",
		"policy":"cG9s",
		"signature":"sig",
		"OSSAccessKeyId":"STS.id",
		"security_token":"tok",
		"key":"refer/model/m.bin",
		"file_url":"https://cdn.example/m.bin",
		"max_bytes":104857600
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if num.MaxBytes != 104857600 || num.ContentType != "" {
		t.Fatalf("%+v", num)
	}

	if _, err := parseAcquireFileURL([]byte(`{"upload_url":"https://up.example/x","file_url":"https://cdn.example/y.bin"}`)); err == nil {
		t.Fatal("expected legacy upload_url payload to fail")
	}
	if _, err := parseAcquireFileURL([]byte(`{
		"host":"http://oss.example",
		"policy":"cG9s",
		"signature":"sig",
		"OSSAccessKeyId":"STS.id",
		"security_token":"tok",
		"key":"refer/model/m.bin",
		"file_url":"https://cdn.example/m.bin",
		"max_bytes":"10"
	}`)); err == nil {
		t.Fatal("expected http host to fail")
	}
	if _, err := parseAcquireFileURL([]byte(`{
		"host":"ftp://files.example",
		"policy":"cG9s",
		"signature":"sig",
		"OSSAccessKeyId":"STS.id",
		"security_token":"tok",
		"key":"refer/model/m.bin",
		"file_url":"https://cdn.example/m.bin",
		"max_bytes":"10"
	}`)); err == nil {
		t.Fatal("expected non-http host to fail")
	}
}

func TestSandboxFileUpload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.bin")
	if err := os.WriteFile(path, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := FileService{sandbox: true}
	res, err := s.UploadFile(FileUploadParams{Path: path, FileType: 2, Checksum: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.FileURL == "" || res.Size != 3 || res.Checksum == "" {
		t.Fatalf("got %+v", res)
	}
}

func TestValidateUploadFileImageSize(t *testing.T) {
	dir := t.TempDir()
	okPath := filepath.Join(dir, "ok.png")
	if err := os.WriteFile(okPath, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateUploadFile(FileTypeImage, okPath, MaxAppImageBytes); err != nil {
		t.Fatal(err)
	}
	if err := validateUploadFile(FileTypeImage, okPath, 3*1024*1024); err != nil {
		t.Fatal(err)
	}
	err := validateUploadFile(FileTypeImage, okPath, MaxAppImageBytes+1)
	if err == nil || !strings.Contains(err.Error(), "5 MB") {
		t.Fatalf("want 5 MB message, got %v", err)
	}
}

func TestValidateUploadFileNonImageSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.bin")
	if err := validateUploadFile(FileTypeFile, path, MaxNonImageUploadBytes); err != nil {
		t.Fatal(err)
	}
	if err := validateUploadFile(FileTypeFile, path, MaxNonImageUploadBytes+1); err == nil {
		t.Fatal("expected size rejection")
	}
	if err := validateUploadFile(FileTypeFile, path, 0); err == nil {
		t.Fatal("expected empty file rejection")
	}
}

func TestValidateImageExtension(t *testing.T) {
	if err := validateImageExtension("a.PNG"); err != nil {
		t.Fatal(err)
	}
	if err := validateImageExtension("a.webp"); err != nil {
		t.Fatal(err)
	}
	if err := validateImageExtension("a.svg"); err == nil {
		t.Fatal("expected svg rejection")
	}
	if err := validateImageExtension("a.bin"); err == nil {
		t.Fatal("expected extension rejection")
	}
}

func TestUploadFileAcceptsMidSizeImageInSandbox(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mid.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(3 * 1024 * 1024); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	s := FileService{sandbox: true}
	res, err := s.UploadFile(FileUploadParams{Path: path, FileType: FileTypeImage})
	if err != nil {
		t.Fatal(err)
	}
	if res.Size != 3*1024*1024 {
		t.Fatalf("size %d", res.Size)
	}
}

func TestUploadFileRejectsOversizedImageInSandbox(t *testing.T) {
	path := filepath.Join(t.TempDir(), "huge.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxAppImageBytes + 1); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	s := FileService{sandbox: true}
	_, err = s.UploadFile(FileUploadParams{Path: path, FileType: FileTypeImage})
	if err == nil || !strings.Contains(err.Error(), "5 MB") {
		t.Fatalf("expected oversized image rejection, got %v", err)
	}
}

func TestUploadFileType5RequiresFlowID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "guide.pdf")
	if err := os.WriteFile(path, []byte("pdf"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := FileService{sandbox: true}
	_, err := s.UploadFile(FileUploadParams{Path: path, FileType: FileTypeDoc})
	if err == nil || !strings.Contains(err.Error(), "flow-id") {
		t.Fatalf("got %v", err)
	}
	res, err := s.UploadFile(FileUploadParams{Path: path, FileType: FileTypeDoc, FlowID: " flow-1 "})
	if err != nil {
		t.Fatal(err)
	}
	if res.FileURL == "" || res.FileType != FileTypeDoc {
		t.Fatalf("%+v", res)
	}
}

func TestValidateAppImageCount(t *testing.T) {
	imgs := make([]string, MaxAppImageCount)
	for i := range imgs {
		imgs[i] = "https://example.invalid/" + string(rune('a'+i%26)) + ".png"
	}
	if err := validateAppImageCount(imgs); err != nil {
		t.Fatal(err)
	}
	imgs = append(imgs, "https://example.invalid/extra.png")
	if err := validateAppImageCount(imgs); err == nil {
		t.Fatal("expected count rejection")
	}
}

type ossCapture struct {
	query    map[string]string
	parts    []string
	fields   map[string]string
	file     []byte
	fileName string
	fileCT   string
	header   http.Header
	posts    int
}

func newUploadServer(t *testing.T, body string, ossStatus int, ossBody string) (*httptest.Server, *ossCapture) {
	t.Helper()
	cap := &ossCapture{fields: map[string]string{}}
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			cap.query = map[string]string{}
			for k, vs := range r.URL.Query() {
				if len(vs) > 0 {
					cap.query[k] = vs[0]
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, strings.ReplaceAll(body, "{{HOST}}", srv.URL))
			return
		}
		cap.posts++
		cap.header = r.Header.Clone()
		media, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || media != "multipart/form-data" {
			http.Error(w, "bad content-type", http.StatusBadRequest)
			return
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			cap.parts = append(cap.parts, p.FormName())
			b, _ := io.ReadAll(p)
			if p.FormName() == "file" {
				cap.file = b
				cap.fileName = p.FileName()
				cap.fileCT = p.Header.Get("Content-Type")
				continue
			}
			cap.fields[p.FormName()] = string(b)
		}
		w.WriteHeader(ossStatus)
		_, _ = io.WriteString(w, ossBody)
	}))
	t.Cleanup(srv.Close)
	prev := externalHTTPClient
	externalHTTPClient = func(timeout time.Duration) *http.Client {
		c := srv.Client()
		c.Timeout = timeout
		c.CheckRedirect = refuseCrossHostRedirect
		return c
	}
	t.Cleanup(func() { externalHTTPClient = prev })
	return srv, cap
}

func uploadService(srv *httptest.Server) FileService {
	return FileService{client: client.NewAIApiClientAt(srv.URL, srv.Client())}
}

func TestUploadFilePostsImageForm(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.png")
	payload := []byte("png-bytes")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	body := `{
		"code":"0",
		"data":{
			"host":"{{HOST}}",
			"policy":"cG9s",
			"signature":"sig",
			"OSSAccessKeyId":"STS.id",
			"security_token":"tok",
			"key":"refer/pic/photo.png",
			"file_url":"https://cdn.example/photo.png",
			"max_bytes":"5242880",
			"content_type":"image/png"
		}
	}`
	srv, cap := newUploadServer(t, body, http.StatusNoContent, "")
	res, err := uploadService(srv).UploadFile(FileUploadParams{Path: path, FileType: FileTypeImage, FlowID: "ignored"})
	if err != nil {
		t.Fatal(err)
	}
	if res.FileURL != "https://cdn.example/photo.png" || res.Size != int64(len(payload)) {
		t.Fatalf("%+v", res)
	}
	if cap.query["file_type"] != "1" || cap.query["file_name"] != "photo.png" || cap.query["flow_id"] != "" {
		t.Fatalf("query %#v", cap.query)
	}
	want := "key,policy,OSSAccessKeyId,Signature,x-oss-security-token,Content-Type,file"
	if strings.Join(cap.parts, ",") != want {
		t.Fatalf("parts %v", cap.parts)
	}
	if cap.fields["Content-Type"] != "image/png" || cap.fields["key"] != "refer/pic/photo.png" || cap.fields["Signature"] != "sig" || cap.fields["x-oss-security-token"] != "tok" || cap.fields["OSSAccessKeyId"] != "STS.id" || cap.fields["policy"] != "cG9s" {
		t.Fatalf("fields %#v", cap.fields)
	}
	if string(cap.file) != string(payload) || cap.fileCT != "image/png" {
		t.Fatalf("file %q ct %q", cap.file, cap.fileCT)
	}
	if cap.header.Get("Authorization") != "" || cap.header.Get("sensecraft-cli-accessid") != "" || cap.header.Get("sensecraft-cli-version") != "" {
		t.Fatalf("OSS request carried CLI headers: %v", cap.header)
	}
}

func TestUploadFileSendsRequestedNameAndKeepsServerURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "local.bin")
	if err := os.WriteFile(path, []byte("bin"), 0o600); err != nil {
		t.Fatal(err)
	}
	body := `{
		"code":"0",
		"data":{
			"host":"{{HOST}}",
			"policy":"cG9s",
			"signature":"sig",
			"OSSAccessKeyId":"STS.id",
			"security_token":"tok",
			"key":"oss/rewritten.bin",
			"file_url":"https://cdn.example/oss/rewritten.bin",
			"max_bytes":104857600
		}
	}`
	srv, cap := newUploadServer(t, body, http.StatusNoContent, "")
	res, err := uploadService(srv).UploadFile(FileUploadParams{
		Path: path, FileType: FileTypeFile, Name: " project muse.bin ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cap.query["file_name"] != "projectmuse.bin" || cap.fileName != "projectmuse.bin" {
		t.Fatalf("query %#v multipart %q", cap.query, cap.fileName)
	}
	if res.FileName != "projectmuse.bin" || res.FileURL != "https://cdn.example/oss/rewritten.bin" {
		t.Fatalf("%+v", res)
	}
	if cap.fields["key"] != "oss/rewritten.bin" {
		t.Fatalf("key %#v", cap.fields)
	}
}

func TestUploadFileRejectsPathName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "local.bin")
	if err := os.WriteFile(path, []byte("bin"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := (FileService{}).UploadFile(FileUploadParams{Path: path, FileType: FileTypeFile, Name: `../secret.bin`})
	if err == nil || !strings.Contains(err.Error(), "invalid file name") {
		t.Fatalf("got %v", err)
	}
}

func TestUploadFilePostsModelWithoutContentType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "model.bin")
	if err := os.WriteFile(path, []byte("bin"), 0o600); err != nil {
		t.Fatal(err)
	}
	body := `{
		"code":"0",
		"data":{
			"host":"{{HOST}}",
			"policy":"cG9s",
			"signature":"sig",
			"OSSAccessKeyId":"STS.id",
			"security_token":"tok",
			"key":"refer/model/model.bin",
			"file_url":"https://cdn.example/model.bin",
			"max_bytes":104857600
		}
	}`
	srv, cap := newUploadServer(t, body, http.StatusNoContent, "")
	if _, err := uploadService(srv).UploadFile(FileUploadParams{Path: path, FileType: FileTypeFile}); err != nil {
		t.Fatal(err)
	}
	want := "key,policy,OSSAccessKeyId,Signature,x-oss-security-token,file"
	if strings.Join(cap.parts, ",") != want {
		t.Fatalf("parts %v", cap.parts)
	}
	if cap.fileCT != "application/octet-stream" {
		t.Fatalf("file content-type %q", cap.fileCT)
	}
	if _, ok := cap.fields["Content-Type"]; ok {
		t.Fatalf("non-image form must omit Content-Type, got %#v", cap.fields)
	}
}

func TestUploadFileType5SendsFlowID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "guide.pdf")
	if err := os.WriteFile(path, []byte("pdf"), 0o600); err != nil {
		t.Fatal(err)
	}
	body := `{
		"code":"0",
		"data":{
			"host":"{{HOST}}",
			"policy":"cG9s",
			"signature":"sig",
			"OSSAccessKeyId":"STS.id",
			"security_token":"tok",
			"key":"refer/static/guide.pdf",
			"file_url":"https://cdn.example/guide.pdf",
			"max_bytes":"100"
		}
	}`
	srv, cap := newUploadServer(t, body, http.StatusNoContent, "")
	if _, err := uploadService(srv).UploadFile(FileUploadParams{Path: path, FileType: FileTypeDoc, FlowID: "flow-9"}); err != nil {
		t.Fatal(err)
	}
	if cap.query["file_type"] != "5" || cap.query["flow_id"] != "flow-9" {
		t.Fatalf("query %#v", cap.query)
	}
}

func TestUploadFileHonorsReturnedMaxBytesBeforePost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.png")
	if err := os.WriteFile(path, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	body := `{
		"code":"0",
		"data":{
			"host":"{{HOST}}",
			"policy":"cG9s",
			"signature":"sig",
			"OSSAccessKeyId":"STS.id",
			"security_token":"tok",
			"key":"refer/pic/photo.png",
			"file_url":"https://cdn.example/photo.png",
			"max_bytes":"1",
			"content_type":"image/png"
		}
	}`
	srv, cap := newUploadServer(t, body, http.StatusNoContent, "")
	_, err := uploadService(srv).UploadFile(FileUploadParams{Path: path, FileType: FileTypeImage})
	if err == nil || !strings.Contains(err.Error(), "max_bytes") {
		t.Fatalf("got %v", err)
	}
	if cap.posts != 0 {
		t.Fatal("must not POST when the file is over max_bytes")
	}
}

func TestUploadFileRequiresImageContentType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.png")
	if err := os.WriteFile(path, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	body := `{
		"code":"0",
		"data":{
			"host":"{{HOST}}",
			"policy":"cG9s",
			"signature":"sig",
			"OSSAccessKeyId":"STS.id",
			"security_token":"tok",
			"key":"refer/pic/photo.png",
			"file_url":"https://cdn.example/photo.png",
			"max_bytes":"10"
		}
	}`
	srv, cap := newUploadServer(t, body, http.StatusNoContent, "")
	_, err := uploadService(srv).UploadFile(FileUploadParams{Path: path, FileType: FileTypeImage})
	if err == nil || !strings.Contains(err.Error(), "content_type") {
		t.Fatalf("got %v", err)
	}
	if cap.posts != 0 {
		t.Fatal("must not POST without content_type")
	}
}

func TestUploadFileOSSStatusAndBusinessErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.png")
	if err := os.WriteFile(path, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	okData := `{
		"code":"0",
		"data":{
			"host":"{{HOST}}",
			"policy":"cG9s",
			"signature":"sig",
			"OSSAccessKeyId":"STS.id",
			"security_token":"tok",
			"key":"refer/pic/photo.png",
			"file_url":"https://cdn.example/photo.png",
			"max_bytes":"10",
			"content_type":"image/png"
		}
	}`
	srv, _ := newUploadServer(t, okData, http.StatusForbidden, `<Error>Your proposed upload exceeds the maximum allowed size</Error>`)
	_, err := uploadService(srv).UploadFile(FileUploadParams{Path: path, FileType: FileTypeImage})
	if err == nil || !strings.Contains(err.Error(), "Your proposed upload exceeds the maximum allowed size") {
		t.Fatalf("got %v", err)
	}

	okSrv, _ := newUploadServer(t, okData, http.StatusOK, "ok")
	_, err = uploadService(okSrv).UploadFile(FileUploadParams{Path: path, FileType: FileTypeImage})
	if err == nil || !strings.Contains(err.Error(), "204") {
		t.Fatalf("non-204 must fail, got %v", err)
	}

	limitSrv, cap := newUploadServer(t, `{"code":"11209","msg":"Uploads are too frequent, please try again later"}`, http.StatusNoContent, "")
	_, err = uploadService(limitSrv).UploadFile(FileUploadParams{Path: path, FileType: FileTypeImage})
	if err == nil || !strings.Contains(err.Error(), "11209") || !strings.Contains(err.Error(), "Uploads are too frequent") {
		t.Fatalf("got %v", err)
	}
	if strings.Contains(err.Error(), "10-minute") || strings.Contains(err.Error(), "Beijing") {
		t.Fatalf("rate-limit error must stay the server message, got %v", err)
	}
	if cap.posts != 0 {
		t.Fatal("business error must not POST")
	}

	daySrv, _ := newUploadServer(t, `{"code":"11210","msg":"Daily upload limit reached, please try again tomorrow"}`, http.StatusNoContent, "")
	_, err = uploadService(daySrv).UploadFile(FileUploadParams{Path: path, FileType: FileTypeImage})
	if err == nil || !strings.Contains(err.Error(), "11210") || !strings.Contains(err.Error(), "Daily upload limit reached") {
		t.Fatalf("got %v", err)
	}

	fmtSrv, _ := newUploadServer(t, `{"code":"11208","msg":"File format is incorrect"}`, http.StatusNoContent, "")
	_, err = uploadService(fmtSrv).UploadFile(FileUploadParams{Path: path, FileType: FileTypeImage})
	if err == nil || !strings.Contains(err.Error(), "11208") || !strings.Contains(err.Error(), "webp") {
		t.Fatalf("got %v", err)
	}
}
