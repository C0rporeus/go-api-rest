package services

import (
	userModel "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/constants"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
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
	if len(urls) == 0 {
		return []string{}
	}
	normalized := make([]string, 0, len(urls))
	for _, url := range urls {
		clean := strings.TrimSpace(url)
		if clean == "" {
			continue
		}
		normalized = append(normalized, clean)
	}
	return normalized
}

// --- HTTP / caching helpers ---

func buildCollectionETag(items []userModel.Experience) string {
	payload, err := json.Marshal(items)
	if err != nil {
		return ""
	}
	sum := sha1.Sum(payload)
	return "W/\"" + hex.EncodeToString(sum[:]) + "\""
}

func matchesIfNoneMatchHeader(ifNoneMatch, etag string) bool {
	if etag == "" {
		return false
	}
	if strings.TrimSpace(ifNoneMatch) == "*" {
		return true
	}

	for _, candidate := range strings.Split(ifNoneMatch, ",") {
		if strings.TrimSpace(candidate) == etag {
			return true
		}
	}
	return false
}

func setPublicCollectionCacheHeaders(c *fiber.Ctx, etag string) {
	c.Set("Cache-Control", constants.PublicCollectionCacheControl)
	if etag != "" {
		c.Set("ETag", etag)
	}
}
