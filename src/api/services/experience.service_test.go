package services

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/repository/memory"

	"github.com/gofiber/fiber/v3"
)

func TestCreateAndListPublicExperiences(t *testing.T) {
	repo := memory.NewExperienceRepository()
	svc := NewExperienceService(repo)

	app := fiber.New()
	app.Post("/private/experiences", svc.CreateExperience)
	app.Get("/experiences", svc.ListPublicExperiences)

	body, _ := json.Marshal(map[string]any{
		"title":      "Proyecto API",
		"summary":    "Backend para portafolio",
		"body":       "Detalle del proyecto",
		"tags":       []string{"go", "api"},
		"visibility": constants.VisibilityPublic,
	})

	createReq := httptest.NewRequest(http.MethodPost, "/private/experiences", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRes, err := app.Test(createReq)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if createRes.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 on create, got %d", createRes.StatusCode)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/experiences", nil)
	listRes, err := app.Test(listReq)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if listRes.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 on list, got %d", listRes.StatusCode)
	}
}

func TestCreateExperienceRejectsMissingTitle(t *testing.T) {
	repo := memory.NewExperienceRepository()
	svc := NewExperienceService(repo)

	app := fiber.New()
	app.Post("/private/experiences", svc.CreateExperience)

	body, _ := json.Marshal(map[string]any{
		"summary": "Sin titulo",
	})
	req := httptest.NewRequest(http.MethodPost, "/private/experiences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 on missing title, got %d", res.StatusCode)
	}
}

