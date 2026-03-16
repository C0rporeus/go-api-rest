package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	models "backend-yonathan/src/models"
)

func stubSignURL(prefix string) func(ctx context.Context, bucket, object string, expiry time.Duration) (string, error) {
	return func(_ context.Context, _, object string, _ time.Duration) (string, error) {
		return prefix + object + "?signed=1", nil
	}
}

func failingSignURL(_ context.Context, _, _ string, _ time.Duration) (string, error) {
	return "", fmt.Errorf("sign error")
}

func TestSignImageURLs_NoBucket(t *testing.T) {
	t.Setenv("GCS_BUCKET_NAME", "")
	urls := []string{"https://example.com/a.jpg"}
	result := SignImageURLs(context.Background(), urls)
	if len(result) != 1 || result[0] != urls[0] {
		t.Fatalf("expected passthrough, got %v", result)
	}
}

func TestSignImageURLs_MixedURLs(t *testing.T) {
	t.Setenv("GCS_BUCKET_NAME", "my-bucket")
	t.Setenv("SIGNED_URL_EXPIRY_HOURS", "1")

	orig := signURLFunc
	signURLFunc = stubSignURL("https://signed.example.com/")
	t.Cleanup(func() { signURLFunc = orig })

	urls := []string{
		"https://storage.googleapis.com/my-bucket/portfolio-images/abc.jpg",
		"https://images.unsplash.com/photo-123",
		"https://storage.googleapis.com/my-bucket/portfolio-images/def.png",
	}

	result := SignImageURLs(context.Background(), urls)

	if result[0] != "https://signed.example.com/portfolio-images/abc.jpg?signed=1" {
		t.Errorf("expected signed URL for [0], got %s", result[0])
	}
	if result[1] != "https://images.unsplash.com/photo-123" {
		t.Errorf("expected passthrough for [1], got %s", result[1])
	}
	if result[2] != "https://signed.example.com/portfolio-images/def.png?signed=1" {
		t.Errorf("expected signed URL for [2], got %s", result[2])
	}
}

func TestSignImageURLs_SignError_Passthrough(t *testing.T) {
	t.Setenv("GCS_BUCKET_NAME", "my-bucket")

	orig := signURLFunc
	signURLFunc = failingSignURL
	t.Cleanup(func() { signURLFunc = orig })

	urls := []string{"https://storage.googleapis.com/my-bucket/portfolio-images/abc.jpg"}
	result := SignImageURLs(context.Background(), urls)

	if result[0] != urls[0] {
		t.Errorf("on sign error, expected raw URL passthrough, got %s", result[0])
	}
}

func TestSignBodyImageURLs_NoBucket(t *testing.T) {
	t.Setenv("GCS_BUCKET_NAME", "")
	body := `<p><img src="https://storage.googleapis.com/my-bucket/img.jpg"></p>`
	result := SignBodyImageURLs(context.Background(), body)
	if result != body {
		t.Fatalf("expected passthrough, got %s", result)
	}
}

func TestSignBodyImageURLs_ReplacesGCSURLs(t *testing.T) {
	t.Setenv("GCS_BUCKET_NAME", "my-bucket")
	t.Setenv("SIGNED_URL_EXPIRY_HOURS", "2")

	orig := signURLFunc
	signURLFunc = stubSignURL("https://signed.example.com/")
	t.Cleanup(func() { signURLFunc = orig })

	body := `<p>Text</p><img src="https://storage.googleapis.com/my-bucket/portfolio-images/a.jpg"><p>More</p><img src="https://images.unsplash.com/photo">`
	result := SignBodyImageURLs(context.Background(), body)

	expected := `<p>Text</p><img src="https://signed.example.com/portfolio-images/a.jpg?signed=1"><p>More</p><img src="https://images.unsplash.com/photo">`
	if result != expected {
		t.Errorf("unexpected body:\ngot:  %s\nwant: %s", result, expected)
	}
}

func TestSignBodyImageURLs_MultipleGCSURLs(t *testing.T) {
	t.Setenv("GCS_BUCKET_NAME", "my-bucket")

	orig := signURLFunc
	signURLFunc = stubSignURL("https://signed.example.com/")
	t.Cleanup(func() { signURLFunc = orig })

	body := `<img src="https://storage.googleapis.com/my-bucket/a.jpg"> and <img src="https://storage.googleapis.com/my-bucket/b.png">`
	result := SignBodyImageURLs(context.Background(), body)

	if !strContains(result, "https://signed.example.com/a.jpg?signed=1") {
		t.Errorf("expected first URL signed, got %s", result)
	}
	if !strContains(result, "https://signed.example.com/b.png?signed=1") {
		t.Errorf("expected second URL signed, got %s", result)
	}
}

func TestSignExperienceImageURLs(t *testing.T) {
	t.Setenv("GCS_BUCKET_NAME", "my-bucket")

	orig := signURLFunc
	signURLFunc = stubSignURL("https://signed.example.com/")
	t.Cleanup(func() { signURLFunc = orig })

	exp := models.Experience{
		ID:    "test-id",
		Title: "Test",
	}
	exp.ImageURLs = []string{
		"https://storage.googleapis.com/my-bucket/portfolio-images/thumb.jpg",
		"https://cdn.external.com/photo.jpg",
	}
	exp.Body = `<img src="https://storage.googleapis.com/my-bucket/portfolio-images/inline.png">`

	SignExperienceImageURLs(context.Background(), &exp)

	if exp.ImageURLs[0] != "https://signed.example.com/portfolio-images/thumb.jpg?signed=1" {
		t.Errorf("expected signed imageUrl[0], got %s", exp.ImageURLs[0])
	}
	if exp.ImageURLs[1] != "https://cdn.external.com/photo.jpg" {
		t.Errorf("expected passthrough imageUrl[1], got %s", exp.ImageURLs[1])
	}
	if !strContains(exp.Body, "https://signed.example.com/portfolio-images/inline.png?signed=1") {
		t.Errorf("expected signed body URL, got %s", exp.Body)
	}
}

func strContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
