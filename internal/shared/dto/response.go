// Package dto เก็บ shared struct สำหรับรับ-ส่งข้อมูลผ่าน API ที่ใช้ร่วมกันทุก module
//
// ใช้แทน fiber.Map{...} ที่เขียนซ้ำในทุก handler
// ทำให้ response format เป็นมาตรฐานเดียวกันทั้ง project
//
// Response format:
//
//	{
//	  "status": "OK",
//	  "status_code": 200,
//	  "message": "success",
//	  "result": { "data": { ... } }
//	}
//
// ตัวอย่างการใช้ใน handler:
//
//	// สำเร็จ
//	return dto.Success(c, fiber.StatusOK, quotation)
//
//	// สำเร็จพร้อม message
//	return dto.SuccessWithMessage(c, fiber.StatusCreated, "created successfully", order)
//
//	// error
//	return dto.Error(c, fiber.StatusNotFound, "quotation not found")
package dto

import (
	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/enum"
)

// ===================================================================
// Response structs — มาตรฐาน JSON response ของทั้ง project
// ===================================================================

// ApiResponse คือ response มาตรฐานสำหรับทุกกรณี (ทั้ง success และ error)
//
// Success JSON:
//
//	{"status": "OK", "status_code": 200, "message": "success", "result": {"data": {...}}}
//
// Error JSON:
//
//	{"status": "ERROR", "status_code": 404, "message": "quotation not found", "result": null}
type ApiResponse struct {
	Status     string `json:"status"`      // "OK" หรือ "ERROR"
	StatusCode int    `json:"status_code"` // HTTP status code (200, 201, 400, 500, ...)
	Message    string `json:"message"`     // คำอธิบาย ("success", "created successfully", "not found")
	Result     any    `json:"result"`      // ข้อมูลผลลัพธ์ — null ถ้า error
}

// ResultData ครอบข้อมูลที่ส่งกลับใน result.data
//
//	{"data": { ... }}
type ResultData struct {
	Data any `json:"data"`           // ข้อมูลหลัก
	Meta any `json:"meta,omitempty"` // metadata เพิ่มเติม (optional)
}

// ErrorResult คือ result object เมื่อเกิด error พร้อม trace_id
//
//	{"trace_id": "qt-not-found"}
type ErrorResult struct {
	TraceID string `json:"trace_id" example:"qt-not-found"` // Error trace identifier
}

// ErrorResponse คือ response เมื่อเกิด error (สำหรับ Swagger)
//
//	{"status":"ERROR","status_code":404,"message":"quotation not found","result":{"trace_id":"qt-not-found"}}
type ErrorResponse struct {
	Status     string       `json:"status" example:"ERROR"`                // "ERROR"
	StatusCode int          `json:"status_code" example:"400"`             // HTTP status code
	Message    string       `json:"message" example:"quotation not found"` // Error description
	Result     *ErrorResult `json:"result"`                                // Contains trace_id
}

// ===================================================================
// Helper functions — ลดโค้ดซ้ำในทุก handler
// ===================================================================

// Success ส่ง response สำเร็จพร้อมข้อมูล (message = "success")
//
//	return dto.Success(c, fiber.StatusOK, user)
func Success(c *fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(ApiResponse{
		Status:     enum.ResponseOK,
		StatusCode: status,
		Message:    "success",
		Result:     ResultData{Data: data},
	})
}

// SuccessWithMessage ส่ง response สำเร็จพร้อม message ที่กำหนดเอง
//
//	return dto.SuccessWithMessage(c, fiber.StatusCreated, "created successfully", order)
func SuccessWithMessage(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(ApiResponse{
		Status:     enum.ResponseOK,
		StatusCode: status,
		Message:    message,
		Result:     ResultData{Data: data},
	})
}

// SuccessWithMeta ส่ง response สำเร็จพร้อมข้อมูลและ metadata
//
//	return dto.SuccessWithMeta(c, fiber.StatusOK, items, metaInfo)
func SuccessWithMeta(c *fiber.Ctx, status int, data any, meta any) error {
	return c.Status(status).JSON(ApiResponse{
		Status:     enum.ResponseOK,
		StatusCode: status,
		Message:    "success",
		Result:     ResultData{Data: data, Meta: meta},
	})
}

// Error ส่ง response error พร้อมข้อความ
//
//	return dto.Error(c, fiber.StatusNotFound, "quotation not found")
func Error(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(ApiResponse{
		Status:     enum.ResponseError,
		StatusCode: status,
		Message:    message,
		Result:     nil,
	})
}

// ErrorWithCode ส่ง response error พร้อมข้อความและ error code (ใส่ใน result)
//
//	return dto.ErrorWithCode(c, fiber.StatusUnauthorized, "token expired", "AUTH_TOKEN_EXPIRED")
func ErrorWithCode(c *fiber.Ctx, status int, message string, code string) error {
	return c.Status(status).JSON(ApiResponse{
		Status:     enum.ResponseError,
		StatusCode: status,
		Message:    message,
		Result: map[string]string{
			"code": code,
		},
	})
}

// ErrorWithTrace ส่ง response error พร้อม trace_id สำหรับ debug
//
//	return dto.ErrorWithTrace(c, fiber.StatusNotFound, "quotation not found", dto.TraceQTNotFound)
//
// Response JSON:
//
//	{"status":"ERROR","status_code":404,"message":"quotation not found","result":{"trace_id":"qt-not-found"}}
func ErrorWithTrace(c *fiber.Ctx, status int, message string, traceID string) error {
	return c.Status(status).JSON(ApiResponse{
		Status:     enum.ResponseError,
		StatusCode: status,
		Message:    message,
		Result: map[string]string{
			"trace_id": traceID,
		},
	})
}
