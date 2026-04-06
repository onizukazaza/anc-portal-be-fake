package middleware

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

// ===================================================================
// Mock Middleware — intercept request แล้วตอบ JSON จาก mockdata/
// ===================================================================
//
// ใช้สำหรับ FE development — เปิด/ปิดจาก config:
//
//	mock:
//	  enabled: true
//	  routesFile: "mockdata/routes.json"
//
// FE เรียก URL เดิม (เช่น GET /v1/cmi/:job_id/...) → ได้ mock response กลับ
// ถ้า route ไม่ match → ส่งต่อไป handler จริง (fall-through)

// ─── Route Definition ────────────────────────────────────────────

// MockRoute คือ entry หนึ่งใน routes.json ที่จับคู่ request กับ mock file
type MockRoute struct {
	Method string `json:"method"` // HTTP method: GET, POST, PUT, DELETE
	Path   string `json:"path"`   // Fiber-style path: /v1/cmi/:job_id/request-policy-single-cmi
	File   string `json:"file"`   // path สัมพัทธ์จาก mockdata/: cmi/get_policy_success.json
}

// ─── Config ──────────────────────────────────────────────────────

// MockConfig holds configuration for the mock middleware.
type MockConfig struct {
	RoutesFile string // path ไปยัง routes.json (เช่น "mockdata/routes.json")
}

// ─── Middleware ───────────────────────────────────────────────────

// Mock creates a Fiber middleware that intercepts matching requests
// and responds with pre-defined JSON from mockdata/ files.
//
// Features:
//   - จับคู่ HTTP method + path pattern (รองรับ :param)
//   - ส่ง status code ตาม status_code ใน JSON
//   - เพิ่ม header X-Mock: true ให้ FE ตรวจสอบได้
//   - Fall-through ไป handler จริงถ้า route ไม่ match
func Mock(cfg MockConfig) fiber.Handler {
	baseDir := filepath.Dir(cfg.RoutesFile)

	// โหลด routes.json ครั้งเดียวตอน startup
	routes := loadMockRoutes(cfg.RoutesFile)

	// cache file content เพื่อไม่ต้องอ่านซ้ำทุก request
	var mu sync.RWMutex
	fileCache := make(map[string][]byte)

	return func(c *fiber.Ctx) error {
		method := c.Method()
		reqPath := c.Path()

		for _, r := range routes {
			if !strings.EqualFold(method, r.Method) {
				continue
			}
			if !matchPath(reqPath, r.Path) {
				continue
			}

			data, err := readMockFile(&mu, fileCache, baseDir, r.File)
			if err != nil {
				log.L().Warn().Err(err).
					Str("mock_file", r.File).
					Msg("mock: cannot read file, falling through to real handler")
				return c.Next()
			}

			statusCode := extractStatusCode(data)

			c.Set("X-Mock", "true")
			c.Set("Content-Type", "application/json")
			return c.Status(statusCode).Send(data)
		}

		// ไม่ match route ใดเลย → ส่งต่อไป handler จริง
		return c.Next()
	}
}

// ─── Internal Helpers ────────────────────────────────────────────

// loadMockRoutes อ่าน routes.json แล้ว unmarshal เป็น []MockRoute
func loadMockRoutes(path string) []MockRoute {
	data, err := os.ReadFile(path)
	if err != nil {
		log.L().Error().Err(err).Str("path", path).Msg("mock: cannot read routes file")
		return nil
	}

	var routes []MockRoute
	if err := json.Unmarshal(data, &routes); err != nil {
		log.L().Error().Err(err).Str("path", path).Msg("mock: cannot parse routes file")
		return nil
	}

	log.L().Info().Int("count", len(routes)).Str("file", path).Msg("mock: loaded routes")
	return routes
}

// readMockFile reads and caches a mock JSON file.
func readMockFile(mu *sync.RWMutex, cache map[string][]byte, baseDir, file string) ([]byte, error) {
	// ป้องกัน path traversal
	cleaned := filepath.Clean(file)
	if strings.Contains(cleaned, "..") {
		return nil, os.ErrPermission
	}

	mu.RLock()
	data, ok := cache[cleaned]
	mu.RUnlock()
	if ok {
		return data, nil
	}

	fullPath := filepath.Join(baseDir, cleaned)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	mu.Lock()
	cache[cleaned] = data
	mu.Unlock()

	return data, nil
}

// matchPath เปรียบเทียบ request path กับ pattern ที่มี :param
//
//	matchPath("/v1/cmi/JOB-001/request-policy-single-cmi",
//	          "/v1/cmi/:job_id/request-policy-single-cmi") → true
func matchPath(reqPath, pattern string) bool {
	reqParts := splitPath(reqPath)
	patParts := splitPath(pattern)

	if len(reqParts) != len(patParts) {
		return false
	}

	for i, pat := range patParts {
		if strings.HasPrefix(pat, ":") {
			continue // wildcard param — match ทุกค่า
		}
		if !strings.EqualFold(reqParts[i], pat) {
			return false
		}
	}
	return true
}

// splitPath แยก path เป็น segments โดย trim leading/trailing slash
func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// extractStatusCode ดึง status_code จาก JSON body
// ถ้าหาไม่เจอ → default 200
func extractStatusCode(data []byte) int {
	var envelope struct {
		StatusCode int `json:"status_code"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fiber.StatusOK
	}
	if envelope.StatusCode == 0 {
		return fiber.StatusOK
	}
	return envelope.StatusCode
}
