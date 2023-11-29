package handlers

import (
	authServices "backend-yonathan/src/api/services"
	"backend-yonathan/src/config"
	"log"

	"github.com/gofiber/fiber/v2"
	_ "github.com/gofiber/swagger"
)

// @Summary Registro de usuarios
// @Description Registro de usuarios api Yonathan Gutierrez Dev
// @Tags Auth
// @Accept json
// @Produce json
// @Param user body userModel.User true "User"
// @Success 200 {object} userModel.User
// @Failure 400 {string} string "bad request"
// @Router /api/register [post]
func RegisterUser(app *fiber.App) {
	dbClient, err := config.ConfigAWS()
	if err != nil {
		log.Fatalf("Error al configurar AWS: %v", err)
	}

	app.Post("/api/register", func(c *fiber.Ctx) error {
		return authServices.Register(c, dbClient)
	})
}

// @Summary Login de usuarios
// @Description Login de usuarios api Yonathan Gutierrez Dev
// @Tags Auth
// @Accept json
// @Produce json
// @Param user body userModel.User true "User"
// @Success 200 {object} userModel.User
// @Failure 400 {string} string "bad request"
// @Router /api/login [post]
func SetupRoutes(app *fiber.App) {
	dbClient, err := config.ConfigAWS()
	if err != nil {
		log.Fatalf("Error al configurar AWS: %v", err)
	}
	app.Post("/api/login", func(c *fiber.Ctx) error {
		return authServices.Login(c, dbClient)
	})
}
