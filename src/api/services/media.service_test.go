package services

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

const testMediaName = "63a96a8f-df27-40b7-930e-78a2b94e4242.jpeg"

func newMediaApp(t *testing.T, read func(ctx context.Context, bucket, object string) ([]byte, string, error)) *fiber.App {
	t.Helper()
	t.Setenv("GCS_BUCKET_NAME", "test-bucket")
	original := readMediaFunc
	readMediaFunc = read
	t.Cleanup(func() { readMediaFunc = original })
	app := fiber.New()
	app.Get("/api/media/:name", ServePublicImage)
	return app
}

func TestServePublicImage_ServesObjectWithCacheHeaders(t *testing.T) {
	var gotObject string
	app := newMediaApp(t, func(_ context.Context, bucket, object string) ([]byte, string, error) {
		gotObject = bucket + "/" + object
		return []byte("jpeg-bytes"), "image/jpeg", nil
	})

	res, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/media/"+testMediaName, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != fiber.StatusOK || string(body) != "jpeg-bytes" {
		t.Fatalf("expected 200 with image body, got %d %q", res.StatusCode, body)
	}
	if gotObject != "test-bucket/portfolio-images/"+testMediaName {
		t.Errorf("unexpected object read: %s", gotObject)
	}
	if ct := res.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %s", ct)
	}
	if cc := res.Header.Get("Cache-Control"); cc == "" {
		t.Error("expected Cache-Control header")
	}
}

func TestServePublicImage_RejectsInvalidNames(t *testing.T) {
	app := newMediaApp(t, func(context.Context, string, string) ([]byte, string, error) {
		t.Fatal("storage must not be read for invalid names")
		return nil, "", nil
	})
	for _, name := range []string{"secret.txt", "..%2Fother.jpg", "not-a-uuid.jpg", "63a96a8f-df27-40b7-930e-78a2b94e4242.svg"} {
		res, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/media/"+name, nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.StatusCode != fiber.StatusNotFound {
			t.Errorf("%s: expected 404, got %d", name, res.StatusCode)
		}
	}
}

func TestServePublicImage_NotFoundAndNonImage(t *testing.T) {
	cases := map[string]func(context.Context, string, string) ([]byte, string, error){
		"missing":   func(context.Context, string, string) ([]byte, string, error) { return nil, "", errMediaNotFound },
		"non-image": func(context.Context, string, string) ([]byte, string, error) { return []byte("x"), "text/html", nil },
	}
	for label, read := range cases {
		app := newMediaApp(t, read)
		res, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/media/"+testMediaName, nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.StatusCode != fiber.StatusNotFound {
			t.Errorf("%s: expected 404, got %d", label, res.StatusCode)
		}
	}
}

func TestServePublicImage_OGVariantIsLandscapeJPEG(t *testing.T) {
	var portrait bytes.Buffer
	if err := png.Encode(&portrait, image.NewRGBA(image.Rect(0, 0, 400, 600))); err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	app := newMediaApp(t, func(context.Context, string, string) ([]byte, string, error) {
		return portrait.Bytes(), "image/png", nil
	})

	res, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/media/"+testMediaName+"?variant=og", nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("expected image/jpeg, got %s", res.Header.Get("Content-Type"))
	}
	if corp := res.Header.Get("Cross-Origin-Resource-Policy"); corp != "cross-origin" {
		t.Errorf("expected CORP cross-origin, got %q", corp)
	}
	cfg, err := jpeg.DecodeConfig(res.Body)
	if err != nil {
		t.Fatalf("decode jpeg: %v", err)
	}
	if cfg.Width != ogImageWidth || cfg.Height != ogImageHeight {
		t.Errorf("expected %dx%d, got %dx%d", ogImageWidth, ogImageHeight, cfg.Width, cfg.Height)
	}
}

func TestCoverCropRect_KeepsOGAspectRatio(t *testing.T) {
	for _, bounds := range []image.Rectangle{image.Rect(0, 0, 1024, 1536), image.Rect(0, 0, 3000, 1000)} {
		r := coverCropRect(bounds)
		if !r.In(bounds) {
			t.Errorf("crop %v outside %v", r, bounds)
		}
		ratio := float64(r.Dx()) / float64(r.Dy())
		if ratio < 1.88 || ratio > 1.93 {
			t.Errorf("crop %v ratio %.3f, want ~1.905", r, ratio)
		}
	}
}
