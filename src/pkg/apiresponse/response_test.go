package apiresponse

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestErrorResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/err", func(c *fiber.Ctx) error {
		c.Locals("requestid", "req-1")
		return Error(c, fiber.StatusBadRequest, "bad_request", "invalid", "detail")
	})

	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 got %d", res.StatusCode)
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if payload["code"] != "bad_request" {
		t.Fatalf("expected code bad_request")
	}
}

func TestSuccessResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/ok", func(c *fiber.Ctx) error {
		return Success(c, fiber.Map{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 got %d", res.StatusCode)
	}
}
