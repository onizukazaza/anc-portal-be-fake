// Package pagination ให้ struct และ helper สำหรับ pagination, sorting, search
// ที่ใช้ร่วมกันได้ทุก module — ไม่ผูกกับ framework หรือ business logic ใด ๆ
//
// โครงสร้างหลัก:
//   - Request    รับค่า page, limit, sort, order, search จาก client
//   - Response[T] ครอบ result set พร้อม metadata (total, totalPages, hasNext, hasPrev)
//   - SQLClause   สร้าง ORDER BY + LIMIT + OFFSET (อยู่ใน sql.go)
//   - FromFiber   parse query string จาก Fiber context (อยู่ใน fiber.go)
//
// วิธีใช้ (ตัวอย่าง quotation module):
//
//	// 1) Handler — parse จาก query string
//	pg := pagination.FromFiber(c)
//
//	// 2) Repository — ใช้ SQL helpers
//	pagination.CountQuery("quotations", "WHERE customer_id = $1")
//	pagination.SQLClause(pg, "created_at", allowedSorts)
//
//	// 3) Service — wrap response
//	pagination.NewResponse(items, total, pg)
package pagination

import "math"

// ===================================================================
// Request — รับค่า pagination + sorting + search จาก client
// ===================================================================

// Request เก็บ parameters สำหรับ pagination ที่ parse มาจาก client
//
// Query string ที่รองรับ:
//
//	?page=2&limit=10&sort=created_at&order=asc&search=keyword
//
// ค่า default (ถ้าไม่ส่งมา):
//   - page  = 1
//   - limit = 20 (สูงสุด 100)
//   - order = "desc"
type Request struct {
	Page   int    `json:"page"    query:"page"`   // หน้าที่ต้องการ (เริ่มจาก 1)
	Limit  int    `json:"limit"   query:"limit"`  // จำนวนรายการต่อหน้า (1-100)
	Sort   string `json:"sort"    query:"sort"`   // คอลัมน์ที่ใช้ sort เช่น "created_at"
	Order  string `json:"order"   query:"order"`  // ทิศทาง: "asc" หรือ "desc"
	Search string `json:"search"  query:"search"` // คำค้นหา (free-text)
}

// Defaults ตั้งค่า default ให้ field ที่ไม่ได้ส่งมาหรือค่าไม่ถูกต้อง
//   - page < 1       → 1
//   - limit < 1      → 20
//   - limit > 100    → 100  (ป้องกัน client ดึงข้อมูลมากเกินไป)
//   - order ไม่ใช่ asc/desc → "desc"
func (r *Request) Defaults() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.Limit < 1 {
		r.Limit = 20
	}
	if r.Limit > 100 {
		r.Limit = 100
	}
	if r.Order != "asc" && r.Order != "desc" {
		r.Order = "desc"
	}
}

// Offset คำนวณ SQL OFFSET จาก page และ limit
//
//	page=1, limit=20 → offset=0
//	page=2, limit=20 → offset=20
//	page=3, limit=10 → offset=20
func (r *Request) Offset() int {
	return (r.Page - 1) * r.Limit
}

// ===================================================================
// Response[T] — generic paginated response ที่ครอบ result set
// ===================================================================

// Response ครอบข้อมูลที่ paginate แล้วพร้อม metadata
// ใช้ generic type T เพื่อรองรับ domain model ใดก็ได้
//
// ตัวอย่าง JSON response:
//
//	{
//	  "items": [...],
//	  "total": 58,
//	  "page": 2,
//	  "limit": 20,
//	  "totalPages": 3,
//	  "hasNext": true,
//	  "hasPrev": true
//	}
type Response[T any] struct {
	Items      []T  `json:"items"`      // รายการข้อมูลของหน้าปัจจุบัน
	Total      int  `json:"total"`      // จำนวนรายการทั้งหมด (ก่อน paginate)
	Page       int  `json:"page"`       // หน้าปัจจุบัน
	Limit      int  `json:"limit"`      // จำนวนรายการต่อหน้า
	TotalPages int  `json:"totalPages"` // จำนวนหน้าทั้งหมด
	HasNext    bool `json:"hasNext"`    // มีหน้าถัดไปหรือไม่
	HasPrev    bool `json:"hasPrev"`    // มีหน้าก่อนหน้าหรือไม่
}

// NewResponse สร้าง paginated response จาก:
//   - items: ข้อมูลของหน้าปัจจุบัน
//   - total: จำนวนทั้งหมดจาก COUNT query
//   - req:   pagination request ที่ใช้
//
// ถ้า items เป็น nil จะแปลงเป็น empty slice [] (ไม่ส่ง null ใน JSON)
func NewResponse[T any](items []T, total int, req Request) Response[T] {
	if items == nil {
		items = []T{}
	}
	if req.Limit < 1 {
		req.Limit = 1
	}
	totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))
	return Response[T]{
		Items:      items,
		Total:      total,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
		HasPrev:    req.Page > 1,
	}
}
