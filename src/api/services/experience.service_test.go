package authServices

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestCreateAndListPublicExperiences(t *testing.T) {
	t.Setenv("PORTFOLIO_DATA_DIR", filepath.Join(t.TempDir(), "data"))

	app := fiber.New()
	app.Post("/private/experiences", CreateExperience)
	app.Get("/experiences", ListPublicExperiences)

	body, _ := json.Marshal(map[string]any{
		"title":      "Proyecto API",
		"summary":    "Backend para portafolio",
		"body":       "Detalle del proyecto",
		"tags":       []string{"go", "api"},
		"visibility": "public",
	})

	createReq := httptest.NewRequest(http.MethodPost, "/private/experiences", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRes, err := app.Test(createReq, -1)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if createRes.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 on create, got %d", createRes.StatusCode)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/experiences", nil)
	listRes, err := app.Test(listReq, -1)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if listRes.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 on list, got %d", listRes.StatusCode)
	}
}

func TestCreateExperienceRejectsMissingTitle(t *testing.T) {
	t.Setenv("PORTFOLIO_DATA_DIR", filepath.Join(t.TempDir(), "data"))
	app := fiber.New()
	app.Post("/private/experiences", CreateExperience)

	body, _ := json.Marshal(map[string]any{
		"summary": "Sin titulo",
	})
	req := httptest.NewRequest(http.MethodPost, "/private/experiences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 on missing title, got %d", res.StatusCode)
	}
}
