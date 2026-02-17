package services

import (
	userModel "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
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

// ListPublicSkills godoc
// @Summary      Listar skills publicas
// @Description  Devuelve experiencias con tag "skill" y visibility=public. Soporta ETag.
// @Tags         Skills
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "items"
// @Success      304  "Not Modified"
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/skills [get]
func ListPublicSkills(c fiber.Ctx) error {
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

// ListAllSkills godoc
// @Summary      Listar todas las skills
// @Description  Devuelve todas las experiencias con tag "skill". Requiere JWT.
// @Tags         Skills
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}  "items"
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/private/skills [get]
func ListAllSkills(c fiber.Ctx) error {
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

// CreateSkill godoc
// @Summary      Crear skill
// @Description  Crea una nueva skill. Agrega tag "skill" automaticamente. Requiere JWT.
// @Tags         Skills
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        skill  body  object{title=string,summary=string,body=string,imageUrls=[]string,tags=[]string,visibility=string}  true  "Datos"
// @Success      200  {object}  userModel.Experience
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/private/skills [post]
func CreateSkill(c fiber.Ctx) error {
	var payload experiencePayload
	if err := c.Bind().Body(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	sanitizePayload(&payload)

	if payload.Title == "" {
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
		Title:      payload.Title,
		Summary:    payload.Summary,
		Body:       payload.Body,
		ImageURLs:  payload.ImageURLs,
		Tags:       ensureSkillTag(payload.Tags),
		Visibility: payload.Visibility,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	experiences = append(experiences, item)
	if err := saveExperiences(experiences); err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "save_skill_failed", "No se pudo guardar la capacidad", err.Error())
	}

	return apiresponse.Success(c, item)
}

// UpdateSkill godoc
// @Summary      Actualizar skill
// @Description  Actualiza una skill por ID. Requiere JWT.
// @Tags         Skills
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id     path  string  true  "ID de la skill"
// @Param        skill  body  object{title=string,summary=string,body=string,imageUrls=[]string,tags=[]string,visibility=string}  true  "Datos"
// @Success      200  {object}  userModel.Experience
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/private/skills/{id} [put]
func UpdateSkill(c fiber.Ctx) error {
	id := c.Params("id")
	if !validatePayloadID(id) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_id", "Formato de ID invalido", nil)
	}

	var payload experiencePayload
	if err := c.Bind().Body(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	sanitizePayload(&payload)

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

		if payload.Title != "" {
			item.Title = payload.Title
		}
		item.Summary = payload.Summary
		item.Body = payload.Body
		item.ImageURLs = payload.ImageURLs
		item.Tags = ensureSkillTag(payload.Tags)
		item.Visibility = payload.Visibility
		item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		experiences[i] = item

		if err := saveExperiences(experiences); err != nil {
			return apiresponse.Error(c, fiber.StatusInternalServerError, "save_skill_failed", "No se pudo actualizar la capacidad", err.Error())
		}
		return apiresponse.Success(c, item)
	}

	return apiresponse.Error(c, fiber.StatusNotFound, "skill_not_found", "Capacidad no encontrada", nil)
}

// DeleteSkill godoc
// @Summary      Eliminar skill
// @Description  Elimina una skill por ID. Requiere JWT.
// @Tags         Skills
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "ID de la skill"
// @Success      200  {object}  map[string]interface{}  "deleted, id"
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/private/skills/{id} [delete]
func DeleteSkill(c fiber.Ctx) error {
	id := c.Params("id")
	if !validatePayloadID(id) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_id", "Formato de ID invalido", nil)
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
