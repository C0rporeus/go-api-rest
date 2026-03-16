package jwtMiddleware

import (
	jwtManager "backend-yonathan/src/pkg/utils"
	"backend-yonathan/src/pkg/apiresponse"
	"strings"

	"github.com/gofiber/fiber/v3"
)

func JWTProtected() fiber.Handler {
	return func(c fiber.Ctx) error {
		tokenString := c.Cookies("portfolio_auth_token")
		if tokenString == "" {
			authHeader := c.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					tokenString = parts[1]
				}
			}
		}

		if tokenString == "" {
			return apiresponse.Error(c, fiber.StatusUnauthorized, "missing_token", "No se ha proporcionado un token", nil)
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
