package services

import (
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/pkg/sanitizer"
	"strings"
)

// experiencePayload is the common request body for creating/updating
// experiences and skills.
type experiencePayload struct {
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Body       string   `json:"body"`
	ImageURLs  []string `json:"imageUrls"`
	Tags       []string `json:"tags"`
	Visibility string   `json:"visibility"`
}

// --- Injectable function pattern (dependency injection convention) ---
//
// This package uses package-level `var` functions as injectable dependencies.
// Each service declares its external I/O operations as `var` function variables
// that default to real implementations.
//
// In tests, swap them via direct assignment + t.Cleanup to restore:
//
//	original := logContactMessage
//	logContactMessage = func(...) { ... }
//	t.Cleanup(func() { logContactMessage = original })
//
// Files using this pattern:
//   - contact.service.go → logContactMessage (logging)

// --- Data normalization helpers ---

func normalizeVisibility(visibility string) string {
	v := strings.ToLower(strings.TrimSpace(visibility))
	if v != constants.VisibilityPrivate {
		return constants.VisibilityPublic
	}
	return v
}

func normalizeImageURLs(urls []string) []string {
	return sanitizer.ValidateURLSlice(urls, constants.MaxImageURLCount)
}

func normalizeTags(tags []string) []string {
	return sanitizer.SanitizeSlice(tags, constants.MaxTagCount, func(tag string) string {
		cleaned := sanitizer.SanitizePlainText(tag, constants.MaxTagLength)
		return strings.ToLower(cleaned)
	})
}

// sanitizePayload applies input sanitization and length limits to an experience/skill payload.
// Title and Summary are stripped of HTML. Body allows safe HTML (UGC policy).
// ImageURLs are validated as HTTP/HTTPS URLs. Tags are sanitized and lowercased.
func sanitizePayload(p *experiencePayload) {
	p.Title = sanitizer.SanitizePlainText(p.Title, constants.MaxTitleLength)
	p.Summary = sanitizer.SanitizePlainText(p.Summary, constants.MaxSummaryLength)
	p.Body = sanitizer.SanitizeRichText(p.Body, constants.MaxBodyLength)
	p.ImageURLs = normalizeImageURLs(p.ImageURLs)
	p.Tags = normalizeTags(p.Tags)
	p.Visibility = normalizeVisibility(p.Visibility)
}

// validatePayloadID validates a resource UUID from the URL path.
func validatePayloadID(id string) bool {
	return sanitizer.IsValidUUID(id)
}
