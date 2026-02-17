package services

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

func TestCreateAndListPublicSkills(t *testing.T) {
	t.Setenv("PORTFOLIO_DATA_DIR", filepath.Join(t.TempDir(), "data"))

	app := fiber.New()
	app.Post("/private/skills", CreateSkill)
	app.Get("/skills", ListPublicSkills)

	body, _ := json.Marshal(map[string]any{
		"title":      "Go Avanzado",
		"summary":    "Capacidad en Go",
		"body":       "Detalle de la habilidad",
		"tags":       []string{"skill", "go"},
		"visibility": "public",
	})

	createReq := httptest.NewRequest(http.MethodPost, "/private/skills", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRes, err := app.Test(createReq, -1)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if createRes.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 on create, got %d", createRes.StatusCode)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/skills", nil)
	listRes, err := app.Test(listReq, -1)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if listRes.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 on list, got %d", listRes.StatusCode)
	}

	raw, _ := io.ReadAll(listRes.Body)
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	items, ok := result["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected 1 public skill, got %v", result["items"])
	}
}

func TestCreateSkillRejectsMissingTitle(t *testing.T) {
	t.Setenv("PORTFOLIO_DATA_DIR", filepath.Join(t.TempDir(), "data"))
	app := fiber.New()
	app.Post("/private/skills", CreateSkill)

	body, _ := json.Marshal(map[string]any{
		"summary": "Sin titulo",
		"tags":    []string{"skill"},
	})
	req := httptest.NewRequest(http.MethodPost, "/private/skills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 on missing title, got %d", res.StatusCode)
	}
}

func TestCreateSkillEnsuresSkillTag(t *testing.T) {
	t.Setenv("PORTFOLIO_DATA_DIR", filepath.Join(t.TempDir(), "data"))
	app := fiber.New()
	app.Post("/private/skills", CreateSkill)

	body, _ := json.Marshal(map[string]any{
		"title":      "TypeScript",
		"tags":       []string{"frontend"},
		"visibility": "public",
	})
	req := httptest.NewRequest(http.MethodPost, "/private/skills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil || res.StatusCode != fiber.StatusOK {
		t.Fatalf("create failed: err=%v status=%d", err, res.StatusCode)
	}

	raw, _ := io.ReadAll(res.Body)
	var created map[string]any
	if err := json.Unmarshal(raw, &created); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	tags, ok := created["tags"].([]any)
	if !ok {
		t.Fatalf("expected tags array, got %v", created["tags"])
	}
	hasSkillTag := false
	for _, tag := range tags {
		if tag == "skill" {
			hasSkillTag = true
			break
		}
	}
	if !hasSkillTag {
		t.Fatalf("expected 'skill' tag to be auto-added, got %v", tags)
	}
}

