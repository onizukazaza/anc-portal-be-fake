package validator

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
)

// ErrValidation is returned when BindAndValidate has already written the
// error response. Handlers should propagate this error directly:
//
//	if err := validator.BindAndValidate(c, &req); err != nil {
//	    return err
//	}
//
// The Fiber custom ErrorHandler (registered in server.go) recognises
// ErrValidation and skips re-writing the response.
var ErrValidation = errors.New("validation: response already sent")

// BindAndValidate parses the JSON body into dest and validates it
// using struct tags. On failure it writes the appropriate JSON error
// response and returns ErrValidation.
//
// Status codes:
//   - 400 Bad Request  → JSON body อ่านไม่ได้ / format ผิด
//   - 422 Unprocessable Entity → field validation fail (พร้อม field-level errors)
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
