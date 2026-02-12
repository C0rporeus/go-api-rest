package handlers

import (
	jwtMiddleware "backend-yonathan/src/api/middlewares"
	authServices "backend-yonathan/src/api/services"
	"backend-yonathan/src/config"
	"backend-yonathan/src/pkg/apiresponse"
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

	public := app.Group("/api")
	public.Post("/login", func(c *fiber.Ctx) error {
		return authServices.Login(c, dbClient)
	})
	public.Post("/register", func(c *fiber.Ctx) error {
		return authServices.Register(c, dbClient)
	})

	tools := public.Group("/tools")
	tools.Post("/base64/encode", authServices.EncodeBase64)
	tools.Post("/base64/decode", authServices.DecodeBase64)
	tools.Get("/uuid/v4", authServices.GenerateUUIDv4)
	tools.Post("/certs/self-signed", authServices.GenerateSelfSignedCert)
	public.Get("/experiences", authServices.ListPublicExperiences)

	private := app.Group("/api/private", jwtMiddleware.JWTProtected())
	private.Get("/me", func(c *fiber.Ctx) error {
		return apiresponse.Success(c, fiber.Map{
			"userId":   c.Locals("userId"),
			"username": c.Locals("username"),
		})
	})
	private.Get("/experiences", authServices.ListAllExperiences)
	private.Post("/experiences", authServices.CreateExperience)
	private.Put("/experiences/:id", authServices.UpdateExperience)
	private.Delete("/experiences/:id", authServices.DeleteExperience)
}
