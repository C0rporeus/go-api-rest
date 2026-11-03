package services

import (
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"encoding/base64"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type base64Request struct {
	Value string `json:"value"`
	// Variant is "std" (RFC 4648 §4, padded; default) or "url" (RFC 4648 §5,
	// unpadded, as used by JWT). Decoding "url" also accepts padded input.
	Variant string `json:"variant"`
}

const (
	base64VariantStd = "std"
	base64VariantURL = "url"
)

// normalizeBase64Variant validates the variant, defaulting to std.
func normalizeBase64Variant(variant string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(variant)) {
	case "", base64VariantStd:
		return base64VariantStd, true
	case base64VariantURL:
		return base64VariantURL, true
	default:
		return "", false
	}
}

// EncodeBase64 godoc
// @Summary      Codificar Base64
// @Description  Codifica un string en Base64
// @Tags         Tools
// @Accept       json
// @Produce      json
// @Param        payload  body  object{value=string}  true  "Texto a codificar"
// @Success      200  {object}  map[string]string  "input, encoded"
// @Failure      400  {object}  map[string]interface{}
// @Router       /api/tools/base64/encode [post]
func EncodeBase64(c fiber.Ctx) error {
	var payload base64Request
	if err := c.Bind().Body(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	if len(payload.Value) > constants.MaxBase64InputSize {
		return apiresponse.Error(c, fiber.StatusBadRequest, "input_too_large",
			"El texto excede el tamano maximo permitido", nil)
	}

	variant, ok := normalizeBase64Variant(payload.Variant)
	if !ok {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_variant", "La variante debe ser 'std' o 'url'", nil)
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(payload.Value))
	if variant == base64VariantURL {
		encoded = base64.RawURLEncoding.EncodeToString([]byte(payload.Value))
	}

	return apiresponse.Success(c, fiber.Map{
		"input":   payload.Value,
		"encoded": encoded,
		"variant": variant,
	})
}

// DecodeBase64 godoc
// @Summary      Decodificar Base64
// @Description  Decodifica un string Base64
// @Tags         Tools
// @Accept       json
// @Produce      json
// @Param        payload  body  object{value=string}  true  "Texto en Base64"
// @Success      200  {object}  map[string]string  "input, decoded"
// @Failure      400  {object}  map[string]interface{}
// @Router       /api/tools/base64/decode [post]
func DecodeBase64(c fiber.Ctx) error {
	var payload base64Request
	if err := c.Bind().Body(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	if len(payload.Value) > constants.MaxBase64InputSize {
		return apiresponse.Error(c, fiber.StatusBadRequest, "input_too_large",
			"El texto excede el tamano maximo permitido", nil)
	}

	variant, ok := normalizeBase64Variant(payload.Variant)
	if !ok {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_variant", "La variante debe ser 'std' o 'url'", nil)
	}
	var decoded []byte
	var err error
	if variant == base64VariantURL {
		decoded, err = base64.RawURLEncoding.DecodeString(strings.TrimRight(payload.Value, "="))
	} else {
		decoded, err = base64.StdEncoding.DecodeString(payload.Value)
	}
	if err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_base64", "El valor no es base64 valido", err.Error())
	}

	return apiresponse.Success(c, fiber.Map{
		"input":   payload.Value,
		"decoded": string(decoded),
		"variant": variant,
	})
}

// GenerateUUIDv4 godoc
// @Summary      Generar UUID v4
// @Description  Genera un UUID v4 aleatorio
// @Tags         Tools
// @Produce      json
// @Success      200  {object}  map[string]string  "uuid"
// @Router       /api/tools/uuid/v4 [get]
func GenerateUUIDv4(c fiber.Ctx) error {
	return apiresponse.Success(c, fiber.Map{
		"uuid": uuid.NewString(),
	})
}
