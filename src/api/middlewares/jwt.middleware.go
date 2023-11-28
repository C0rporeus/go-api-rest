package jwtMiddleware

import (
	jwtManager "backend-yonathan/src/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

func JWTProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := c.Get("Authorization")

		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No se ha proporcionado un token",
			})
		}

		token, err := jwtManager.VerificateToken(tokenString)
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "El token no es válido",
			})
		}
		return c.Next()
	}
}
