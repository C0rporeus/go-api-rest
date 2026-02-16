package authServices

import (
	userModel "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func normalizeTagValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isSkillExperience(item userModel.Experience) bool {
	for _, tag := range item.Tags {
		normalized := normalizeTagValue(tag)
		for _, skillTag := range constants.SkillTags {
			if normalized == skillTag {
				return true
			}
		}
	}
	return false
}

func ensureSkillTag(tags []string) []string {
	normalized := make([]string, 0, len(tags)+1)
	seen := map[string]bool{}
	for _, tag := range tags {
		clean := normalizeTagValue(tag)
		if clean == "" || seen[clean] {
			continue
		}
		normalized = append(normalized, clean)
		seen[clean] = true
	}

	if !seen["skill"] {
		normalized = append(normalized, "skill")
	}

	return normalized
}

func ListPublicSkills(c *fiber.Ctx) error {
	experienceStoreLock.RLock()
	defer experienceStoreLock.RUnlock()

	experiences, err := loadExperiences()
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_skills_failed", "No se pudo cargar capacidades", err.Error())
	}

	skills := make([]userModel.Experience, 0, len(experiences))
	for _, item := range experiences {
		if item.Visibility == constants.VisibilityPublic && isSkillExperience(item) {
			skills = append(skills, item)
		}
	}

	etag := buildCollectionETag(skills)
	setPublicCollectionCacheHeaders(c, etag)
	if matchesIfNoneMatchHeader(c.Get("If-None-Match"), etag) {
		return c.SendStatus(fiber.StatusNotModified)
	}

	return apiresponse.Success(c, fiber.Map{"items": skills})
}

func ListAllSkills(c *fiber.Ctx) error {
	experienceStoreLock.RLock()
	defer experienceStoreLock.RUnlock()

	experiences, err := loadExperiences()
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_skills_failed", "No se pudo cargar capacidades", err.Error())
	}

	skills := make([]userModel.Experience, 0, len(experiences))
	for _, item := range experiences {
		if isSkillExperience(item) {
			skills = append(skills, item)
		}
	}

	return apiresponse.Success(c, fiber.Map{"items": skills})
}

func CreateSkill(c *fiber.Ctx) error {
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
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_skills_failed", "No se pudo cargar capacidades", err.Error())
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item := userModel.Experience{
		ID:         uuid.NewString(),
		Title:      strings.TrimSpace(payload.Title),
		Summary:    strings.TrimSpace(payload.Summary),
		Body:       strings.TrimSpace(payload.Body),
		ImageURLs:  normalizeImageURLs(payload.ImageURLs),
		Tags:       ensureSkillTag(payload.Tags),
		Visibility: normalizeVisibility(payload.Visibility),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	experiences = append(experiences, item)
	if err := saveExperiences(experiences); err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "save_skill_failed", "No se pudo guardar la capacidad", err.Error())
	}

	return apiresponse.Success(c, item)
}

func UpdateSkill(c *fiber.Ctx) error {
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
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_skills_failed", "No se pudo cargar capacidades", err.Error())
	}

	for i, item := range experiences {
		if item.ID != id {
			continue
		}
		if !isSkillExperience(item) {
			return apiresponse.Error(c, fiber.StatusNotFound, "skill_not_found", "Capacidad no encontrada", nil)
		}

		if strings.TrimSpace(payload.Title) != "" {
			item.Title = strings.TrimSpace(payload.Title)
		}
		item.Summary = strings.TrimSpace(payload.Summary)
		item.Body = strings.TrimSpace(payload.Body)
		item.ImageURLs = normalizeImageURLs(payload.ImageURLs)
		item.Tags = ensureSkillTag(payload.Tags)
		item.Visibility = normalizeVisibility(payload.Visibility)
		item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		experiences[i] = item

		if err := saveExperiences(experiences); err != nil {
			return apiresponse.Error(c, fiber.StatusInternalServerError, "save_skill_failed", "No se pudo actualizar la capacidad", err.Error())
		}
		return apiresponse.Success(c, item)
	}

	return apiresponse.Error(c, fiber.StatusNotFound, "skill_not_found", "Capacidad no encontrada", nil)
}

func DeleteSkill(c *fiber.Ctx) error {
	id := c.Params("id")
	if strings.TrimSpace(id) == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_id", "El id es requerido", nil)
	}

	experienceStoreLock.Lock()
	defer experienceStoreLock.Unlock()

	experiences, err := loadExperiences()
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_skills_failed", "No se pudo cargar capacidades", err.Error())
	}

	filtered := make([]userModel.Experience, 0, len(experiences))
	found := false

	for _, item := range experiences {
		if item.ID == id && isSkillExperience(item) {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}

	if !found {
		return apiresponse.Error(c, fiber.StatusNotFound, "skill_not_found", "Capacidad no encontrada", nil)
	}
	if err := saveExperiences(filtered); err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "save_skill_failed", "No se pudo eliminar la capacidad", err.Error())
	}

	return apiresponse.Success(c, fiber.Map{"deleted": true, "id": id})
}
