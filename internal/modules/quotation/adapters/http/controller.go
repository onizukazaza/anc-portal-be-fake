package http

import "github.com/gofiber/fiber/v2"

// QuotationController defines delivery contract for quotation endpoints.
type QuotationController interface {
	GetByID(ctx *fiber.Ctx) error
	ListByCustomer(ctx *fiber.Ctx) error
}
