package services

import (
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/pkg/sanitizer"
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type contactPayload struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

// SubmitContact godoc
// @Summary      Enviar formulario de contacto
// @Description  Recibe y valida un formulario de contacto (nombre, email, mensaje)
// @Tags         Contact
// @Accept       json
// @Produce      json
// @Param        payload  body  object{name=string,email=string,message=string}  true  "Datos de contacto"
// @Success      200  {object}  map[string]bool  "sent"
// @Failure      400  {object}  map[string]interface{}
// @Router       /api/contact [post]
func SubmitContact(c fiber.Ctx) error {
	var payload contactPayload
	if err := c.Bind().Body(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	payload.Email = strings.TrimSpace(strings.ToLower(payload.Email))

	if sanitizer.ContainsDangerousPatterns(payload.Name) || sanitizer.ContainsDangerousPatterns(payload.Message) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "dangerous_input",
			"El contenido contiene patrones no permitidos", nil)
	}

	payload.Name = sanitizer.SanitizePlainText(payload.Name, constants.MaxContactName)
	if len(payload.Name) < constants.MinContactName {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_name",
			"El nombre debe tener al menos 2 caracteres", nil)
	}
	if !sanitizer.IsValidEmail(payload.Email) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_email",
			"Introduce un correo electronico valido", nil)
	}
	if len(strings.TrimSpace(payload.Message)) == 0 {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_message",
			"El mensaje es requerido", nil)
	}
	if len(payload.Message) > constants.MaxMessageLength {
		return apiresponse.Error(c, fiber.StatusBadRequest, "message_too_long",
			"El mensaje no puede tener mas de 500 caracteres", nil)
	}
	payload.Message = sanitizer.SanitizePlainText(payload.Message, constants.MaxMessageLength)

	logContactMessage(payload)

	return apiresponse.Success(c, fiber.Map{
		"sent": true,
	})
}

// logContactMessage is the default contact handler — logs the message.
// Replace with SES/SNS integration when ready.
var logContactMessage = func(p contactPayload) {
	log.Printf("[CONTACT] from=%q email=%q message=%q", p.Name, p.Email, p.Message)
}
