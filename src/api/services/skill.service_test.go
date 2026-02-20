package services

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/repository/memory"

	"github.com/gofiber/fiber/v3"
)

func TestCreateAndListPublicSkills(t *testing.T) {
	repo := memory.NewExperienceRepository()
	svc := NewSkillService(repo)

	app := fiber.New()
	app.Post("/private/skills", svc.CreateSkill)
	app.Get("/skills", svc.ListPublicSkills)

	body, _ := json.Marshal(map[string]any{
		"title":      "Go Avanzado",
		"summary":    "Capacidad en Go",
		"body":       "Detalle de la habilidad",
		"tags":       []string{"skill", "go"},
		"visibility": constants.VisibilityPublic,
	})

	createReq := httptest.NewRequest(http.MethodPost, "/private/skills", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRes, err := app.Test(createReq)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if createRes.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 on create, got %d", createRes.StatusCode)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/skills", nil)
	listRes, err := app.Test(listReq)
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
	repo := memory.NewExperienceRepository()
	svc := NewSkillService(repo)

	app := fiber.New()
	app.Post("/private/skills", svc.CreateSkill)

	body, _ := json.Marshal(map[string]any{
		"summary": "Sin titulo",
		"tags":    []string{"skill"},
	})
	req := httptest.NewRequest(http.MethodPost, "/private/skills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 on missing title, got %d", res.StatusCode)
	}
}

