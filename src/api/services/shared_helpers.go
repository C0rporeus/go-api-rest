package services

import (
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/pkg/sanitizer"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	userModel "backend-yonathan/src/models"
)

// experienceStoreLock protects concurrent access to the experiences JSON store.
// Use RLock/RUnlock for read operations, Lock/Unlock for writes.
var experienceStoreLock sync.RWMutex

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

// --- Injectable file I/O (swap in tests) ---

var readFileFunc = func(path string) ([]byte, error) {
	return os.ReadFile(path)
}

var writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

var mkdirAllFunc = func(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// --- Store helpers (file-based persistence) ---

func experiencesFilePath() string {
	dataDir := os.Getenv(constants.DataDirEnvVar)
	if dataDir == "" {
		dataDir = constants.DefaultDataDir
	}
	return filepath.Join(dataDir, constants.ExperiencesFilename)
}

func loadExperiences() ([]userModel.Experience, error) {
	filePath := experiencesFilePath()
	data, err := readFileFunc(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []userModel.Experience{}, nil
		}
		return nil, err
	}

	var experiences []userModel.Experience
	if len(data) == 0 {
		return []userModel.Experience{}, nil
	}
	if err := json.Unmarshal(data, &experiences); err != nil {
		return nil, err
	}
	return experiences, nil
}

func saveExperiences(experiences []userModel.Experience) error {
	filePath := experiencesFilePath()
	if err := mkdirAllFunc(filepath.Dir(filePath), constants.DirPermission); err != nil {
		return err
	}
	data, err := json.MarshalIndent(experiences, "", "  ")
	if err != nil {
		return err
	}
	return writeFileFunc(filePath, data, constants.FilePermission)
}

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

