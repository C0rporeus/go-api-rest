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

// stripQueryParams removes query-string parameters from a URL, including
// the case where '?' was URL-encoded as '%3F' (e.g. when a signed URL was
// saved back by the editor and then re-processed on read).
func stripQueryParams(u string) string {
	if pos := strings.Index(u, "?"); pos >= 0 {
		return u[:pos]
	}
	// Also handle URL-encoded '?' left by a previous signing round-trip.
	if pos := strings.Index(strings.ToLower(u), "%3f"); pos >= 0 {
		return u[:pos]
	}
	return u
}

// signSingleURL signs a GCS URL if it matches our bucket; external URLs pass through.
// It always strips any pre-existing query parameters so that a URL that was
// previously signed and saved back (e.g. by the admin editor) is not double-signed.
func signSingleURL(ctx context.Context, rawURL, gcsPrefix string, expiry time.Duration) string {
	if gcsPrefix == "" || !strings.HasPrefix(rawURL, gcsPrefix) {
		return rawURL
	}
	baseURL := stripQueryParams(rawURL)
	objectPath := strings.TrimPrefix(baseURL, gcsPrefix)
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

// StripBodySignedParams removes GCS query parameters from all image URLs in an
// HTML body string. Call this before persisting the body so the database never
// stores time-limited signed URLs that would expire or cause double-signing.
func StripBodySignedParams(body string) string {
	prefix := constants.GCSURLPrefix()
	if prefix == "" || !strings.Contains(body, prefix) {
		return body
	}
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
		clean := stripQueryParams(rawURL)
		result = result[:start] + clean + result[end:]
		idx = start + len(clean)
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
