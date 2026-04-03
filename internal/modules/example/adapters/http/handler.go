package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/example/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
)

// GetByID handles GET /examples/:id
func (ctrl *Controller) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return dto.Error(c, fiber.StatusBadRequest, "id is required")
	}

	item, err := ctrl.service.GetByID(c.UserContext(), id)
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			return dto.Error(c, fiber.StatusNotFound, "example not found")
		}
		return dto.Error(c, fiber.StatusInternalServerError, "internal error")
	}

	return dto.Success(c, fiber.StatusOK, item)
}

// List handles GET /examples
func (ctrl *Controller) List(c *fiber.Ctx) error {
	items, err := ctrl.service.List(c.UserContext())
	if err != nil {
		return dto.Error(c, fiber.StatusInternalServerError, "internal error")
	}

	return dto.Success(c, fiber.StatusOK, items)
}
