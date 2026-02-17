// Package apiresponse provides standardized API response helpers.
//
// # Response contract
//
// Success responses return the payload directly (HTTP 200):
//
//	{ ...payload }
//
// Error responses wrap in a structured envelope:
//
//	{ "code": "<error_code>", "message": "<human_readable>", "details": { "requestId": "...", "context": ... } }
//
// The frontend http-client expects this asymmetry:
// success bodies are consumed as-is, while errors are parsed from the envelope.
package apiresponse

import "github.com/gofiber/fiber/v3"

func requestID(c fiber.Ctx) string {
	value, ok := c.Locals("requestid").(string)
	if ok {
		return value
	}
	return ""
}

// Error sends a structured error response with code, message, and traceability.
func Error(c fiber.Ctx, status int, code, message string, details interface{}) error {
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

// Success sends the payload directly as JSON with HTTP 200.
// The payload is NOT wrapped in an envelope — this is by design.
// See package-level documentation for the full response contract.
func Success(c fiber.Ctx, payload interface{}) error {
	return c.JSON(payload)
}
