// Package validator ครอบ go-playground/validator/v10 เป็น singleton
// ให้ทุก handler ใช้ instance เดียวกัน — สร้างครั้งเดียวตอน init
//
// วิธีใช้:
//
//	type createOrderReq struct {
//	    CustomerID string `json:"customer_id" validate:"required"`
//	    Amount     int    `json:"amount"      validate:"required,gt=0"`
//	}
//
//	func (h *Handler) CreateOrder(c *fiber.Ctx) error {
//	    var req createOrderReq
//	    if err := validator.BindAndValidate(c, &req); err != nil {
//	        return err // ส่ง 422 พร้อม field-level errors อัตโนมัติ
//	    }
//	    // req พร้อมใช้งาน
//	}
package validator

import (
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	instance *validator.Validate
	once     sync.Once
)

// Get คืน singleton validator instance
func Get() *validator.Validate {
	once.Do(func() {
		instance = validator.New(validator.WithRequiredStructEnabled())
	})
	return instance
}

// FieldError คือ error ของแต่ละ field ที่ validation fail
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// FormatErrors แปลง validator.ValidationErrors เป็น slice ของ FieldError
// ใช้ชื่อ field จาก JSON tag (lowercase) แทน Go struct field name
func FormatErrors(err error) []FieldError {
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return []FieldError{{Field: "_", Message: err.Error()}}
	}

	out := make([]FieldError, 0, len(ve))
	for _, fe := range ve {
		out = append(out, FieldError{
			Field:   jsonFieldName(fe),
			Message: buildMessage(fe),
		})
	}
	return out
}

// jsonFieldName คืนชื่อ field จาก json tag ถ้ามี ไม่งั้นใช้ struct field name (lowercase)
func jsonFieldName(fe validator.FieldError) string {
	ns := fe.Namespace()
	// ตัดชื่อ struct ออก เช่น "CreateOrderReq.CustomerID" → "CustomerID"
	if idx := strings.Index(ns, "."); idx >= 0 {
		ns = ns[idx+1:]
	}
	return strings.ToLower(ns)
}

// buildMessage สร้าง human-readable message จาก validation tag
func buildMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "field is required"
	case "email":
		return "must be a valid email"
	case "min":
		return "must be at least " + fe.Param()
	case "max":
		return "must be at most " + fe.Param()
	case "gt":
		return "must be greater than " + fe.Param()
	case "gte":
		return "must be greater than or equal to " + fe.Param()
	case "lt":
		return "must be less than " + fe.Param()
	case "lte":
		return "must be less than or equal to " + fe.Param()
	case "len":
		return "must have length " + fe.Param()
	case "oneof":
		return "must be one of: " + fe.Param()
	case "uuid":
		return "must be a valid UUID"
	case "url":
		return "must be a valid URL"
	default:
		return "failed on '" + fe.Tag() + "' validation"
	}
}
