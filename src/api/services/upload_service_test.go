package services

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestUploadImage_NotConfigured(t *testing.T) {
	os.Unsetenv("GCS_BUCKET_NAME")
	t.Cleanup(func() { os.Unsetenv("GCS_BUCKET_NAME") })

	app := fiber.New()
	app.Post("/upload-image", UploadImage)

	body := &bytes.Buffer{}
	mp := multipart.NewWriter(body)
	part, _ := mp.CreateFormFile("file", "test.jpg")
	_, _ = part.Write([]byte("fake image"))
	_ = mp.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload-image", body)
	req.Header.Set("Content-Type", mp.FormDataContentType())
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusServiceUnavailable {
		t.Errorf("expected 503 when bucket not configured, got %d", res.StatusCode)
	}
}

func TestUploadImage_MissingFile(t *testing.T) {
	os.Setenv("GCS_BUCKET_NAME", "test-bucket")
	t.Cleanup(func() { os.Unsetenv("GCS_BUCKET_NAME") })

	app := fiber.New()
	app.Post("/upload-image", UploadImage)

	req := httptest.NewRequest(http.MethodPost, "/upload-image", strings.NewReader(""))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400 when file missing, got %d", res.StatusCode)
	}
}

func TestUploadImage_FileTooLarge(t *testing.T) {
	os.Setenv("GCS_BUCKET_NAME", "test-bucket")
	t.Cleanup(func() { os.Unsetenv("GCS_BUCKET_NAME") })

	app := fiber.New(fiber.Config{BodyLimit: 7 * 1024 * 1024})
	app.Post("/upload-image", UploadImage)

	body := &bytes.Buffer{}
	mp := multipart.NewWriter(body)
	part, _ := mp.CreateFormFile("file", "huge.jpg")
	big := make([]byte, 6*1024*1024)
	_, _ = part.Write(big)
	_ = mp.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload-image", body)
	req.Header.Set("Content-Type", mp.FormDataContentType())
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 when file too large, got %d", res.StatusCode)
	}
}

func TestUploadImage_InvalidContentType(t *testing.T) {
	os.Setenv("GCS_BUCKET_NAME", "test-bucket")
	t.Cleanup(func() { os.Unsetenv("GCS_BUCKET_NAME") })

	app := fiber.New()
	app.Post("/upload-image", UploadImage)

	body := &bytes.Buffer{}
	mp := multipart.NewWriter(body)
	part, _ := mp.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="file"; filename="x.pdf"`},
		"Content-Type":        {"application/pdf"},
	})
	_, _ = part.Write([]byte("fake pdf"))
	_ = mp.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload-image", body)
	req.Header.Set("Content-Type", mp.FormDataContentType())
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400 when content type not image, got %d", res.StatusCode)
	}
}

func TestUploadImage_Success(t *testing.T) {
	os.Setenv("GCS_BUCKET_NAME", "test-bucket")
	t.Cleanup(func() { os.Unsetenv("GCS_BUCKET_NAME") })

	original := uploadToBucketFunc
	uploadToBucketFunc = func(ctx context.Context, bucket, objectPath, contentType string, content io.Reader) (string, error) {
		if bucket != "test-bucket" {
			t.Errorf("expected bucket test-bucket, got %s", bucket)
		}
		if !strings.HasPrefix(objectPath, "portfolio-images/") {
			t.Errorf("expected path prefix portfolio-images/, got %s", objectPath)
		}
		return "https://storage.googleapis.com/test-bucket/" + objectPath, nil
	}
	t.Cleanup(func() { uploadToBucketFunc = original })

	app := fiber.New()
	app.Post("/upload-image", UploadImage)

	body := &bytes.Buffer{}
	mp := multipart.NewWriter(body)
	part, _ := mp.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="file"; filename="photo.jpg"`},
		"Content-Type":        {"image/jpeg"},
	})
	_, _ = part.Write([]byte("fake jpeg content"))
	_ = mp.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload-image", body)
	req.Header.Set("Content-Type", mp.FormDataContentType())
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d body=%s", res.StatusCode, string(raw))
		return
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	u, ok := out["url"]
	if !ok {
		t.Errorf("expected url in response, got %v", out)
		return
	}
	if s, ok := u.(string); !ok || s == "" {
		t.Errorf("expected url string, got %v", u)
	}
}

func TestIsAllowedImageContentType(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/gif", true},
		{"image/webp", true},
		{"IMAGE/JPEG", true},
		{"application/pdf", false},
		{"", false},
		{"text/plain", false},
	}
	for _, tt := range tests {
		if got := isAllowedImageContentType(tt.ct); got != tt.want {
			t.Errorf("isAllowedImageContentType(%q) = %v, want %v", tt.ct, got, tt.want)
		}
	}
}

func TestMapExt(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{"jpg", "jpg"},
		{"jpeg", "jpeg"},
		{"png", "png"},
		{"gif", "gif"},
		{"webp", "webp"},
		{"bmp", "jpg"},
		{"", "jpg"},
	}
	for _, tt := range tests {
		if got := mapExt(tt.ext); got != tt.want {
			t.Errorf("mapExt(%q) = %q, want %q", tt.ext, got, tt.want)
		}
	}
}
