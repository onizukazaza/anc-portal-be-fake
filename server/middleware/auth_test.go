package middleware_test

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/ports"
	mw "github.com/onizukazaza/anc-portal-be-fake/server/middleware"
)

// ── Fake TokenSigner ─────────────────────────────────────────────

type fakeTokenSigner struct {
	claims *ports.Claims
	err    error
}

func (f *fakeTokenSigner) SignAccessToken(_ context.Context, _ string, _ []string) (string, error) {
	return "fake-token", nil
}

func (f *fakeTokenSigner) VerifyAccessToken(_ context.Context, _ string) (*ports.Claims, error) {
	return f.claims, f.err
}

// ── Helpers ──────────────────────────────────────────────────────

func setupJWTApp(signer ports.TokenSigner) *fiber.App {
	app := fiber.New()
	app.Use(mw.Auth(mw.AuthConfig{TokenSigner: signer}))
	app.Get("/v1/protected", func(c *fiber.Ctx) error {
		userID := c.Locals(mw.CtxUserID)
		return c.JSON(fiber.Map{"userID": userID})
	})
	return app
}

func setupAPIKeyApp(validKeys []string) *fiber.App {
	app := fiber.New()
	app.Use(mw.APIKey(mw.APIKeyConfig{ValidKeys: validKeys}))
	app.Get("/v1/data", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	return app
}

func doRequest(t *testing.T, app *fiber.App, method, path, headerKey, headerVal string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if headerKey != "" {
		req.Header.Set(headerKey, headerVal)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

// ── JWT Auth Tests ───────────────────────────────────────────────

func TestAuth_MissingHeader(t *testing.T) {
	app := setupJWTApp(&fakeTokenSigner{})

	status, _ := doRequest(t, app, "GET", "/v1/protected", "", "")
	if status != 401 {
		t.Errorf("missing header: status = %d, want 401", status)
	}
}

func TestAuth_InvalidFormat(t *testing.T) {
	app := setupJWTApp(&fakeTokenSigner{})

	status, _ := doRequest(t, app, "GET", "/v1/protected", "Authorization", "Basic abc123")
	if status != 401 {
		t.Errorf("invalid format: status = %d, want 401", status)
	}
}

func TestAuth_EmptyToken(t *testing.T) {
	app := setupJWTApp(&fakeTokenSigner{})

	status, _ := doRequest(t, app, "GET", "/v1/protected", "Authorization", "Bearer ")
	if status != 401 {
		t.Errorf("empty token: status = %d, want 401", status)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	app := setupJWTApp(&fakeTokenSigner{err: fiber.ErrUnauthorized})

	status, _ := doRequest(t, app, "GET", "/v1/protected", "Authorization", "Bearer bad-token")
	if status != 401 {
		t.Errorf("invalid token: status = %d, want 401", status)
	}
}

func TestAuth_ValidToken(t *testing.T) {
	signer := &fakeTokenSigner{
		claims: &ports.Claims{UserID: "user-123", Roles: []string{"admin"}},
	}
	app := setupJWTApp(signer)

	status, body := doRequest(t, app, "GET", "/v1/protected", "Authorization", "Bearer valid-token")
	if status != 200 {
		t.Errorf("valid token: status = %d, want 200", status)
	}
	if body == "" {
		t.Error("expected response body")
	}
}

func TestAuth_RouteLevel_PublicAndProtected(t *testing.T) {
	signer := &fakeTokenSigner{
		claims: &ports.Claims{UserID: "u1", Roles: []string{"admin"}},
	}

	app := fiber.New()
	api := app.Group("/v1")

	// public route — no middleware
	api.Post("/auth/login", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	// protected route — JWT middleware
	protected := api.Group("", mw.Auth(mw.AuthConfig{TokenSigner: signer}))
	protected.Get("/protected", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"userID": c.Locals(mw.CtxUserID)})
	})

	// public route should work without token
	status, _ := doRequest(t, app, "POST", "/v1/auth/login", "", "")
	if status != 200 {
		t.Errorf("public route: status = %d, want 200", status)
	}

	// protected route should require token
	status, _ = doRequest(t, app, "GET", "/v1/protected", "", "")
	if status != 401 {
		t.Errorf("protected without token: status = %d, want 401", status)
	}

	// protected route should work with valid token
	status, _ = doRequest(t, app, "GET", "/v1/protected", "Authorization", "Bearer valid-token")
	if status != 200 {
		t.Errorf("protected with token: status = %d, want 200", status)
	}
}

// ── API Key Tests ────────────────────────────────────────────────

func TestAPIKey_MissingHeader(t *testing.T) {
	app := setupAPIKeyApp([]string{"secret-key-1"})

	status, _ := doRequest(t, app, "GET", "/v1/data", "", "")
	if status != 401 {
		t.Errorf("missing API key: status = %d, want 401", status)
	}
}

func TestAPIKey_InvalidKey(t *testing.T) {
	app := setupAPIKeyApp([]string{"secret-key-1"})

	status, _ := doRequest(t, app, "GET", "/v1/data", "X-API-Key", "wrong-key")
	if status != 401 {
		t.Errorf("invalid API key: status = %d, want 401", status)
	}
}

func TestAPIKey_ValidKey(t *testing.T) {
	app := setupAPIKeyApp([]string{"secret-key-1", "secret-key-2"})

	status, _ := doRequest(t, app, "GET", "/v1/data", "X-API-Key", "secret-key-2")
	if status != 200 {
		t.Errorf("valid API key: status = %d, want 200", status)
	}
}

func TestAPIKey_RouteLevel(t *testing.T) {
	app := fiber.New()
	api := app.Group("/v1")

	// API key protected
	partner := api.Group("/partner", mw.APIKey(mw.APIKeyConfig{ValidKeys: []string{"pk-123"}}))
	partner.Get("/data", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	// public route
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// health should work without key
	status, _ := doRequest(t, app, "GET", "/v1/health", "", "")
	if status != 200 {
		t.Errorf("public route: status = %d, want 200", status)
	}

	// partner without key should fail
	status, _ = doRequest(t, app, "GET", "/v1/partner/data", "", "")
	if status != 401 {
		t.Errorf("partner without key: status = %d, want 401", status)
	}

	// partner with valid key should work
	status, _ = doRequest(t, app, "GET", "/v1/partner/data", "X-API-Key", "pk-123")
	if status != 200 {
		t.Errorf("partner with key: status = %d, want 200", status)
	}
}
