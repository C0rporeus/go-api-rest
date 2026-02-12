package jwtMiddleware

import (
	jwtManager "backend-yonathan/src/pkg/utils"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestJWTProtectedRejectsMissingToken(t *testing.T) {
	app := fiber.New()
	app.Get("/private", JWTProtected(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected app test error: %v", err)
	}
	if res.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}

func TestJWTProtectedAcceptsValidBearer(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	_ = os.Setenv("JWT_SECRET", "test-secret")

	token, err := jwtManager.GenerateToken("u-123", "tester")
	if err != nil {
		t.Fatalf("unexpected token generation error: %v", err)
	}

	app := fiber.New()
	app.Get("/private", JWTProtected(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected app test error: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}
