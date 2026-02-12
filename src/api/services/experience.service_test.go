package authServices

import (
	"bytes"
	"encoding/json"
	"io"
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

func TestUpdateDeleteAndListAllExperiences(t *testing.T) {
	t.Setenv("PORTFOLIO_DATA_DIR", filepath.Join(t.TempDir(), "data"))
	app := fiber.New()
	app.Post("/private/experiences", CreateExperience)
	app.Put("/private/experiences/:id", UpdateExperience)
	app.Delete("/private/experiences/:id", DeleteExperience)
	app.Get("/private/experiences", ListAllExperiences)

	createBody, _ := json.Marshal(map[string]any{
		"title":      "Proyecto inicial",
		"summary":    "Summary",
		"body":       "Body",
		"tags":       []string{"go"},
		"visibility": "public",
	})
	createReq := httptest.NewRequest(http.MethodPost, "/private/experiences", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRes, err := app.Test(createReq, -1)
	if err != nil || createRes.StatusCode != fiber.StatusOK {
		t.Fatalf("create failed: %v status=%d", err, createRes.StatusCode)
	}

	raw, _ := io.ReadAll(createRes.Body)
	var created map[string]any
	if err := json.Unmarshal(raw, &created); err != nil {
		t.Fatalf("unmarshal created failed: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("expected id in create response")
	}

	updateBody, _ := json.Marshal(map[string]any{
		"title":      "Proyecto actualizado",
		"summary":    "Summary 2",
		"body":       "Body 2",
		"tags":       []string{"go", "api"},
		"visibility": "private",
	})
	updateReq := httptest.NewRequest(http.MethodPut, "/private/experiences/"+id, bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRes, err := app.Test(updateReq, -1)
	if err != nil || updateRes.StatusCode != fiber.StatusOK {
		t.Fatalf("update failed: %v status=%d", err, updateRes.StatusCode)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/private/experiences", nil)
	listRes, err := app.Test(listReq, -1)
	if err != nil || listRes.StatusCode != fiber.StatusOK {
		t.Fatalf("list all failed: %v status=%d", err, listRes.StatusCode)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/private/experiences/"+id, nil)
	deleteRes, err := app.Test(deleteReq, -1)
	if err != nil || deleteRes.StatusCode != fiber.StatusOK {
		t.Fatalf("delete failed: %v status=%d", err, deleteRes.StatusCode)
	}
}
