package authServices

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestLoginRejectsInvalidPayload(t *testing.T) {
	app := fiber.New()
	app.Post("/login", func(c *fiber.Ctx) error {
		return Login(c, nil)
	})

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("{invalid-json"))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestRegisterRejectsInvalidPayload(t *testing.T) {
	app := fiber.New()
	app.Post("/register", func(c *fiber.Ctx) error {
		return Register(c, nil)
	})

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader("{invalid-json"))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}
