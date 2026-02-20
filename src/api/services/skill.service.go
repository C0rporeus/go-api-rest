package services

import (
	"context"
	"errors"
	"strings"
	"time"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/repository"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// SkillService handles skill CRUD business logic.
// Skills are experiences that carry one of the recognized skill tags.
type SkillService struct {
	repo repository.ExperienceRepository
}

// NewSkillService creates a SkillService backed by the given ExperienceRepository.
func NewSkillService(repo repository.ExperienceRepository) *SkillService {
	return &SkillService{repo: repo}
}

func normalizeTagValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isSkillExperience(item models.Experience) bool {
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
func (s *SkillService) ListPublicSkills(c fiber.Ctx) error {
	all, err := s.repo.List(context.Background())
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_skills_failed", "No se pudo cargar capacidades", err.Error())
	}

	skills := make([]models.Experience, 0, len(all))
	for _, item := range all {
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
func (s *SkillService) ListAllSkills(c fiber.Ctx) error {
	all, err := s.repo.List(context.Background())
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_skills_failed", "No se pudo cargar capacidades", err.Error())
	}

	skills := make([]models.Experience, 0, len(all))
	for _, item := range all {
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
func (s *SkillService) CreateSkill(c fiber.Ctx) error {
	var payload experiencePayload
	if err := c.Bind().Body(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	sanitizePayload(&payload)

	if payload.Title == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_title", "El titulo es requerido", nil)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item := models.Experience{
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

	if err := s.repo.Create(context.Background(), item); err != nil {
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
func (s *SkillService) UpdateSkill(c fiber.Ctx) error {
	id := c.Params("id")
	if !validatePayloadID(id) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_id", "Formato de ID invalido", nil)
	}

	var payload experiencePayload
	if err := c.Bind().Body(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	sanitizePayload(&payload)

	existing, err := s.repo.GetByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiresponse.Error(c, fiber.StatusNotFound, "skill_not_found", "Capacidad no encontrada", nil)
		}
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_skills_failed", "No se pudo cargar capacidades", err.Error())
	}

	if !isSkillExperience(existing) {
		return apiresponse.Error(c, fiber.StatusNotFound, "skill_not_found", "Capacidad no encontrada", nil)
	}

	if payload.Title != "" {
		existing.Title = payload.Title
	}
	existing.Summary = payload.Summary
	existing.Body = payload.Body
	existing.ImageURLs = payload.ImageURLs
	existing.Tags = ensureSkillTag(payload.Tags)
	existing.Visibility = payload.Visibility
	existing.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := s.repo.Update(context.Background(), existing); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiresponse.Error(c, fiber.StatusNotFound, "skill_not_found", "Capacidad no encontrada", nil)
		}
		return apiresponse.Error(c, fiber.StatusInternalServerError, "save_skill_failed", "No se pudo actualizar la capacidad", err.Error())
	}

	return apiresponse.Success(c, existing)
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
func (s *SkillService) DeleteSkill(c fiber.Ctx) error {
	id := c.Params("id")
	if !validatePayloadID(id) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_id", "Formato de ID invalido", nil)
	}

	existing, err := s.repo.GetByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiresponse.Error(c, fiber.StatusNotFound, "skill_not_found", "Capacidad no encontrada", nil)
		}
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_skills_failed", "No se pudo cargar capacidades", err.Error())
	}

	if !isSkillExperience(existing) {
		return apiresponse.Error(c, fiber.StatusNotFound, "skill_not_found", "Capacidad no encontrada", nil)
	}

	if err := s.repo.Delete(context.Background(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiresponse.Error(c, fiber.StatusNotFound, "skill_not_found", "Capacidad no encontrada", nil)
		}
		return apiresponse.Error(c, fiber.StatusInternalServerError, "save_skill_failed", "No se pudo eliminar la capacidad", err.Error())
	}

	return apiresponse.Success(c, fiber.Map{"deleted": true, "id": id})
}