func TestCreateSkillEnsuresSkillTag(t *testing.T) {
	repo := memory.NewExperienceRepository()
	svc := NewSkillService(repo)

	app := fiber.New()
	app.Post("/private/skills", svc.CreateSkill)

	body, _ := json.Marshal(map[string]any{
		"title":      "TypeScript",
		"tags":       []string{"frontend"},
		"visibility": constants.VisibilityPublic,
	})
	req := httptest.NewRequest(http.MethodPost, "/private/skills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
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
	repo := memory.NewExperienceRepository()
	svc := NewSkillService(repo)

	app := fiber.New()
	app.Post("/private/skills", svc.CreateSkill)
	app.Put("/private/skills/:id", svc.UpdateSkill)
	app.Delete("/private/skills/:id", svc.DeleteSkill)
	app.Get("/private/skills", svc.ListAllSkills)

	createBody, _ := json.Marshal(map[string]any{
		"title":      "Docker",
		"summary":    "Contenedores",
		"body":       "Detalle Docker",
		"tags":       []string{"skill", "devops"},
		"visibility": constants.VisibilityPublic,
	})
	createReq := httptest.NewRequest(http.MethodPost, "/private/skills", bytes.NewReader(createBody))
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
		"title":      "Docker Avanzado",
		"summary":    "Contenedores y orquestacion",
		"body":       "Docker + Compose",
		"tags":       []string{"skill", "devops", "docker"},
		"visibility": constants.VisibilityPrivate,
	})
	updateReq := httptest.NewRequest(http.MethodPut, "/private/skills/"+id, bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRes, err := app.Test(updateReq)
	if err != nil || updateRes.StatusCode != fiber.StatusOK {
		t.Fatalf("update failed: %v status=%d", err, updateRes.StatusCode)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/private/skills", nil)
	listRes, err := app.Test(listReq)
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

	deleteReq := httptest.NewRequest(http.MethodDelete, "/private/skills/"+id, nil)
	deleteRes, err := app.Test(deleteReq)
	if err != nil || deleteRes.StatusCode != fiber.StatusOK {
		t.Fatalf("delete failed: %v status=%d", err, deleteRes.StatusCode)
	}

	listReq2 := httptest.NewRequest(http.MethodGet, "/private/skills", nil)
	listRes2, err := app.Test(listReq2)
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
	repo := memory.NewExperienceRepository()
	svc := NewSkillService(repo)

	app := fiber.New()
	app.Put("/private/skills/:id", svc.UpdateSkill)

	body, _ := json.Marshal(map[string]any{"title": "X"})
	req := httptest.NewRequest(http.MethodPut, "/private/skills/00000000-0000-0000-0000-000000000099", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 for missing skill, got %d", res.StatusCode)
	}
}

func TestDeleteSkillNotFound(t *testing.T) {
	repo := memory.NewExperienceRepository()
	svc := NewSkillService(repo)

	app := fiber.New()
	app.Delete("/private/skills/:id", svc.DeleteSkill)

	req := httptest.NewRequest(http.MethodDelete, "/private/skills/00000000-0000-0000-0000-000000000099", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 for missing skill, got %d", res.StatusCode)
	}
}

func TestListPublicSkillsFiltersPrivate(t *testing.T) {
	repo := memory.NewExperienceRepository()
	svc := NewSkillService(repo)

	app := fiber.New()
	app.Post("/private/skills", svc.CreateSkill)
	app.Get("/skills", svc.ListPublicSkills)

	body, _ := json.Marshal(map[string]any{
		"title":      "Skill Privado",
		"tags":       []string{"skill"},
		"visibility": constants.VisibilityPrivate,
	})
	req := httptest.NewRequest(http.MethodPost, "/private/skills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil || res.StatusCode != fiber.StatusOK {
		t.Fatalf("create failed: err=%v status=%d", err, res.StatusCode)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/skills", nil)
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
		t.Fatalf("expected 0 public skills (private only created), got %d", len(items))
	}
}

func TestUpdateSkillNotASkill(t *testing.T) {
	repo := memory.NewExperienceRepository()
	expSvc := NewExperienceService(repo)
	svc := NewSkillService(repo)

	app := fiber.New()
	app.Post("/private/experiences", expSvc.CreateExperience)
	app.Put("/private/skills/:id", svc.UpdateSkill)

	// Create a regular experience (not a skill)
	createBody, _ := json.Marshal(map[string]any{
		"title":      "Regular Experience",
		"tags":       []string{"api"},
		"visibility": constants.VisibilityPublic,
	})
	createReq := httptest.NewRequest(http.MethodPost, "/private/experiences", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRes, err := app.Test(createReq)
	if err != nil || createRes.StatusCode != fiber.StatusOK {
		t.Fatalf("create experience failed: err=%v status=%d", err, createRes.StatusCode)
	}

	raw, _ := io.ReadAll(createRes.Body)
	var created map[string]any
	json.Unmarshal(raw, &created)
	id, _ := created["id"].(string)

	// Try to update it as a skill — should 404
	updateBody, _ := json.Marshal(map[string]any{"title": "Updated"})
	req := httptest.NewRequest(http.MethodPut, "/private/skills/"+id, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 for non-skill update, got %d", res.StatusCode)
	}
}

func TestDeleteSkillNotASkill(t *testing.T) {
	repo := memory.NewExperienceRepository()
	expSvc := NewExperienceService(repo)
	svc := NewSkillService(repo)

	app := fiber.New()
	app.Post("/private/experiences", expSvc.CreateExperience)
	app.Delete("/private/skills/:id", svc.DeleteSkill)

	// Create a regular experience (not a skill)
	createBody, _ := json.Marshal(map[string]any{
		"title":      "Regular Experience",
		"tags":       []string{"api"},
		"visibility": constants.VisibilityPublic,
	})
	createReq := httptest.NewRequest(http.MethodPost, "/private/experiences", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRes, err := app.Test(createReq)
	if err != nil || createRes.StatusCode != fiber.StatusOK {
		t.Fatalf("create experience failed: err=%v status=%d", err, createRes.StatusCode)
	}

	raw, _ := io.ReadAll(createRes.Body)
	var created map[string]any
	json.Unmarshal(raw, &created)
	id, _ := created["id"].(string)

	// Try to delete it as a skill — should 404
	req := httptest.NewRequest(http.MethodDelete, "/private/skills/"+id, nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 for non-skill delete, got %d", res.StatusCode)
	}
}
