package handlers

import (
	jwtMiddleware "backend-yonathan/src/api/middlewares"
	"backend-yonathan/src/api/services"
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/repository"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

func SetupRoutes(app *fiber.App, userRepo repository.UserRepository, expRepo repository.ExperienceRepository) {
	auth := services.NewAuthService(userRepo)
	exp := services.NewExperienceService(expRepo)
	skill := services.NewSkillService(expRepo)

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
	public.Get("/health", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	public.Post("/login", authLimiter, auth.Login)
	public.Post("/register", authLimiter, auth.Register)
	public.Post("/logout", auth.Logout)
	public.Post("/contact", authLimiter, services.SubmitContact)
	public.Get("/experiences", exp.ListPublicExperiences)
	public.Get("/skills", skill.ListPublicSkills)

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

	private.Get("/experiences", exp.ListAllExperiences)
	private.Post("/experiences", exp.CreateExperience)
	private.Put("/experiences/:id", exp.UpdateExperience)
	private.Delete("/experiences/:id", exp.DeleteExperience)
	private.Post("/upload-image", services.UploadImage)

	private.Get("/skills", skill.ListAllSkills)
	private.Post("/skills", skill.CreateSkill)
	private.Put("/skills/:id", skill.UpdateSkill)
	private.Delete("/skills/:id", skill.DeleteSkill)

	private.Get("/ops/metrics", services.GetOpsMetrics)
	private.Get("/ops/alerts", services.GetOpsAlerts)
	private.Get("/ops/health", services.GetOpsHealth)
	private.Get("/ops/history", services.GetOpsHistory)
	private.Get("/ops/summary", services.GetOpsSummary)
}
