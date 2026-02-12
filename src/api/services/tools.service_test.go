package authServices

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestEncodeBase64(t *testing.T) {
	app := fiber.New()
	app.Post("/encode", EncodeBase64)

	body, _ := json.Marshal(map[string]string{"value": "hola"})
	req := httptest.NewRequest(http.MethodPost, "/encode", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestDecodeBase64InvalidInput(t *testing.T) {
	app := fiber.New()
	app.Post("/decode", DecodeBase64)

	body, _ := json.Marshal(map[string]string{"value": "%%%invalid%%%"})
	req := httptest.NewRequest(http.MethodPost, "/decode", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestGenerateUUIDv4(t *testing.T) {
	app := fiber.New()
	app.Get("/uuid", GenerateUUIDv4)

	req := httptest.NewRequest(http.MethodGet, "/uuid", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}
