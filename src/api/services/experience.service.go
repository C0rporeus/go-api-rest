package services

import (
	"context"
	"errors"
	"time"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/repository"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ExperienceService handles experience CRUD business logic.
type ExperienceService struct {
	repo repository.ExperienceRepository
}

// NewExperienceService creates an ExperienceService backed by the given ExperienceRepository.
func NewExperienceService(repo repository.ExperienceRepository) *ExperienceService {
	return &ExperienceService{repo: repo}
}

// ListPublicExperiences godoc
// @Summary      Listar experiencias publicas
// @Description  Devuelve experiencias con visibility=public. Soporta ETag/If-None-Match.
// @Tags         Experiences
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "items"
// @Success      304  "Not Modified"
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/experiences [get]
func (s *ExperienceService) ListPublicExperiences(c fiber.Ctx) error {
	all, err := s.repo.List(context.Background())
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_experiences_failed", "No se pudo cargar experiencias", err.Error())
	}

	public := make([]models.Experience, 0, len(all))
	for _, item := range all {
		if item.Visibility == constants.VisibilityPublic {
			public = append(public, item)
		}
	}

	etag := buildCollectionETag(public)
	setPublicCollectionCacheHeaders(c, etag)
	if matchesIfNoneMatchHeader(c.Get("If-None-Match"), etag) {
		return c.SendStatus(fiber.StatusNotModified)
	}

	return apiresponse.Success(c, fiber.Map{"items": public})
}

// ListAllExperiences godoc
// @Summary      Listar todas las experiencias
// @Description  Devuelve todas las experiencias (publicas y privadas). Requiere JWT.
// @Tags         Experiences
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}  "items"
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/private/experiences [get]
func (s *ExperienceService) ListAllExperiences(c fiber.Ctx) error {
	all, err := s.repo.List(context.Background())
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_experiences_failed", "No se pudo cargar experiencias", err.Error())
	}
	return apiresponse.Success(c, fiber.Map{"items": all})
}

// CreateExperience godoc
// @Summary      Crear experiencia
// @Description  Crea una nueva experiencia. Requiere JWT.
// @Tags         Experiences
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        experience  body  object{title=string,summary=string,body=string,imageUrls=[]string,tags=[]string,visibility=string}  true  "Datos"
// @Success      200  {object}  userModel.Experience
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/private/experiences [post]
func (s *ExperienceService) CreateExperience(c fiber.Ctx) error {
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
		Tags:       payload.Tags,
		Visibility: payload.Visibility,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.Create(context.Background(), item); err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "save_experience_failed", "No se pudo guardar la experiencia", err.Error())
	}

	return apiresponse.Success(c, item)
}

// UpdateExperience godoc
// @Summary      Actualizar experiencia
// @Description  Actualiza una experiencia por ID. Requiere JWT.
// @Tags         Experiences
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id          path  string  true  "ID de la experiencia"
// @Param        experience  body  object{title=string,summary=string,body=string,imageUrls=[]string,tags=[]string,visibility=string}  true  "Datos"
// @Success      200  {object}  userModel.Experience
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/private/experiences/{id} [put]
func (s *ExperienceService) UpdateExperience(c fiber.Ctx) error {
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
			return apiresponse.Error(c, fiber.StatusNotFound, "experience_not_found", "Experiencia no encontrada", nil)
		}
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_experiences_failed", "No se pudo cargar experiencias", err.Error())
	}

	if payload.Title != "" {
		existing.Title = payload.Title
	}
	existing.Summary = payload.Summary
	existing.Body = payload.Body
	existing.ImageURLs = payload.ImageURLs
	existing.Tags = payload.Tags
	existing.Visibility = payload.Visibility
	existing.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := s.repo.Update(context.Background(), existing); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiresponse.Error(c, fiber.StatusNotFound, "experience_not_found", "Experiencia no encontrada", nil)
		}
		return apiresponse.Error(c, fiber.StatusInternalServerError, "save_experience_failed", "No se pudo actualizar la experiencia", err.Error())
	}

	return apiresponse.Success(c, existing)
}

// DeleteExperience godoc
// @Summary      Eliminar experiencia
// @Description  Elimina una experiencia por ID. Requiere JWT.
// @Tags         Experiences
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "ID de la experiencia"
// @Success      200  {object}  map[string]interface{}  "deleted, id"
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/private/experiences/{id} [delete]
func (s *ExperienceService) DeleteExperience(c fiber.Ctx) error {
	id := c.Params("id")
	if !validatePayloadID(id) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_id", "Formato de ID invalido", nil)
	}

	if err := s.repo.Delete(context.Background(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiresponse.Error(c, fiber.StatusNotFound, "experience_not_found", "Experiencia no encontrada", nil)
		}
		return apiresponse.Error(c, fiber.StatusInternalServerError, "save_experience_failed", "No se pudo eliminar la experiencia", err.Error())
	}

	return apiresponse.Success(c, fiber.Map{"deleted": true, "id": id})
}
