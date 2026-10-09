package jwtMiddleware

import (
	"backend-yonathan/src/pkg/apiresponse"
	jwtManager "backend-yonathan/src/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

func JWTProtected() fiber.Handler {
	return func(c fiber.Ctx) error {
		tokenString := c.Cookies("portfolio_auth_token")

		if tokenString == "" {
			return apiresponse.Error(c, fiber.StatusUnauthorized, "missing_session", "No se ha proporcionado una sesion", nil)
		}

		token, claims, err := jwtManager.VerifyToken(tokenString)
		if err != nil || !token.Valid {
			return apiresponse.Error(c, fiber.StatusUnauthorized, "invalid_token", "El token no es valido", nil)
		}

		c.Locals("userId", claims.UserID)
		c.Locals("username", claims.Username)
		return c.Next()
	}
}
