package main

import (
	"backend-yonathan/src/api/handlers"
	"backend-yonathan/src/pkg/apiresponse"
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	_ "backend-yonathan/docs"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/google/uuid"
	"github.com/gofiber/swagger"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				code = fiberErr.Code
			}
			return apiresponse.Error(c, code, "internal_error", "Error procesando la solicitud", err.Error())
		},
	})

	app.Use(cors.New())
	app.Use(func(c *fiber.Ctx) error {
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

		if encoded, marshalErr := json.Marshal(logPayload); marshalErr == nil {
			log.Println(string(encoded))
		}
		return err
	})

	handlers.SetupRoutes(app)
	app.Get("/swagger/*", swagger.HandlerDefault)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3100"
	}

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}
