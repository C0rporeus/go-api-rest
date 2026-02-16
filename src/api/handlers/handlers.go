package handlers

import (
	jwtMiddleware "backend-yonathan/src/api/middlewares"
	authServices "backend-yonathan/src/api/services"
	"backend-yonathan/src/config"
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/telemetry"
	"log"

	"github.com/gofiber/fiber/v2"
	_ "github.com/gofiber/swagger"
)

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
	tools.Get("/dns/resolve", authServices.ResolveDomain)
	tools.Get("/dns/propagation", authServices.CheckPropagation)
	tools.Get("/dns/mail-records", authServices.GetMailRecords)
	tools.Get("/dns/blacklist", authServices.CheckBlacklist)
	public.Get("/experiences", authServices.ListPublicExperiences)
	public.Get("/skills", authServices.ListPublicSkills)

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
	private.Get("/skills", authServices.ListAllSkills)
	private.Post("/skills", authServices.CreateSkill)
	private.Put("/skills/:id", authServices.UpdateSkill)
	private.Delete("/skills/:id", authServices.DeleteSkill)
	private.Get("/ops/metrics", func(c *fiber.Ctx) error {
		return apiresponse.Success(c, telemetry.Snapshot())
	})
	private.Get("/ops/alerts", func(c *fiber.Ctx) error {
		return apiresponse.Success(c, telemetry.Alerts())
	})
	private.Get("/ops/health", func(c *fiber.Ctx) error {
		return apiresponse.Success(c, telemetry.Health())
	})
	private.Get("/ops/history", func(c *fiber.Ctx) error {
		return apiresponse.Success(c, telemetry.History())
	})
	private.Get("/ops/summary", func(c *fiber.Ctx) error {
		return apiresponse.Success(c, telemetry.Summary())
	})
}
