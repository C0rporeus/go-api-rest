package main

import (
	"backend-yonathan/src/api/handlers"
	config "backend-yonathan/src/config"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

func init() {
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file")
    }
}

func main() {
	app := fiber.New()

	app.Use(cors.New())
	app.Use(logger.New())

	// usa la conexion a dynamo db que esta en la ruta src/api/services/auth.service.go
	config.ConfigAWS()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("¡Hola, mundo desde Fiber!")
	})

	handlers.SetupRoutes(app)

	if err := app.Listen(":3100"); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}
