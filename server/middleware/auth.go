// Package middleware รวม Fiber middleware ที่ใช้ร่วมกันใน server
package middleware

import (
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/ports"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

// ─── Context Keys ────────────────────────────────────────────────
// ใช้ c.Locals(key) เพื่อดึง claims ที่ middleware inject ไว้
//
//	userID := c.Locals(middleware.CtxUserID).(string)
//	roles  := c.Locals(middleware.CtxRoles).([]string)

const (
	CtxUserID = "userID"
	CtxRoles  = "roles"
)

// ─── JWT Auth Config ─────────────────────────────────────────────

// AuthConfig holds configuration for the JWT auth middleware.
type AuthConfig struct {
	// TokenSigner is the port that verifies tokens (JWT or dev-token).
	TokenSigner ports.TokenSigner
}

// ─── API Key Config ──────────────────────────────────────────────

// APIKeyConfig holds configuration for the API key middleware.
type APIKeyConfig struct {
	// ValidKeys is the list of accepted API keys.
	ValidKeys []string
}

// ─── JWT Middleware ──────────────────────────────────────────────

// Auth creates a Fiber middleware that extracts and verifies Bearer tokens.
// ใช้แปะที่ route group เพื่อบังคับ JWT authentication.
//
// Flow:
//  1. Extract "Bearer {token}" from Authorization header
//  2. Verify token via TokenSigner.VerifyAccessToken
//  3. Inject userID + roles into fiber.Ctx.Locals
//  4. Call next handler
func Auth(cfg AuthConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// extract Bearer token
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return dto.ErrorWithTrace(c, fiber.StatusUnauthorized, "missing authorization header", dto.TraceAuthNoHeader)
		}

		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			return dto.ErrorWithTrace(c, fiber.StatusUnauthorized, "invalid authorization format", dto.TraceAuthNoHeader)
		}

		tokenString := authHeader[len(bearerPrefix):]
		if tokenString == "" {
			return dto.ErrorWithTrace(c, fiber.StatusUnauthorized, "empty token", dto.TraceAuthNoHeader)
		}

		// verify token
		claims, err := cfg.TokenSigner.VerifyAccessToken(c.UserContext(), tokenString)
		if err != nil {
			log.L().Warn().Err(err).Str("path", c.Path()).Msg("token verification failed")
			return dto.ErrorWithTrace(c, fiber.StatusUnauthorized, "invalid or expired token", dto.TraceAuthVerifyFailed)
		}

		// inject claims into context
		c.Locals(CtxUserID, claims.UserID)
		c.Locals(CtxRoles, claims.Roles)

		return c.Next()
	}
}

// ─── API Key Middleware ──────────────────────────────────────────

// APIKey creates a Fiber middleware that validates X-API-Key header.
// ใช้แปะที่ route group สำหรับ service-to-service หรือ partner calls.
//
// Flow:
//  1. Extract X-API-Key header
//  2. Compare against ValidKeys using constant-time comparison
//  3. Call next handler
func APIKey(cfg APIKeyConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Get("X-API-Key")
		if key == "" {
			return dto.ErrorWithTrace(c, fiber.StatusUnauthorized, "missing X-API-Key header", dto.TraceAuthAPIKeyMissing)
		}

		if !matchAPIKey(key, cfg.ValidKeys) {
			log.L().Warn().Str("path", c.Path()).Msg("invalid API key")
			return dto.ErrorWithTrace(c, fiber.StatusUnauthorized, "invalid API key", dto.TraceAuthAPIKeyInvalid)
		}

		return c.Next()
	}
}

// matchAPIKey checks if key matches any entry using constant-time comparison.
func matchAPIKey(key string, validKeys []string) bool {
	for _, vk := range validKeys {
		if subtle.ConstantTimeCompare([]byte(key), []byte(vk)) == 1 {
			return true
		}
	}
	return false
}