func TestUpdateDeleteAndListAllExperiences(t *testing.T) {
	repo := memory.NewExperienceRepository()
	svc := NewExperienceService(repo)

	app := fiber.New()
	app.Post("/private/experiences", svc.CreateExperience)
	app.Put("/private/experiences/:id", svc.UpdateExperience)
	app.Delete("/private/experiences/:id", svc.DeleteExperience)
	app.Get("/private/experiences", svc.ListAllExperiences)

	createBody, _ := json.Marshal(map[string]any{
		"title":      "Proyecto inicial",
		"summary":    "Summary",
		"body":       "Body",
		"tags":       []string{"go"},
		"visibility": constants.VisibilityPublic,
	})
	createReq := httptest.NewRequest(http.MethodPost, "/private/experiences", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRes, err := app.Test(createReq)
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
		"visibility": constants.VisibilityPrivate,
	})
	updateReq := httptest.NewRequest(http.MethodPut, "/private/experiences/"+id, bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRes, err := app.Test(updateReq)
	if err != nil || updateRes.StatusCode != fiber.StatusOK {
		t.Fatalf("update failed: %v status=%d", err, updateRes.StatusCode)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/private/experiences", nil)
	listRes, err := app.Test(listReq)
	if err != nil || listRes.StatusCode != fiber.StatusOK {
		t.Fatalf("list all failed: %v status=%d", err, listRes.StatusCode)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/private/experiences/"+id, nil)
	deleteRes, err := app.Test(deleteReq)
	if err != nil || deleteRes.StatusCode != fiber.StatusOK {
		t.Fatalf("delete failed: %v status=%d", err, deleteRes.StatusCode)
	}
}

func TestUpdateExperienceNotFound(t *testing.T) {
	repo := memory.NewExperienceRepository()
	svc := NewExperienceService(repo)

	app := fiber.New()
	app.Put("/private/experiences/:id", svc.UpdateExperience)

	body, _ := json.Marshal(map[string]any{"title": "X"})
	req := httptest.NewRequest(http.MethodPut, "/private/experiences/00000000-0000-0000-0000-000000000099", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}

func TestDeleteExperienceNotFound(t *testing.T) {
	repo := memory.NewExperienceRepository()
	svc := NewExperienceService(repo)

	app := fiber.New()
	app.Delete("/private/experiences/:id", svc.DeleteExperience)

	req := httptest.NewRequest(http.MethodDelete, "/private/experiences/00000000-0000-0000-0000-000000000099", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}

func TestListPublicExperiencesFiltersPrivate(t *testing.T) {
	repo := memory.NewExperienceRepository()
	svc := NewExperienceService(repo)

	app := fiber.New()
	app.Post("/private/experiences", svc.CreateExperience)
	app.Get("/experiences", svc.ListPublicExperiences)

	body, _ := json.Marshal(map[string]any{
		"title":      "Private Exp",
		"visibility": constants.VisibilityPrivate,
	})
	createReq := httptest.NewRequest(http.MethodPost, "/private/experiences", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRes, err := app.Test(createReq)
	if err != nil || createRes.StatusCode != fiber.StatusOK {
		t.Fatalf("create failed: err=%v status=%d", err, createRes.StatusCode)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/experiences", nil)
	listRes, err := app.Test(listReq)
	if err != nil || listRes.StatusCode != fiber.StatusOK {
		t.Fatalf("list failed: err=%v status=%d", err, listRes.StatusCode)
	}

	raw, _ := io.ReadAll(listRes.Body)
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	items, ok := result["items"].([]any)
	if !ok || len(items) != 0 {
		t.Fatalf("expected 0 public experiences, got %v", result["items"])
	}
}

func TestListPublicExperiencesUsesStableETagForSignedImageURLs(t *testing.T) {
	t.Setenv("GCS_BUCKET_NAME", "portfolio-images")

	repo := memory.NewExperienceRepository()
	item := models.Experience{
		ID:         "article-1",
		Title:      "Artículo de prueba",
		Summary:    "Resumen",
		Body:       `<p><img src="https://storage.googleapis.com/portfolio-images/article.png?X-Goog-Signature=old"></p>`,
		ImageURLs:  []string{"https://storage.googleapis.com/portfolio-images/cover.png?X-Goog-Signature=old"},
		Visibility: constants.VisibilityPublic,
		CreatedAt:  "2026-01-01T00:00:00Z",
		UpdatedAt:  "2026-01-01T00:00:00Z",
	}
	if err := repo.Create(context.Background(), item); err != nil {
		t.Fatalf("create experience: %v", err)
	}

	originalSign := signURLFunc
	signCalls := 0
	signURLFunc = func(_ context.Context, _, object string, _ time.Duration) (string, error) {
		signCalls++
		return "https://storage.googleapis.com/portfolio-images/" + object + "?X-Goog-Signature=" + strconv.Itoa(signCalls), nil
	}
	t.Cleanup(func() { signURLFunc = originalSign })

	app := fiber.New()
	app.Get("/experiences", NewExperienceService(repo).ListPublicExperiences)
	firstResponse, err := app.Test(httptest.NewRequest(http.MethodGet, "/experiences", nil))
	if err != nil {
		t.Fatalf("first request: %v", err)
	}
	if firstResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("expected first response 200, got %d", firstResponse.StatusCode)
	}
	etag := firstResponse.Header.Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag on first response")
	}
	signCallsAfterFirstResponse := signCalls

	secondRequest := httptest.NewRequest(http.MethodGet, "/experiences", nil)
	secondRequest.Header.Set("If-None-Match", etag)
	secondResponse, err := app.Test(secondRequest)
	if err != nil {
		t.Fatalf("revalidation request: %v", err)
	}
	if secondResponse.StatusCode != fiber.StatusNotModified {
		t.Fatalf("expected revalidation response 304, got %d", secondResponse.StatusCode)
	}
	if secondResponse.Header.Get("ETag") != etag {
		t.Fatalf("expected stable ETag %q, got %q", etag, secondResponse.Header.Get("ETag"))
	}
	if signCalls != signCallsAfterFirstResponse {
		t.Fatalf("expected 304 revalidation to skip URL signing; signing calls changed from %d to %d", signCallsAfterFirstResponse, signCalls)
	}
}
