package enum

// ===================================================================
// DB Health Status — สถานะการเชื่อมต่อ external database
// ===================================================================

const (
	// DBHealthy database เชื่อมต่อสำเร็จ
	DBHealthy = "healthy"

	// DBUnhealthy database เชื่อมต่อได้แต่ query ล้มเหลว
	DBUnhealthy = "unhealthy"

	// DBError database เชื่อมต่อไม่ได้เลย (pool not found, etc.)
	DBError = "error"
)

// ===================================================================
// Health Check Status — สถานะของ /healthz, /ready endpoints
// ===================================================================

const (
	// HealthOK ใช้ใน /healthz response
	HealthOK = "ok"

	// HealthReady ใช้ใน /ready response เมื่อทุก dependency พร้อม
	HealthReady = "ready"

	// HealthNotReady ใช้ใน /ready response เมื่อ dependency ยังไม่พร้อม
	HealthNotReady = "not_ready"
)
