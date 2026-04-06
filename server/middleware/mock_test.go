package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// ─── matchPath ───────────────────────────────────────────────────

func TestMatchPath(t *testing.T) {
	tests := []struct {
		name    string
		req     string
		pattern string
		want    bool
	}{
		{
			name:    "exact match",
			req:     "/v1/auth/login",
			pattern: "/v1/auth/login",
			want:    true,
		},
		{
			name:    "param match",
			req:     "/v1/cmi/JOB-001/request-policy-single-cmi",
			pattern: "/v1/cmi/:job_id/request-policy-single-cmi",
			want:    true,
		},
		{
			name:    "different length",
			req:     "/v1/auth",
			pattern: "/v1/auth/login",
			want:    false,
		},
		{
			name:    "mismatch segment",
			req:     "/v1/cmi/JOB-001/wrong-path",
			pattern: "/v1/cmi/:job_id/request-policy-single-cmi",
			want:    false,
		},
		{
			name:    "case insensitive",
			req:     "/V1/Auth/Login",
			pattern: "/v1/auth/login",
			want:    true,
		},
		{
			name:    "trailing slash ignored",
			req:     "/v1/auth/login/",
			pattern: "/v1/auth/login",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPath(tt.req, tt.pattern)
			if got != tt.want {
				t.Errorf("matchPath(%q, %q) = %v, want %v", tt.req, tt.pattern, got, tt.want)
			}
		})
	}
}

// ─── extractStatusCode ───────────────────────────────────────────

