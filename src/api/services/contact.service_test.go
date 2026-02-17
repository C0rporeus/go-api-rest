package services

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestSubmitContactSuccess(t *testing.T) {
	var captured contactPayload
	original := logContactMessage
	logContactMessage = func(p contactPayload) { captured = p }
	defer func() { logContactMessage = original }()

	app := fiber.New()
	app.Post("/contact", SubmitContact)

	body, _ := json.Marshal(map[string]string{
		"name":    "Maria Lopez",
		"email":   "maria@example.com",
		"message": "Hola, me interesa tu servicio.",
	})
	req := httptest.NewRequest(http.MethodPost, "/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if result["sent"] != true {
		t.Fatal("expected sent=true")
	}
	if captured.Name != "Maria Lopez" {
		t.Fatalf("expected name 'Maria Lopez', got %q", captured.Name)
	}
}

func TestSubmitContactMissingName(t *testing.T) {
	app := fiber.New()
	app.Post("/contact", SubmitContact)

	body, _ := json.Marshal(map[string]string{
		"name":    "AB",
		"email":   "test@test.com",
		"message": "hola",
	})
	req := httptest.NewRequest(http.MethodPost, "/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestSubmitContactInvalidEmail(t *testing.T) {
	app := fiber.New()
	app.Post("/contact", SubmitContact)

	body, _ := json.Marshal(map[string]string{
		"name":    "Carlos Test",
		"email":   "not-an-email",
		"message": "hola",
	})
	req := httptest.NewRequest(http.MethodPost, "/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestSubmitContactEmptyMessage(t *testing.T) {
	app := fiber.New()
	app.Post("/contact", SubmitContact)

	body, _ := json.Marshal(map[string]string{
		"name":    "Carlos Test",
		"email":   "carlos@test.com",
		"message": "",
	})
	req := httptest.NewRequest(http.MethodPost, "/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestSubmitContactMessageTooLong(t *testing.T) {
	app := fiber.New()
	app.Post("/contact", SubmitContact)

	body, _ := json.Marshal(map[string]string{
		"name":    "Carlos Test",
		"email":   "carlos@test.com",
		"message": strings.Repeat("a", 501),
	})
	req := httptest.NewRequest(http.MethodPost, "/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}
