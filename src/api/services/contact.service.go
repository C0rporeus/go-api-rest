package services

import (
	"backend-yonathan/src/pkg/apiresponse"
	"log"
	"net/mail"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type contactPayload struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

const (
	contactNameMinLen = 5
	contactMsgMaxLen  = 500
)

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
func SubmitContact(c *fiber.Ctx) error {
	var payload contactPayload
	if err := c.BodyParser(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	payload.Name = strings.TrimSpace(payload.Name)
	payload.Email = strings.TrimSpace(payload.Email)
	payload.Message = strings.TrimSpace(payload.Message)

	if len(payload.Name) < contactNameMinLen {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_name",
			"El nombre debe tener al menos 5 caracteres", nil)
	}
	if _, err := mail.ParseAddress(payload.Email); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_email",
			"Introduce un correo electronico valido", nil)
	}
	if payload.Message == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_message",
			"El mensaje es requerido", nil)
	}
	if len(payload.Message) > contactMsgMaxLen {
		return apiresponse.Error(c, fiber.StatusBadRequest, "message_too_long",
			"El mensaje no puede tener mas de 500 caracteres", nil)
	}

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
