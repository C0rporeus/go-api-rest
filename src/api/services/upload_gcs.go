package services

import (
	"context"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
)

// saEmail caches the service account email used for GCS URL signing.
var (
	saEmail     string
	saEmailOnce sync.Once
)

// resolveServiceAccountEmail returns the SA email for signing GCS URLs.
// Priority: GCP_SERVICE_ACCOUNT_EMAIL env var → GCP metadata server.
func resolveServiceAccountEmail() string {
	saEmailOnce.Do(func() {
		if email := os.Getenv("GCP_SERVICE_ACCOUNT_EMAIL"); email != "" {
			saEmail = email
			return
		}
		// On Cloud Run / GCE the metadata server is always available.
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		email, err := metadata.EmailWithContext(ctx, "default")
		if err != nil {
			log.Printf("[upload_gcs] could not resolve SA email for signing: %v", err)
			return
		}
		saEmail = email
		log.Printf("[upload_gcs] resolved signing SA: %s", saEmail)
	})
	return saEmail
}

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
// GoogleAccessID is required on Cloud Run so the SDK uses the IAM
// signBlob API instead of a local private key.
func signGCSURL(ctx context.Context, bucketName, objectPath string, expiry time.Duration) (string, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	opts := &storage.SignedURLOptions{
		GoogleAccessID: resolveServiceAccountEmail(),
		Scheme:         storage.SigningSchemeV4,
		Method:         "GET",
		Expires:        time.Now().Add(expiry),
	}
	return client.Bucket(bucketName).SignedURL(objectPath, opts)
}