func TestExtractStatusCode(t *testing.T) {
	tests := []struct {
		name string
		data string
		want int
	}{
		{
			name: "200 from json",
			data: `{"status":"OK","status_code":200}`,
			want: 200,
		},
		{
			name: "404 from json",
			data: `{"status":"ERROR","status_code":404}`,
			want: 404,
		},
		{
			name: "missing status_code defaults to 200",
			data: `{"status":"OK"}`,
			want: 200,
		},
		{
			name: "invalid json defaults to 200",
			data: `not-json`,
			want: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractStatusCode([]byte(tt.data))
			if got != tt.want {
				t.Errorf("extractStatusCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

// ─── Mock middleware integration ─────────────────────────────────

func TestMockMiddleware(t *testing.T) {
	// สร้าง temp directory เป็น mockdata
	dir := t.TempDir()

	// สร้าง mock JSON file
	authDir := filepath.Join(dir, "auth")
	if err := os.MkdirAll(authDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mockJSON := `{"status":"OK","status_code":200,"message":"success","result":{"data":{"accessToken":"mock-token"}}}`
	if err := os.WriteFile(filepath.Join(authDir, "login.json"), []byte(mockJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	// สร้าง routes.json
	routesJSON := `[{"method":"POST","path":"/v1/auth/login","file":"auth/login.json"}]`
	routesFile := filepath.Join(dir, "routes.json")
	if err := os.WriteFile(routesFile, []byte(routesJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	app := fiber.New()
	app.Use(Mock(MockConfig{RoutesFile: routesFile}))

	// fallback handler สำหรับ route ที่ไม่ match
	app.All("/*", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusTeapot).SendString("real handler")
	})

	t.Run("matching route returns mock", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if resp.Header.Get("X-Mock") != "true" {
			t.Error("missing X-Mock header")
		}

		body, _ := io.ReadAll(resp.Body)
		if string(body) != mockJSON {
			t.Errorf("body = %s, want %s", body, mockJSON)
		}
	})

	t.Run("non-matching route falls through", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/unknown", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusTeapot {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusTeapot)
		}
		if resp.Header.Get("X-Mock") != "" {
			t.Error("X-Mock header should not be present for non-mock routes")
		}
	})

	t.Run("wrong method does not match", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/auth/login", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusTeapot {
			t.Errorf("status = %d, want %d (should fall through)", resp.StatusCode, http.StatusTeapot)
		}
	})
}

func TestMockMiddleware_ErrorStatusCode(t *testing.T) {
	dir := t.TempDir()

	cmiDir := filepath.Join(dir, "cmi")
	if err := os.MkdirAll(cmiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	errorJSON := `{"status":"ERROR","status_code":404,"message":"job not found","result":{"trace_id":"cmi-job-not-found"}}`
	if err := os.WriteFile(filepath.Join(cmiDir, "not_found.json"), []byte(errorJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	routesJSON := `[{"method":"GET","path":"/v1/cmi/:job_id/request-policy-single-cmi","file":"cmi/not_found.json"}]`
	routesFile := filepath.Join(dir, "routes.json")
	if err := os.WriteFile(routesFile, []byte(routesJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	app := fiber.New()
	app.Use(Mock(MockConfig{RoutesFile: routesFile}))

	req := httptest.NewRequest(http.MethodGet, "/v1/cmi/JOB-999/request-policy-single-cmi", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestMockMiddleware_PathTraversal(t *testing.T) {
	dir := t.TempDir()

	// route ที่พยายามทำ path traversal
	routesJSON := `[{"method":"GET","path":"/v1/hack","file":"../../etc/passwd"}]`
	routesFile := filepath.Join(dir, "routes.json")
	if err := os.WriteFile(routesFile, []byte(routesJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	app := fiber.New()
	app.Use(Mock(MockConfig{RoutesFile: routesFile}))
	app.All("/*", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusTeapot).SendString("fallback")
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/hack", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// ควร fall through ไม่ใช่ serve file จาก path traversal
	if resp.StatusCode != http.StatusTeapot {
		t.Errorf("status = %d, want %d (path traversal should be blocked)", resp.StatusCode, http.StatusTeapot)
	}
}

// ─── Enabled / Disabled per-route ────────────────────────────────

func TestMockMiddleware_EnabledField(t *testing.T) {
	dir := t.TempDir()

	authDir := filepath.Join(dir, "auth")
	if err := os.MkdirAll(authDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mockJSON := `{"status":"OK","status_code":200,"message":"success","result":{"data":{"accessToken":"mock-token"}}}`
	if err := os.WriteFile(filepath.Join(authDir, "login.json"), []byte(mockJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	cmiDir := filepath.Join(dir, "cmi")
	if err := os.MkdirAll(cmiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cmiJSON := `{"status":"OK","status_code":200,"message":"success","result":{"data":{"job_id":"JOB-001"}}}`
	if err := os.WriteFile(filepath.Join(cmiDir, "policy.json"), []byte(cmiJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	// auth enabled=true, cmi enabled=false
	routesJSON := `[
		{"method":"POST","path":"/v1/auth/login","file":"auth/login.json","enabled":true},
		{"method":"GET","path":"/v1/cmi/:job_id/request-policy-single-cmi","file":"cmi/policy.json","enabled":false}
	]`
	routesFile := filepath.Join(dir, "routes.json")
	if err := os.WriteFile(routesFile, []byte(routesJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	app := fiber.New()
	app.Use(Mock(MockConfig{RoutesFile: routesFile}))
	app.All("/*", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusTeapot).SendString("real handler")
	})

	t.Run("enabled route returns mock", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if resp.Header.Get("X-Mock") != "true" {
			t.Error("missing X-Mock header")
		}
	})

	t.Run("disabled route falls through to real handler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/cmi/JOB-001/request-policy-single-cmi", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusTeapot {
			t.Errorf("status = %d, want %d (disabled mock should fall through)", resp.StatusCode, http.StatusTeapot)
		}
		if resp.Header.Get("X-Mock") != "" {
			t.Error("X-Mock header should not be present for disabled routes")
		}
	})
}

func TestMockMiddleware_EnabledDefault(t *testing.T) {
	dir := t.TempDir()

	authDir := filepath.Join(dir, "auth")
	if err := os.MkdirAll(authDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mockJSON := `{"status":"OK","status_code":200,"message":"success"}`
	if err := os.WriteFile(filepath.Join(authDir, "login.json"), []byte(mockJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	// ไม่มี enabled field → default ต้องเป็น true
	routesJSON := `[{"method":"POST","path":"/v1/auth/login","file":"auth/login.json"}]`
	routesFile := filepath.Join(dir, "routes.json")
	if err := os.WriteFile(routesFile, []byte(routesJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	app := fiber.New()
	app.Use(Mock(MockConfig{RoutesFile: routesFile}))

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d (no enabled field should default to true)", resp.StatusCode, http.StatusOK)
	}
	if resp.Header.Get("X-Mock") != "true" {
		t.Error("missing X-Mock header")
	}
}

func TestIsEnabled(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name    string
		enabled *bool
		want    bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", &trueVal, true},
		{"explicit false", &falseVal, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := MockRoute{Enabled: tt.enabled}
			if got := r.isEnabled(); got != tt.want {
				t.Errorf("isEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
