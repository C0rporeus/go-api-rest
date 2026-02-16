package authServices

import (
	userModel "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var experienceStoreLock sync.RWMutex

func experiencesFilePath() string {
	dataDir := os.Getenv("PORTFOLIO_DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	return filepath.Join(dataDir, "experiences.json")
}

func loadExperiences() ([]userModel.Experience, error) {
	filePath := experiencesFilePath()
	data, err := os.ReadFile(filePath)
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
	if err := os.MkdirAll(filepath.Dir(filePath), constants.DirPermission); err != nil {
		return err
	}
	data, err := json.MarshalIndent(experiences, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, constants.FilePermission)
}

type experiencePayload struct {
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Body       string   `json:"body"`
	ImageURLs  []string `json:"imageUrls"`
	Tags       []string `json:"tags"`
	Visibility string   `json:"visibility"`
}

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

func ListPublicExperiences(c *fiber.Ctx) error {
	experienceStoreLock.RLock()
	defer experienceStoreLock.RUnlock()

	experiences, err := loadExperiences()
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_experiences_failed", "No se pudo cargar experiencias", err.Error())
	}

	publicExperiences := make([]userModel.Experience, 0, len(experiences))
	for _, item := range experiences {
		if item.Visibility == constants.VisibilityPublic {
			publicExperiences = append(publicExperiences, item)
		}
	}

	etag := buildCollectionETag(publicExperiences)
	setPublicCollectionCacheHeaders(c, etag)
	if matchesIfNoneMatchHeader(c.Get("If-None-Match"), etag) {
		return c.SendStatus(fiber.StatusNotModified)
	}

	return apiresponse.Success(c, fiber.Map{"items": publicExperiences})
}

func ListAllExperiences(c *fiber.Ctx) error {
	experienceStoreLock.RLock()
	defer experienceStoreLock.RUnlock()

	experiences, err := loadExperiences()
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_experiences_failed", "No se pudo cargar experiencias", err.Error())
	}
	return apiresponse.Success(c, fiber.Map{"items": experiences})
}

func CreateExperience(c *fiber.Ctx) error {
	var payload experiencePayload
	if err := c.BodyParser(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}
	if strings.TrimSpace(payload.Title) == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_title", "El titulo es requerido", nil)
	}

	experienceStoreLock.Lock()
	defer experienceStoreLock.Unlock()

	experiences, err := loadExperiences()
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_experiences_failed", "No se pudo cargar experiencias", err.Error())
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item := userModel.Experience{
		ID:         uuid.NewString(),
		Title:      strings.TrimSpace(payload.Title),
		Summary:    strings.TrimSpace(payload.Summary),
		Body:       strings.TrimSpace(payload.Body),
		ImageURLs:  normalizeImageURLs(payload.ImageURLs),
		Tags:       payload.Tags,
		Visibility: normalizeVisibility(payload.Visibility),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	experiences = append(experiences, item)
	if err := saveExperiences(experiences); err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "save_experience_failed", "No se pudo guardar la experiencia", err.Error())
	}

	return apiresponse.Success(c, item)
}

func UpdateExperience(c *fiber.Ctx) error {
	id := c.Params("id")
	if strings.TrimSpace(id) == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_id", "El id es requerido", nil)
	}

	var payload experiencePayload
	if err := c.BodyParser(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	experienceStoreLock.Lock()
	defer experienceStoreLock.Unlock()

	experiences, err := loadExperiences()
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_experiences_failed", "No se pudo cargar experiencias", err.Error())
	}

	for i, item := range experiences {
		if item.ID == id {
			if strings.TrimSpace(payload.Title) != "" {
				item.Title = strings.TrimSpace(payload.Title)
			}
			item.Summary = strings.TrimSpace(payload.Summary)
			item.Body = strings.TrimSpace(payload.Body)
			item.ImageURLs = normalizeImageURLs(payload.ImageURLs)
			item.Tags = payload.Tags
			item.Visibility = normalizeVisibility(payload.Visibility)
			item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			experiences[i] = item

			if err := saveExperiences(experiences); err != nil {
				return apiresponse.Error(c, fiber.StatusInternalServerError, "save_experience_failed", "No se pudo actualizar la experiencia", err.Error())
			}
			return apiresponse.Success(c, item)
		}
	}

	return apiresponse.Error(c, fiber.StatusNotFound, "experience_not_found", "Experiencia no encontrada", nil)
}

func DeleteExperience(c *fiber.Ctx) error {
	id := c.Params("id")
	if strings.TrimSpace(id) == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_id", "El id es requerido", nil)
	}

	experienceStoreLock.Lock()
	defer experienceStoreLock.Unlock()

	experiences, err := loadExperiences()
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_experiences_failed", "No se pudo cargar experiencias", err.Error())
	}

	filtered := make([]userModel.Experience, 0, len(experiences))
	found := false
	for _, item := range experiences {
		if item.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}

	if !found {
		return apiresponse.Error(c, fiber.StatusNotFound, "experience_not_found", "Experiencia no encontrada", nil)
	}
	if err := saveExperiences(filtered); err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "save_experience_failed", "No se pudo eliminar la experiencia", err.Error())
	}
	return apiresponse.Success(c, fiber.Map{"deleted": true, "id": id})
}
