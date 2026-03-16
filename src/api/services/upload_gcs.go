package services

import (
	"context"
	"io"
	"time"

	"cloud.google.com/go/storage"
)

// uploadToGCS writes content to GCS and returns the canonical storage URL.
func uploadToGCS(ctx context.Context, bucketName, objectPath, contentType string, content io.Reader) (string, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	w := client.Bucket(bucketName).Object(objectPath).NewWriter(ctx)
	w.ContentType = contentType
	w.CacheControl = "private, max-age=0"
	if _, err := io.Copy(w, content); err != nil {
		_ = w.Close()
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	return "https://storage.googleapis.com/" + bucketName + "/" + objectPath, nil
}

// signGCSURL generates a V4 signed URL for reading a GCS object.
func signGCSURL(ctx context.Context, bucketName, objectPath string, expiry time.Duration) (string, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expiry),
	}
	return client.Bucket(bucketName).SignedURL(objectPath, opts)
}