func TestUpdateDeleteAndListAllSkills(t *testing.T) {
	t.Setenv("PORTFOLIO_DATA_DIR", filepath.Join(t.TempDir(), "data"))
	app := fiber.New()
	app.Post("/private/skills", CreateSkill)
	app.Put("/private/skills/:id", UpdateSkill)
	app.Delete("/private/skills/:id", DeleteSkill)
	app.Get("/private/skills", ListAllSkills)

	createBody, _ := json.Marshal(map[string]any{
		"title":      "Docker",
		"summary":    "Contenedores",
		"body":       "Detalle Docker",
		"tags":       []string{"skill", "devops"},
		"visibility": "public",
	})
	createReq := httptest.NewRequest(http.MethodPost, "/private/skills", bytes.NewReader(createBody))
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

	// Update skill
	updateBody, _ := json.Marshal(map[string]any{
		"title":      "Docker Avanzado",
		"summary":    "Contenedores y orquestacion",
		"body":       "Docker + Compose",
		"tags":       []string{"skill", "devops", "docker"},
		"visibility": "private",
	})
	updateReq := httptest.NewRequest(http.MethodPut, "/private/skills/"+id, bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRes, err := app.Test(updateReq, -1)
	if err != nil || updateRes.StatusCode != fiber.StatusOK {
		t.Fatalf("update failed: %v status=%d", err, updateRes.StatusCode)
	}

	// List all (should include private)
	listReq := httptest.NewRequest(http.MethodGet, "/private/skills", nil)
	listRes, err := app.Test(listReq, -1)
	if err != nil || listRes.StatusCode != fiber.StatusOK {
		t.Fatalf("list all failed: %v status=%d", err, listRes.StatusCode)
	}

	listRaw, _ := io.ReadAll(listRes.Body)
	var listResult map[string]any
	if err := json.Unmarshal(listRaw, &listResult); err != nil {
		t.Fatalf("unmarshal list failed: %v", err)
	}
	items, ok := listResult["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected 1 skill in list all, got %v", listResult["items"])
	}

	// Delete skill
	deleteReq := httptest.NewRequest(http.MethodDelete, "/private/skills/"+id, nil)
	deleteRes, err := app.Test(deleteReq, -1)
	if err != nil || deleteRes.StatusCode != fiber.StatusOK {
		t.Fatalf("delete failed: %v status=%d", err, deleteRes.StatusCode)
	}

	// Verify empty after delete
	listReq2 := httptest.NewRequest(http.MethodGet, "/private/skills", nil)
	listRes2, err := app.Test(listReq2, -1)
	if err != nil || listRes2.StatusCode != fiber.StatusOK {
		t.Fatalf("list after delete failed: %v status=%d", err, listRes2.StatusCode)
	}

	listRaw2, _ := io.ReadAll(listRes2.Body)
	var listResult2 map[string]any
	if err := json.Unmarshal(listRaw2, &listResult2); err != nil {
		t.Fatalf("unmarshal list2 failed: %v", err)
	}
	items2, ok := listResult2["items"].([]any)
	if !ok || len(items2) != 0 {
		t.Fatalf("expected 0 skills after delete, got %d", len(items2))
	}
}

func TestUpdateSkillNotFound(t *testing.T) {
	t.Setenv("PORTFOLIO_DATA_DIR", filepath.Join(t.TempDir(), "data"))
	app := fiber.New()
	app.Put("/private/skills/:id", UpdateSkill)

	body, _ := json.Marshal(map[string]any{"title": "X"})
	req := httptest.NewRequest(http.MethodPut, "/private/skills/nonexistent-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 for missing skill, got %d", res.StatusCode)
	}
}

func TestDeleteSkillNotFound(t *testing.T) {
	t.Setenv("PORTFOLIO_DATA_DIR", filepath.Join(t.TempDir(), "data"))
	app := fiber.New()
	app.Delete("/private/skills/:id", DeleteSkill)

	req := httptest.NewRequest(http.MethodDelete, "/private/skills/nonexistent-id", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 for missing skill, got %d", res.StatusCode)
	}
}

func TestListPublicSkillsFiltersPrivate(t *testing.T) {
	t.Setenv("PORTFOLIO_DATA_DIR", filepath.Join(t.TempDir(), "data"))
	app := fiber.New()
	app.Post("/private/skills", CreateSkill)
	app.Get("/skills", ListPublicSkills)

	// Create a private skill
	body, _ := json.Marshal(map[string]any{
		"title":      "Skill Privado",
		"tags":       []string{"skill"},
		"visibility": "private",
	})
	req := httptest.NewRequest(http.MethodPost, "/private/skills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil || res.StatusCode != fiber.StatusOK {
		t.Fatalf("create failed: err=%v status=%d", err, res.StatusCode)
	}

	// List public — should be empty
	listReq := httptest.NewRequest(http.MethodGet, "/skills", nil)
	listRes, err := app.Test(listReq, -1)
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
		t.Fatalf("expected 0 public skills (private only created), got %d", len(items))
	}
}
