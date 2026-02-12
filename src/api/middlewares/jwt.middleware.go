package jwtMiddleware

import (
	jwtManager "backend-yonathan/src/pkg/utils"
	"backend-yonathan/src/pkg/apiresponse"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func JWTProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")

		if authHeader == "" {
			return apiresponse.Error(c, fiber.StatusUnauthorized, "missing_token", "No se ha proporcionado un token", nil)
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return apiresponse.Error(c, fiber.StatusUnauthorized, "invalid_token_format", "Formato de token invalido. Use Bearer <token>", nil)
		}

		token, claims, err := jwtManager.VerificateToken(parts[1])
		if err != nil || !token.Valid {
			return apiresponse.Error(c, fiber.StatusUnauthorized, "invalid_token", "El token no es valido", nil)
		}

		c.Locals("userId", claims.UserID)
		c.Locals("username", claims.Username)
		return c.Next()
	}
}
