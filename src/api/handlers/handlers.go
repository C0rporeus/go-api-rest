package handlers

import (
	authServices "backend-yonathan/src/api/services"
	"backend-yonathan/src/config"
	"log"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	dbClient, err := config.ConfigAWS()
	if err != nil {
		log.Fatalf("Error al configurar AWS: %v", err)
	}
	api := app.Group("/api")
	app.Post("/api/register", func(c *fiber.Ctx) error {
		return authServices.Register(c, dbClient)
	})
	app.Post("/api/login", func(c *fiber.Ctx) error {
		return authServices.Login(c, dbClient)
	})

	api.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong desde handlers")
	})
}
