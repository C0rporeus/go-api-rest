package authServices

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestResolveDomainMissingParam(t *testing.T) {
	app := fiber.New()
	app.Get("/resolve", ResolveDomain)

	req := httptest.NewRequest(http.MethodGet, "/resolve", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestResolveDomainValid(t *testing.T) {
	app := fiber.New()
	app.Get("/resolve", ResolveDomain)

	req := httptest.NewRequest(http.MethodGet, "/resolve?domain=google.com", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["domain"] != "google.com" {
		t.Fatalf("expected domain google.com, got %v", body["domain"])
	}
}

func TestCheckPropagationMissingDomain(t *testing.T) {
	app := fiber.New()
	app.Get("/propagation", CheckPropagation)

	req := httptest.NewRequest(http.MethodGet, "/propagation", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestCheckPropagationInvalidType(t *testing.T) {
	app := fiber.New()
	app.Get("/propagation", CheckPropagation)

	req := httptest.NewRequest(http.MethodGet, "/propagation?domain=google.com&type=INVALID", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestCheckPropagationValidA(t *testing.T) {
	app := fiber.New()
	app.Get("/propagation", CheckPropagation)

	req := httptest.NewRequest(http.MethodGet, "/propagation?domain=google.com&type=A", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestGetMailRecordsMissingDomain(t *testing.T) {
	app := fiber.New()
	app.Get("/mail", GetMailRecords)

	req := httptest.NewRequest(http.MethodGet, "/mail", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestGetMailRecordsValid(t *testing.T) {
	app := fiber.New()
	app.Get("/mail", GetMailRecords)

	req := httptest.NewRequest(http.MethodGet, "/mail?domain=google.com", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestCheckBlacklistMissingIP(t *testing.T) {
	app := fiber.New()
	app.Get("/blacklist", CheckBlacklist)

	req := httptest.NewRequest(http.MethodGet, "/blacklist", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestCheckBlacklistInvalidIP(t *testing.T) {
	app := fiber.New()
	app.Get("/blacklist", CheckBlacklist)

	req := httptest.NewRequest(http.MethodGet, "/blacklist?ip=not-an-ip", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestCheckBlacklistValidIP(t *testing.T) {
	app := fiber.New()
	app.Get("/blacklist", CheckBlacklist)

	req := httptest.NewRequest(http.MethodGet, "/blacklist?ip=8.8.8.8", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}
