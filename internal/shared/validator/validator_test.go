package validator

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
)

// ───────────────────────────────────────────────────────────────────
// Test Structs
// ───────────────────────────────────────────────────────────────────

type testReq struct {
	Name  string `json:"name"  validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"gte=1,lte=120"`
}

// ───────────────────────────────────────────────────────────────────
// Helpers
// ───────────────────────────────────────────────────────────────────

func doTestRequest(t *testing.T, app *fiber.App, body string) (int, dto.ApiResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	var result dto.ApiResponse
	if len(data) > 0 {
		_ = json.Unmarshal(data, &result)
	}

	return resp.StatusCode, result
}

// ───────────────────────────────────────────────────────────────────
// Tests — BindAndValidate
// ───────────────────────────────────────────────────────────────────

func TestBindAndValidate_Success(t *testing.T) {
	app := fiber.New()
	app.Post("/test", func(c *fiber.Ctx) error {
		var req testReq
		if err := BindAndValidate(c, &req); err != nil {
			return nil
		}
		return c.JSON(fiber.Map{"name": req.Name})
	})

	status, _ := doTestRequest(t, app, `{"name":"John","email":"john@example.com","age":25}`)
	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}
}

func TestBindAndValidate_InvalidJSON(t *testing.T) {
	app := fiber.New()
	app.Post("/test", func(c *fiber.Ctx) error {
		var req testReq
		if err := BindAndValidate(c, &req); err != nil {
			return nil
		}
		return c.JSON(fiber.Map{"name": req.Name})
	})

	status, result := doTestRequest(t, app, `{invalid json}`)
	if status != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", status)
	}
	if result.Message != "invalid request body" {
		t.Errorf("expected 'invalid request body', got %q", result.Message)
	}
}

func TestBindAndValidate_ValidationFail(t *testing.T) {
	app := fiber.New()
	app.Post("/test", func(c *fiber.Ctx) error {
		var req testReq
		if err := BindAndValidate(c, &req); err != nil {
			return nil
		}
		return c.JSON(fiber.Map{"name": req.Name})
	})

	// missing name, bad email
	status, result := doTestRequest(t, app, `{"email":"not-an-email","age":0}`)
	if status != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", status)
	}
	if result.Message != "validation failed" {
		t.Errorf("expected 'validation failed', got %q", result.Message)
	}
}

// ───────────────────────────────────────────────────────────────────
// Tests — FormatErrors
// ───────────────────────────────────────────────────────────────────

func TestFormatErrors_RequiredTag(t *testing.T) {
	v := Get()
	err := v.Struct(testReq{})
	errs := FormatErrors(err)
	if len(errs) < 2 {
		t.Fatalf("expected at least 2 errors, got %d", len(errs))
	}

	// Check that fields have human-readable messages
	found := false
	for _, e := range errs {
		if e.Field == "name" && e.Message == "field is required" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'name' field with 'field is required' message")
	}
}

func TestGet_ReturnsSameInstance(t *testing.T) {
	v1 := Get()
	v2 := Get()
	if v1 != v2 {
		t.Error("expected same validator instance (singleton)")
	}
}
