package services

import (
	"bytes"
	"context"
	"image/png"
	"io"
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
// @Success      200   {object}  map[string]string  "url (canónica, para persistir) y previewUrl (firmada, temporal)"
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
	original, err := io.ReadAll(opened)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "read_file_failed", "No se pudo leer la imagen", err.Error())
	}
	optimized := optimizePNG(contentType, original)

	ctx := context.Background()
	publicURL, err := uploadToBucketFunc(ctx, bucket, objectPath, contentType, bytes.NewReader(optimized))
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "upload_failed", "No se pudo subir la imagen", err.Error())
	}

	// The bucket is private: "url" is the canonical URL to persist, while
	// "previewUrl" is a short-lived signed URL so the admin can render it now.
	previewURL := signSingleURL(ctx, publicURL, constants.GCSURLPrefix(), constants.SignedURLExpiry())
	return apiresponse.Success(c, fiber.Map{"url": publicURL, "previewUrl": previewURL})
}

// optimizePNG recompresses valid, reasonably sized PNGs losslessly. Other
// formats and malformed PNG payloads are passed through unchanged.
func optimizePNG(contentType string, original []byte) []byte {
	if strings.ToLower(strings.TrimSpace(contentType)) != "image/png" {
		return original
	}
	config, err := png.DecodeConfig(bytes.NewReader(original))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 25_000_000 {
		return original
	}
	image, err := png.Decode(bytes.NewReader(original))
	if err != nil {
		return original
	}
	var compressed bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&compressed, image); err != nil || compressed.Len() >= len(original) {
		return original
	}
	return compressed.Bytes()
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
