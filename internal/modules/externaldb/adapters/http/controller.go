package http

import "github.com/gofiber/fiber/v2"

// ExternalDBController defines delivery contract for external-db diagnostic endpoints.
type ExternalDBController interface {
	CheckAll(ctx *fiber.Ctx) error
	CheckByName(ctx *fiber.Ctx) error
}
