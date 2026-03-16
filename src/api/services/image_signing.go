package services

import (
	"context"
	"log"
	"strings"
	"time"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/constants"
)

// signURLFunc is injectable for tests. Default: signGCSURL.
var signURLFunc = signGCSURL

// signSingleURL signs a GCS URL if it matches our bucket; external URLs pass through.
func signSingleURL(ctx context.Context, rawURL, gcsPrefix string, expiry time.Duration) string {
	if gcsPrefix == "" || !strings.HasPrefix(rawURL, gcsPrefix) {
		return rawURL
	}
	objectPath := strings.TrimPrefix(rawURL, gcsPrefix)
	bucket := constants.GCSBucketName()
	signed, err := signURLFunc(ctx, bucket, objectPath, expiry)
	if err != nil {
		log.Printf("[image_signing] failed to sign %s: %v", objectPath, err)
		return rawURL
	}
	return signed
}

// SignImageURLs processes a slice of URLs, signing GCS ones and passing through external ones.
func SignImageURLs(ctx context.Context, urls []string) []string {
	prefix := constants.GCSURLPrefix()
	if prefix == "" {
		return urls
	}
	expiry := constants.SignedURLExpiry()
	result := make([]string, len(urls))
	for i, u := range urls {
		result[i] = signSingleURL(ctx, u, prefix, expiry)
	}
	return result
}

// SignBodyImageURLs replaces GCS image URLs in HTML body with signed versions.
func SignBodyImageURLs(ctx context.Context, body string) string {
	prefix := constants.GCSURLPrefix()
	if prefix == "" || !strings.Contains(body, prefix) {
		return body
	}
	expiry := constants.SignedURLExpiry()
	result := body
	idx := 0
	for {
		pos := strings.Index(result[idx:], prefix)
		if pos < 0 {
			break
		}
		start := idx + pos
		end := start + len(prefix)
		for end < len(result) && result[end] != '"' && result[end] != '\'' && result[end] != ' ' && result[end] != '<' && result[end] != '>' {
			end++
		}
		rawURL := result[start:end]
		signed := signSingleURL(ctx, rawURL, prefix, expiry)
		result = result[:start] + signed + result[end:]
		idx = start + len(signed)
	}
	return result
}

// SignExperienceImageURLs signs all GCS image URLs in a single experience (imageUrls + body).
func SignExperienceImageURLs(ctx context.Context, exp *models.Experience) {
	exp.ImageURLs = SignImageURLs(ctx, exp.ImageURLs)
	exp.Body = SignBodyImageURLs(ctx, exp.Body)
}

// SignExperienceList signs all GCS image URLs in a list of experiences.
func SignExperienceList(ctx context.Context, items []models.Experience) {
	for i := range items {
		SignExperienceImageURLs(ctx, &items[i])
	}
}
