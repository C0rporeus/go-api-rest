package apiresponse

import "github.com/gofiber/fiber/v2"

func requestID(c *fiber.Ctx) string {
	value, ok := c.Locals("requestid").(string)
	if ok {
		return value
	}
	return ""
}

func Error(c *fiber.Ctx, status int, code, message string, details interface{}) error {
	payloadDetails := fiber.Map{
		"requestId": requestID(c),
	}
	if details != nil {
		payloadDetails["context"] = details
	}

	return c.Status(status).JSON(fiber.Map{
		"code":    code,
		"message": message,
		"details": payloadDetails,
	})
}

func Success(c *fiber.Ctx, payload interface{}) error {
	return c.JSON(payload)
}
