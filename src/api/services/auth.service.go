package services

import (
	"context"
	"strings"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/pkg/sanitizer"
	jwtManager "backend-yonathan/src/pkg/utils"
	"backend-yonathan/src/repository"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication business logic.
type AuthService struct {
	users repository.UserRepository
}

// NewAuthService creates an AuthService backed by the given UserRepository.
func NewAuthService(repo repository.UserRepository) *AuthService {
	return &AuthService{users: repo}
}

func respondWithToken(c fiber.Ctx, userID, username string) error {
	token, err := jwtManager.GenerateToken(userID, username)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "token_generation_failed", "No se pudo generar el token", err.Error())
	}
	return apiresponse.Success(c, fiber.Map{"token": token})
}

// Register godoc
// @Summary      Registro de usuarios
// @Description  Crea una cuenta nueva y devuelve un JWT
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        user  body  object{email=string,password=string,username=string}  true  "Datos de registro"
// @Success      200  {object}  map[string]string  "token"
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/register [post]
func (s *AuthService) Register(c fiber.Ctx) error {
	if !constants.RegistrationEnabled() {
		return apiresponse.Error(c, fiber.StatusForbidden, "registration_disabled", "El registro de usuarios esta deshabilitado", nil)
	}

	var user models.User
	if err := c.Bind().Body(&user); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	user.Email = strings.TrimSpace(strings.ToLower(user.Email))
	user.UserName = sanitizer.SanitizePlainText(user.UserName, constants.MaxTitleLength)

	if !sanitizer.IsValidEmail(user.Email) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_email", "Formato de email invalido", nil)
	}
	if !sanitizer.IsValidPassword(user.Password) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "weak_password",
			"La contrasena debe tener al menos 8 caracteres, una mayuscula, una minuscula y un numero", nil)
	}
	if user.UserName == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_username", "El nombre de usuario es requerido", nil)
	}

	_, err := s.users.GetUserByEmail(context.Background(), user.Email)
	if err == nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "user_already_exists", "El usuario ya existe", nil)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "password_hash_failed", "No se pudo procesar la contrasena", err.Error())
	}
	user.Password = string(hashedPassword)
	if err := s.users.SaveUser(context.Background(), user); err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "save_user_failed", "No se pudo registrar el usuario", err.Error())
	}
	return respondWithToken(c, user.UserId, user.UserName)
}

// Login godoc
// @Summary      Login de usuarios
// @Description  Autentica con email/password y devuelve un JWT
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        credentials  body  object{email=string,password=string}  true  "Credenciales"
// @Success      200  {object}  map[string]string  "token"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /api/login [post]
func (s *AuthService) Login(c fiber.Ctx) error {
	var loginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind().Body(&loginRequest); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	loginRequest.Email = strings.TrimSpace(strings.ToLower(loginRequest.Email))

	if !sanitizer.IsValidEmail(loginRequest.Email) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_email", "Formato de email invalido", nil)
	}
	if loginRequest.Password == "" || len(loginRequest.Password) > constants.MaxPasswordLength {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_password", "Contrasena invalida", nil)
	}

	user, err := s.users.GetUserByEmail(context.Background(), loginRequest.Email)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusUnauthorized, "invalid_credentials", "Unauthorized", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginRequest.Password)); err != nil {
		return apiresponse.Error(c, fiber.StatusUnauthorized, "invalid_credentials", "Unauthorized", nil)
	}

	return respondWithToken(c, user.UserId, user.UserName)
}

// GetCurrentUser godoc
// @Summary      Usuario autenticado
// @Description  Devuelve userId y username del JWT actual
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  map[string]interface{}
// @Router       /api/private/me [get]
func GetCurrentUser(c fiber.Ctx) error {
	return apiresponse.Success(c, fiber.Map{
		"userId":   c.Locals("userId"),
		"username": c.Locals("username"),
	})
}

// RefreshToken godoc
// @Summary      Renovar token JWT
// @Description  Genera un nuevo JWT a partir del token actual (debe ser valido)
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]string  "token"
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/private/refresh [post]
func RefreshToken(c fiber.Ctx) error {
	userId, _ := c.Locals("userId").(string)
	username, _ := c.Locals("username").(string)

	if userId == "" || username == "" {
		return apiresponse.Error(c, fiber.StatusUnauthorized, "invalid_session", "Sesion invalida", nil)
	}

	return respondWithToken(c, userId, username)
}

