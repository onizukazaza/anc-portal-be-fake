package validator

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
)

// ErrValidation ใช้ signal ว่า BindAndValidate ล้มเหลว (response ถูกเขียนแล้ว)
// handler ที่เรียกควร return nil ทันที เพื่อไม่ให้ Fiber error handler เขียนทับ
var ErrValidation = errors.New("validation: response already sent")

// BindAndValidate parse JSON body เข้า dest แล้ว validate ด้วย struct tags
// ถ้า parse หรือ validate fail → เขียน JSON error response แล้ว return ErrValidation
//
// Status codes:
//   - 400 Bad Request  → JSON body อ่านไม่ได้ / format ผิด
//   - 422 Unprocessable Entity → field validation fail (พร้อม field-level errors)
//
// Usage:
//
//	if err := validator.BindAndValidate(c, &req); err != nil {
//	    return nil // response already sent
//	}
func BindAndValidate(c *fiber.Ctx, dest any) error {
	if err := c.BodyParser(dest); err != nil {
		_ = dto.Error(c, fiber.StatusBadRequest, "invalid request body")
		return ErrValidation
	}

	if err := Get().Struct(dest); err != nil {
		_ = c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ApiResponse{
			Status:     "ERROR",
			StatusCode: fiber.StatusUnprocessableEntity,
			Message:    "validation failed",
			Result:     FormatErrors(err),
		})
		return ErrValidation
	}

	return nil
}
