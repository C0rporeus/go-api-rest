package services

import (
	userModel "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ListPublicExperiences godoc
// @Summary      Listar experiencias publicas
// @Description  Devuelve experiencias con visibility=public. Soporta ETag/If-None-Match.
// @Tags         Experiences
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "items"
// @Success      304  "Not Modified"
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/experiences [get]
func ListPublicExperiences(c fiber.Ctx) error {
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
func ListAllExperiences(c fiber.Ctx) error {
	experienceStoreLock.RLock()
	defer experienceStoreLock.RUnlock()

	experiences, err := loadExperiences()
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_experiences_failed", "No se pudo cargar experiencias", err.Error())
	}
	return apiresponse.Success(c, fiber.Map{"items": experiences})
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
func CreateExperience(c fiber.Ctx) error {
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
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_experiences_failed", "No se pudo cargar experiencias", err.Error())
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item := userModel.Experience{
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

	experiences = append(experiences, item)
	if err := saveExperiences(experiences); err != nil {
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
func UpdateExperience(c fiber.Ctx) error {
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
		return apiresponse.Error(c, fiber.StatusInternalServerError, "load_experiences_failed", "No se pudo cargar experiencias", err.Error())
	}

	for i, item := range experiences {
		if item.ID == id {
			if payload.Title != "" {
				item.Title = payload.Title
			}
			item.Summary = payload.Summary
			item.Body = payload.Body
			item.ImageURLs = payload.ImageURLs
			item.Tags = payload.Tags
			item.Visibility = payload.Visibility
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
func DeleteExperience(c fiber.Ctx) error {
	id := c.Params("id")
	if !validatePayloadID(id) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_id", "Formato de ID invalido", nil)
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
