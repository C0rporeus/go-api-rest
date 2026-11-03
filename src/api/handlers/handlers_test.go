package handlers

import (
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/repository/memory"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestAuthRateLimiter(t *testing.T) {
	app := fiber.New()
	
	// Create mock repositories
	userRepo := memory.NewUserRepository()
	expRepo := memory.NewExperienceRepository()
	
	// Setup routes exactly as in production
	SetupRoutes(app, userRepo, expRepo)

	// Max limit for auth is constants.RateLimitAuthMax
	limit := constants.RateLimitAuthMax

	// We hit the endpoint limit + 1 times
	for i := 0; i < limit+1; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
		// We can send empty body, it will fail validation (400), but that's fine.
		// What we care about is that after `limit` requests, we get a 429.
		res, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if i < limit {
			// Expected to pass the rate limiter and hit the handler
			if res.StatusCode == fiber.StatusTooManyRequests {
				t.Fatalf("request %d failed with 429 too early", i+1)
			}
		} else {
			// Expected to be blocked by rate limiter
			if res.StatusCode != fiber.StatusTooManyRequests {
				t.Fatalf("request %d expected 429, got %d", i+1, res.StatusCode)
			}
		}
	}
}

func TestMailAuditHasItsOwnStricterRateLimit(t *testing.T) {
	app := fiber.New()
	SetupRoutes(app, memory.NewUserRepository(), memory.NewExperienceRepository())

	limit := constants.RateLimitMailAuditMax
	if limit >= constants.RateLimitToolsMax {
		t.Fatalf("mail-audit limit (%d) must be stricter than the generic tools limit (%d)", limit, constants.RateLimitToolsMax)
	}
	for i := 0; i < limit+1; i++ {
		// An invalid domain answers 400 without any DNS query: only the limiter is exercised.
		res, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/tools/dns/mail-audit?domain=bad_domain", nil))
		if err != nil {
			t.Fatal(err)
		}
		if i < limit && res.StatusCode == fiber.StatusTooManyRequests {
			t.Fatalf("request %d limited too early", i+1)
		}
		if i == limit && res.StatusCode != fiber.StatusTooManyRequests {
			t.Fatalf("request %d expected 429, got %d", i+1, res.StatusCode)
		}
	}
}
