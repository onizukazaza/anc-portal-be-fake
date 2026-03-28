package http

import "github.com/gofiber/fiber/v2"

// CMIController defines delivery contract for CMI endpoints.
type CMIController interface {
	GetPolicyByJobID(ctx *fiber.Ctx) error
}
