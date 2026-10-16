package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAcquireFileURL(t *testing.T) {
	u, f, err := parseAcquireFileURL([]byte(`{"upload_url":"https://up.example/x","file_url":"https://cdn.example/y.bin"}`))
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://up.example/x" || f != "https://cdn.example/y.bin" {
		t.Fatalf("got %q %q", u, f)
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

func TestValidateUploadFileImageSizeAppVsModel(t *testing.T) {
	dir := t.TempDir()
	okPath := filepath.Join(dir, "ok.png")
	if err := os.WriteFile(okPath, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Model / default type-1: 2MB
	if err := validateUploadFile(FileTypeImage, okPath, MaxModelImageBytes, false); err != nil {
		t.Fatal(err)
	}
	if err := validateUploadFile(FileTypeImage, okPath, MaxModelImageBytes+1, false); err == nil {
		t.Fatal("expected model image size rejection")
	}
	err := validateUploadFile(FileTypeImage, okPath, MaxModelImageBytes+1, false)
	if err == nil || !strings.Contains(err.Error(), "2 MB") {
		t.Fatalf("want 2 MB message, got %v", err)
	}
	// App images: 5MB
	if err := validateUploadFile(FileTypeImage, okPath, MaxAppImageBytes, true); err != nil {
		t.Fatal(err)
	}
	if err := validateUploadFile(FileTypeImage, okPath, MaxAppImageBytes+1, true); err == nil {
		t.Fatal("expected app image size rejection")
	}
	// 3MB OK for app, not for model
	mid := int64(MaxModelImageBytes + 1)
	if err := validateUploadFile(FileTypeImage, okPath, mid, true); err != nil {
		t.Fatal(err)
	}
	if err := validateUploadFile(FileTypeImage, okPath, mid, false); err == nil {
		t.Fatal("expected 3MB rejected as model image")
	}
}

func TestValidateUploadFileNonImageSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.bin")
	if err := validateUploadFile(FileTypeFile, path, MaxNonImageUploadBytes, false); err != nil {
		t.Fatal(err)
	}
	if err := validateUploadFile(FileTypeFile, path, MaxNonImageUploadBytes+1, false); err == nil {
		t.Fatal("expected size rejection")
	}
	if err := validateUploadFile(FileTypeFile, path, 0, false); err == nil {
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

func TestUploadFileRejectsOversizedModelImageInSandbox(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxModelImageBytes + 1); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	s := FileService{sandbox: true}
	_, err = s.UploadFile(FileUploadParams{Path: path, FileType: FileTypeImage})
	if err == nil {
		t.Fatal("expected oversized model image rejection")
	}
	_, err = s.UploadFile(FileUploadParams{Path: path, FileType: FileTypeImage, ForAppImage: true})
	if err != nil {
		t.Fatalf("3MB-ish should pass as app image if under 5MB: %v", err)
	}
}

func TestUploadFileRejectsOversizedAppImageInSandbox(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "huge.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxAppImageBytes + 1); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	s := FileService{sandbox: true}
	_, err = s.UploadFile(FileUploadParams{Path: path, FileType: FileTypeImage, ForAppImage: true})
	if err == nil {
		t.Fatal("expected oversized app image rejection")
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
