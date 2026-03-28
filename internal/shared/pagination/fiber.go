package pagination

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// FromFiber parse pagination parameters จาก Fiber query string
// แล้วเรียก Defaults() ให้อัตโนมัติ
//
// Query string ที่รองรับ:
//
//	?page=1&limit=20&sort=created_at&order=desc&search=keyword
//
// ตัวอย่างการใช้ใน handler:
//
//	func (h *Handler) List(c *fiber.Ctx) error {
//	    pg := pagination.FromFiber(c)
//	    result, err := h.service.List(c.UserContext(), pg)
//	    ...
//	}
//
// หมายเหตุ: ถ้าต้องการใช้กับ framework อื่น (echo, chi, net/http)
// สร้างไฟล์ใหม่ เช่น echo.go แล้ว return Request เหมือนกัน
func FromFiber(c *fiber.Ctx) Request {
	req := Request{
		Page:   queryInt(c, "page", 1),
		Limit:  queryInt(c, "limit", 20),
		Sort:   c.Query("sort"),
		Order:  c.Query("order", "desc"),
		Search: c.Query("search"),
	}
	req.Defaults()
	return req
}

// queryInt ดึงค่า int จาก query string พร้อม fallback ถ้า parse ไม่ได้
func queryInt(c *fiber.Ctx, key string, fallback int) int {
	v := c.Query(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
