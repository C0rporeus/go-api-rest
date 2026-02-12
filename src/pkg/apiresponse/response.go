package apiresponse

import "github.com/gofiber/fiber/v2"

func Error(c *fiber.Ctx, status int, code, message string, details interface{}) error {
	return c.Status(status).JSON(fiber.Map{
		"code":    code,
		"message": message,
		"details": details,
	})
}

func Success(c *fiber.Ctx, payload interface{}) error {
	return c.JSON(payload)
}
