package authServices

import (
	"errors"
	userModel "backend-yonathan/src/models"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

func TestLoginRejectsInvalidPayload(t *testing.T) {
	app := fiber.New()
	app.Post("/login", func(c *fiber.Ctx) error {
		return Login(c, nil)
	})

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("{invalid-json"))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestRegisterRejectsInvalidPayload(t *testing.T) {
	app := fiber.New()
	app.Post("/register", func(c *fiber.Ctx) error {
		return Register(c, nil)
	})

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader("{invalid-json"))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func withMockDBCalls(
	t *testing.T,
	queryMock func(*dynamodb.Client, *dynamodb.QueryInput) (*dynamodb.QueryOutput, error),
	putMock func(*dynamodb.Client, *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error),
	getMock func(*dynamodb.Client, *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error),
) {
	t.Helper()
	originalQuery := queryFunc
	originalPut := putItemFunc
	originalGet := getItemFunc

	if queryMock != nil {
		queryFunc = queryMock
	}
	if putMock != nil {
		putItemFunc = putMock
	}
	if getMock != nil {
		getItemFunc = getMock
	}

	t.Cleanup(func() {
		queryFunc = originalQuery
		putItemFunc = originalPut
		getItemFunc = originalGet
	})
}

func TestGetUserByEmailAndByID(t *testing.T) {
	withMockDBCalls(
		t,
		func(_ *dynamodb.Client, _ *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"userId":   &types.AttributeValueMemberS{Value: "u-1"},
						"email":    &types.AttributeValueMemberS{Value: "mail@test.com"},
						"password": &types.AttributeValueMemberS{Value: "hash"},
						"username": &types.AttributeValueMemberS{Value: "tester"},
					},
				},
			}, nil
		},
		nil,
		func(_ *dynamodb.Client, _ *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"userId":   &types.AttributeValueMemberS{Value: "u-1"},
					"email":    &types.AttributeValueMemberS{Value: "mail@test.com"},
					"password": &types.AttributeValueMemberS{Value: "hash"},
					"username": &types.AttributeValueMemberS{Value: "tester"},
				},
			}, nil
		},
	)

	userByEmail, err := GetUserByEmail(nil, "mail@test.com")
	if err != nil || userByEmail.Email != "mail@test.com" {
		t.Fatalf("expected user by email, err=%v user=%+v", err, userByEmail)
	}

	userByID, err := GetUserById(nil, "u-1")
	if err != nil || userByID.UserId != "u-1" {
		t.Fatalf("expected user by id, err=%v user=%+v", err, userByID)
	}
}

func TestGetUserByIdNotFoundAndError(t *testing.T) {
	withMockDBCalls(
		t,
		nil,
		nil,
		func(_ *dynamodb.Client, _ *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	)
	if _, err := GetUserById(nil, "missing"); err == nil {
		t.Fatalf("expected not found error")
	}

	withMockDBCalls(
		t,
		nil,
		nil,
		func(_ *dynamodb.Client, _ *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
			return nil, errors.New("get failed")
		},
	)
	if _, err := GetUserById(nil, "id"); err == nil {
		t.Fatalf("expected get error")
	}
}

func TestSaveUser(t *testing.T) {
	withMockDBCalls(
		t,
		nil,
		func(_ *dynamodb.Client, _ *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
		nil,
	)

	err := SaveUser(nil, userModel.User{
		Email:    "mail@test.com",
		Password: "pass",
		UserName: "tester",
	})
	if err != nil {
		t.Fatalf("expected save user success: %v", err)
	}
}

func TestRegisterAndLoginSuccess(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret")
	existingUserQueryCalls := 0

	hashed, err := bcrypt.GenerateFromPassword([]byte("1234"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}

	withMockDBCalls(
		t,
		func(_ *dynamodb.Client, _ *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
			existingUserQueryCalls++
			if existingUserQueryCalls == 1 {
				return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil
			}
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"userId":   &types.AttributeValueMemberS{Value: "u-login"},
						"email":    &types.AttributeValueMemberS{Value: "mail@test.com"},
						"password": &types.AttributeValueMemberS{Value: string(hashed)},
						"username": &types.AttributeValueMemberS{Value: "tester"},
					},
				},
			}, nil
		},
		func(_ *dynamodb.Client, _ *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
		nil,
	)

	app := fiber.New()
	app.Post("/register", func(c *fiber.Ctx) error { return Register(c, nil) })
	app.Post("/login", func(c *fiber.Ctx) error { return Login(c, nil) })

	registerReq := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"mail@test.com","password":"1234","username":"tester"}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRes, err := app.Test(registerReq, -1)
	if err != nil || registerRes.StatusCode != fiber.StatusOK {
		t.Fatalf("register failed err=%v status=%d", err, registerRes.StatusCode)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"mail@test.com","password":"1234"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRes, err := app.Test(loginReq, -1)
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

func TestRegisterAndLoginErrorPaths(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret")
	withMockDBCalls(
		t,
		func(_ *dynamodb.Client, _ *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
			return nil, errors.New("db query failed")
		},
		func(_ *dynamodb.Client, _ *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
			return nil, errors.New("db put failed")
		},
		func(_ *dynamodb.Client, _ *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
			return nil, errors.New("db get failed")
		},
	)

	app := fiber.New()
	app.Post("/login", func(c *fiber.Ctx) error { return Login(c, nil) })
	app.Post("/register", func(c *fiber.Ctx) error { return Register(c, nil) })

	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"mail@test.com","password":"1234"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRes, err := app.Test(loginReq, -1)
	if err != nil {
		t.Fatalf("login call failed: %v", err)
	}
	if loginRes.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected unauthorized status, got %d", loginRes.StatusCode)
	}

	registerReq := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"mail@test.com","password":"1234","username":"tester"}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRes, err := app.Test(registerReq, -1)
	if err != nil {
		t.Fatalf("register call failed: %v", err)
	}
	if registerRes.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected internal server error status, got %d", registerRes.StatusCode)
	}
}
