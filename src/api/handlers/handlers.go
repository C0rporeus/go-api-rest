package handlers

import (
	jwtMiddleware "backend-yonathan/src/api/middlewares"
	"backend-yonathan/src/api/services"
	"backend-yonathan/src/config"
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

func SetupRoutes(app *fiber.App) {
	dbClient, err := config.ConfigAWS()
	if err != nil {
		log.Fatalf("Error al configurar AWS: %v", err)
	}

	rateLimitReached := func(c fiber.Ctx) error {
		return apiresponse.Error(c, fiber.StatusTooManyRequests,
			"rate_limit_exceeded",
			"Demasiadas solicitudes. Intenta de nuevo en un momento.",
			nil)
	}

	// Stricter rate limiter for auth endpoints (brute-force protection)
	authLimiter := limiter.New(limiter.Config{
		Max:          constants.RateLimitAuthMax,
		Expiration:   time.Duration(constants.RateLimitAuthWindow) * time.Second,
		LimitReached: rateLimitReached,
	})

	// Moderate rate limiter for tools endpoints (abuse protection)
	toolsLimiter := limiter.New(limiter.Config{
		Max:          constants.RateLimitToolsMax,
		Expiration:   time.Duration(constants.RateLimitToolsWindow) * time.Second,
		LimitReached: rateLimitReached,
	})

	// --- Public routes ---

	public := app.Group("/api")
	public.Post("/login", authLimiter, func(c fiber.Ctx) error {
		return services.Login(c, dbClient)
	})
	public.Post("/register", authLimiter, func(c fiber.Ctx) error {
		return services.Register(c, dbClient)
	})
	public.Post("/contact", authLimiter, services.SubmitContact)
	public.Get("/experiences", services.ListPublicExperiences)
	public.Get("/skills", services.ListPublicSkills)

	// --- Tools (public, no auth) ---

	tools := public.Group("/tools", toolsLimiter)
	tools.Post("/base64/encode", services.EncodeBase64)
	tools.Post("/base64/decode", services.DecodeBase64)
	tools.Get("/uuid/v4", services.GenerateUUIDv4)
	tools.Post("/certs/self-signed", services.GenerateSelfSignedCert)
	tools.Get("/dns/resolve", services.ResolveDomain)
	tools.Get("/dns/propagation", services.CheckPropagation)
	tools.Get("/dns/mail-records", services.GetMailRecords)
	tools.Get("/dns/blacklist", services.CheckBlacklist)

	// --- Private routes (require JWT) ---

	private := app.Group("/api/private", jwtMiddleware.JWTProtected())
	private.Get("/me", services.GetCurrentUser)
	private.Post("/refresh", services.RefreshToken)

	private.Get("/experiences", services.ListAllExperiences)
	private.Post("/experiences", services.CreateExperience)
	private.Put("/experiences/:id", services.UpdateExperience)
	private.Delete("/experiences/:id", services.DeleteExperience)

	private.Get("/skills", services.ListAllSkills)
	private.Post("/skills", services.CreateSkill)
	private.Put("/skills/:id", services.UpdateSkill)
	private.Delete("/skills/:id", services.DeleteSkill)

	private.Get("/ops/metrics", services.GetOpsMetrics)
	private.Get("/ops/alerts", services.GetOpsAlerts)
	private.Get("/ops/health", services.GetOpsHealth)
	private.Get("/ops/history", services.GetOpsHistory)
	private.Get("/ops/summary", services.GetOpsSummary)
}
