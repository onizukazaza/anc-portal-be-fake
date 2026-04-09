package ports

import "github.com/gofiber/fiber/v2"

// AuthController defines delivery contract for auth endpoints.
type AuthController interface {
	Login(ctx *fiber.Ctx) error
}
