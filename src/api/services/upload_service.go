package services

import (
	"context"
	"path/filepath"
	"strings"

	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// uploadToBucketFunc is injectable for tests. Default: uploadToGCS.
var uploadToBucketFunc = uploadToGCS

// UploadImage godoc
// @Summary      Subir imagen
// @Description  Recibe un archivo de imagen (multipart "file"), lo sube al bucket GCP y devuelve la URL pública. Requiere JWT.
// @Tags         Upload
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "Imagen (JPEG, PNG, GIF, WebP; máx 5 MB)"
// @Success      200   {object}  map[string]string  "url"
// @Failure      400   {object}  map[string]interface{}
// @Failure      413   {object}  map[string]interface{}
// @Failure      503   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /api/private/upload-image [post]
func UploadImage(c fiber.Ctx) error {
	bucket := constants.GCSBucketName()
	if bucket == "" {
		return apiresponse.Error(c, fiber.StatusServiceUnavailable, "upload_not_configured",
			"Subida de imágenes no configurada (GCS_BUCKET_NAME)", nil)
	}

	file, err := c.FormFile("file")
	if err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_file", "Falta el archivo en el campo 'file'", err.Error())
	}

	if file.Size > constants.MaxImageUploadBytes {
		return apiresponse.Error(c, fiber.StatusRequestEntityTooLarge, "file_too_large",
			"El archivo supera el límite de 5 MB", nil)
	}

	contentType := file.Header.Get("Content-Type")
	if !isAllowedImageContentType(contentType) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_content_type",
			"Solo se permiten imágenes JPEG, PNG, GIF o WebP", nil)
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Filename), "."))
	if ext == "" {
		ext = "jpg"
	}
	safeExt := mapExt(ext)
	objectPath := constants.StorageImagePrefix + "/" + uuid.New().String() + "." + safeExt

	opened, err := file.Open()
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "read_file_failed", "No se pudo leer el archivo", err.Error())
	}
	defer opened.Close()

	ctx := context.Background()
	publicURL, err := uploadToBucketFunc(ctx, bucket, objectPath, contentType, opened)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "upload_failed", "No se pudo subir la imagen", err.Error())
	}

	return apiresponse.Success(c, fiber.Map{"url": publicURL})
}

func isAllowedImageContentType(ct string) bool {
	ct = strings.TrimSpace(strings.ToLower(ct))
	for _, allowed := range constants.AllowedImageContentTypes {
		if ct == allowed {
			return true
		}
	}
	return false
}

func mapExt(ext string) string {
	switch ext {
	case "jpeg", "jpg", "png", "gif", "webp":
		return ext
	default:
		return "jpg"
	}
}
