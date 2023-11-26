package main

import (
	"./src/api/handlers"
	"./src/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	app := fiber.New()
	dbClient, s3Client, err := config.ConfigAWS()
	if err != nil {
		logger.Fatal(err)
		panic(err)
	}

	handlers.SetupRoutes(app, dbClient, s3Client)

	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
	}))

	app.Use(logger.New())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("¡Hola, mundo desde Fiber!")
	})
	app.Listen(":3000")
}
