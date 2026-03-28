package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func TestAccessLog_LogsRequest(t *testing.T) {
	app := fiber.New()
	app.Use(requestid.New())
	app.Use(AccessLog())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAccessLog_SkipPaths(t *testing.T) {
	app := fiber.New()
	app.Use(AccessLog(AccessLogConfig{
		SkipPaths: []string{"/healthz", "/ready", "/metrics"},
	}))

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/api/data", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"data": "value"})
	})

	// health endpoint should still work (just not logged — no way to assert log output here)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// api endpoint should also work
	req2 := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	resp2, err := app.Test(req2, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	data, _ := io.ReadAll(resp2.Body)
	var result map[string]any
	_ = json.Unmarshal(data, &result)
	if result["data"] != "value" {
		t.Errorf("expected 'value', got %v", result["data"])
	}
}

func TestAccessLog_ErrorStatusLogged(t *testing.T) {
	app := fiber.New()
	app.Use(AccessLog())

	app.Get("/fail", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "boom"})
	})

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}
