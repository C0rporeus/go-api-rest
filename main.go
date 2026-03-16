package main

import (
	"backend-yonathan/src/api/handlers"
	"backend-yonathan/src/api/services"
	"backend-yonathan/src/config"
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/pkg/telemetry"
	"backend-yonathan/src/repository"
	dynamoRepo "backend-yonathan/src/repository/dynamodb"
	firestoreRepo "backend-yonathan/src/repository/firestore"
	jsonRepo "backend-yonathan/src/repository/json"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	_ "backend-yonathan/docs"

	swaggo "github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// @title          Portfolio API
// @version        1.0
// @description    API REST para portfolio personal de Yonathan Gutierrez
// @host           localhost:3100
// @BasePath       /
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 JWT token con formato "Bearer {token}"

func init() {
	// En producción (Cloud Run) no hay .env; las vars ya están inyectadas por la plataforma.
	_ = godotenv.Load()
}

func buildRepositories() (repository.UserRepository, repository.ExperienceRepository) {
	switch strings.ToLower(os.Getenv("DB_PROVIDER")) {
	case "dynamodb":
		client, err := config.ConfigAWS()
		if err != nil {
			log.Fatalf("AWS config error: %v", err)
		}
		return dynamoRepo.NewUserRepository(client), jsonRepo.NewExperienceRepository()
	case "firestore":
		client, err := config.ConfigFirestore()
		if err != nil {
			log.Fatalf("Firestore config error: %v", err)
		}
		return firestoreRepo.NewUserRepository(client), firestoreRepo.NewExperienceRepository(client)
	case "json":
		return jsonRepo.NewUserRepository(), jsonRepo.NewExperienceRepository()
	default:
		log.Fatalf("DB_PROVIDER no configurado o no reconocido. Valores validos: dynamodb, firestore, json")
		return nil, nil
	}
}

func main() {
	app := fiber.New(fiber.Config{
		TrustProxy:  true,
		ProxyHeader: fiber.HeaderXForwardedFor,
		BodyLimit:   constants.BodyLimitDefault,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				code = fiberErr.Code
			}
			return apiresponse.Error(c, code, "internal_error", "Error procesando la solicitud", err.Error())
		},
	})

	// Security headers (X-Content-Type-Options, X-Frame-Options, CSP, etc.)
	app.Use(helmet.New())

	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(allowedOrigins, ","),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
	}))

	// Global rate limiter
	app.Use(limiter.New(limiter.Config{
		Max:        constants.RateLimitGlobalMax,
		Expiration: time.Duration(constants.RateLimitGlobalWindow) * time.Second,
		LimitReached: func(c fiber.Ctx) error {
			return apiresponse.Error(c, fiber.StatusTooManyRequests,
				"rate_limit_exceeded",
				"Demasiadas solicitudes. Intenta de nuevo en un momento.",
				nil)
		},
	}))
	app.Use(func(c fiber.Ctx) error {
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set("X-Request-ID", requestID)
		c.Locals("requestid", requestID)

		start := time.Now()
		err := c.Next()

		logPayload := map[string]interface{}{
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
			"requestId":  requestID,
			"method":     c.Method(),
			"path":       c.Path(),
			"statusCode": c.Response().StatusCode(),
			"latencyMs":  time.Since(start).Milliseconds(),
		}
		if err != nil {
			logPayload["error"] = err.Error()
		}

		statusCode := c.Response().StatusCode()
		path := c.Path()
		isAuthFailure := statusCode == fiber.StatusUnauthorized &&
			(strings.HasPrefix(path, "/api/private") || strings.HasPrefix(path, "/api/login"))
		telemetry.TrackRequest(statusCode, isAuthFailure)

		if encoded, marshalErr := json.Marshal(logPayload); marshalErr == nil {
			log.Println(string(encoded))
		}
		return err
	})

	userRepo, expRepo := buildRepositories()
	services.SeedAdminUser(userRepo)
	handlers.SetupRoutes(app, userRepo, expRepo)
	app.Get("/swagger/*", swaggo.HandlerDefault)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3100"
	}

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}
