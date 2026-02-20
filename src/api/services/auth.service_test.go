package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/repository"
	"backend-yonathan/src/repository/memory"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
)

// errSaveUserRepo wraps a UserRepository and always fails on SaveUser.
type errSaveUserRepo struct {
	delegate repository.UserRepository
}

func (e errSaveUserRepo) SaveUser(_ context.Context, _ models.User) error {
	return errors.New("save failed")
}
func (e errSaveUserRepo) GetUserByID(ctx context.Context, id string) (models.User, error) {
	return e.delegate.GetUserByID(ctx, id)
}
func (e errSaveUserRepo) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	return e.delegate.GetUserByEmail(ctx, email)
}

func TestLoginRejectsInvalidPayload(t *testing.T) {
	svc := NewAuthService(memory.NewUserRepository())
	app := fiber.New()
	app.Post("/login", svc.Login)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("{invalid-json"))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestRegisterDisabledByDefault(t *testing.T) {
	svc := NewAuthService(memory.NewUserRepository())
	app := fiber.New()
	app.Post("/register", svc.Register)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"ok@test.com","password":"Test1234","username":"tester"}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 when registration disabled, got %d", res.StatusCode)
	}
}

func TestRegisterRejectsInvalidPayload(t *testing.T) {
	t.Setenv("REGISTRATION_ENABLED", "true")
	svc := NewAuthService(memory.NewUserRepository())
	app := fiber.New()
	app.Post("/register", svc.Register)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader("{invalid-json"))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestRegisterAndLoginSuccess(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret")
	t.Setenv("REGISTRATION_ENABLED", "true")

	repo := memory.NewUserRepository()
	svc := NewAuthService(repo)

	app := fiber.New()
	app.Post("/register", svc.Register)
	app.Post("/login", svc.Login)

	registerReq := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"mail@test.com","password":"Test1234","username":"tester"}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRes, err := app.Test(registerReq)
	if err != nil || registerRes.StatusCode != fiber.StatusOK {
		t.Fatalf("register failed err=%v status=%d", err, registerRes.StatusCode)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"mail@test.com","password":"Test1234"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRes, err := app.Test(loginReq)
	if err != nil || loginRes.StatusCode != fiber.StatusOK {
		t.Fatalf("login failed err=%v status=%d", err, loginRes.StatusCode)
	}

	raw, _ := io.ReadAll(loginRes.Body)
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("invalid login payload: %v", err)
	}
	if _, ok := payload["token"]; !ok {
		t.Fatalf("expected token in login payload")
	}
}

func TestLoginUserNotFound(t *testing.T) {
	svc := NewAuthService(memory.NewUserRepository())
	app := fiber.New()
	app.Post("/login", svc.Login)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"nobody@test.com","password":"Test1234"}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret")

	repo := memory.NewUserRepository()
	hashed, _ := bcrypt.GenerateFromPassword([]byte("RealPass1"), bcrypt.DefaultCost)
	_ = repo.SaveUser(context.Background(), models.User{
		UserId:   "u-1",
		Email:    "user@test.com",
		Password: string(hashed),
		UserName: "tester",
	})

	svc := NewAuthService(repo)
	app := fiber.New()
	app.Post("/login", svc.Login)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"user@test.com","password":"WrongPass1"}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}

func TestRegisterUserAlreadyExists(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret")
	t.Setenv("REGISTRATION_ENABLED", "true")

	repo := memory.NewUserRepository()
	svc := NewAuthService(repo)
	app := fiber.New()
	app.Post("/register", svc.Register)

	body := `{"email":"dup@test.com","password":"Test1234","username":"tester"}`

	// First registration
	req1 := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	res1, err := app.Test(req1)
	if err != nil || res1.StatusCode != fiber.StatusOK {
		t.Fatalf("first register failed: err=%v status=%d", err, res1.StatusCode)
	}

	// Duplicate registration
	req2 := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	res2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 on duplicate register, got %d", res2.StatusCode)
	}
}

func TestRegisterSaveUserFails(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret")
	t.Setenv("REGISTRATION_ENABLED", "true")

	// Use a repo where GetUserByEmail always returns not-found but SaveUser fails.
	svc := NewAuthService(errSaveUserRepo{delegate: memory.NewUserRepository()})
	app := fiber.New()
	app.Post("/register", svc.Register)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"fail@test.com","password":"Test1234","username":"tester"}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.StatusCode)
	}
}

func TestRegisterInvalidEmail(t *testing.T) {
	t.Setenv("REGISTRATION_ENABLED", "true")
	svc := NewAuthService(memory.NewUserRepository())
	app := fiber.New()
	app.Post("/register", svc.Register)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"not-an-email","password":"Test1234","username":"tester"}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestRegisterWeakPassword(t *testing.T) {
	t.Setenv("REGISTRATION_ENABLED", "true")
	svc := NewAuthService(memory.NewUserRepository())
	app := fiber.New()
	app.Post("/register", svc.Register)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"ok@test.com","password":"weak","username":"tester"}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestRegisterEmptyUsername(t *testing.T) {
	t.Setenv("REGISTRATION_ENABLED", "true")
	svc := NewAuthService(memory.NewUserRepository())
	app := fiber.New()
	app.Post("/register", svc.Register)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"ok@test.com","password":"Test1234","username":""}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestRefreshTokenSuccess(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret")

	app := fiber.New()
	app.Post("/refresh", func(c fiber.Ctx) error {
		c.Locals("userId", "u-1")
		c.Locals("username", "tester")
		return RefreshToken(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	res, err := app.Test(req)
	if err != nil || res.StatusCode != fiber.StatusOK {
		t.Fatalf("refresh failed err=%v status=%d", err, res.StatusCode)
	}

	raw, _ := io.ReadAll(res.Body)
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("invalid refresh payload: %v", err)
	}
	if _, ok := payload["token"]; !ok {
		t.Fatalf("expected token in refresh payload")
	}
}

func TestRefreshTokenMissingLocals(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret")

	app := fiber.New()
	app.Post("/refresh", RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}

func TestRefreshTokenEmptyUsername(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret")

	app := fiber.New()
	app.Post("/refresh", func(c fiber.Ctx) error {
		c.Locals("userId", "u-1")
		c.Locals("username", "")
		return RefreshToken(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 for empty username, got %d", res.StatusCode)
	}
}
