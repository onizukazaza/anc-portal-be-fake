package ports

import "github.com/gofiber/fiber/v2"

// WebhookController defines delivery contract for webhook endpoints.
type WebhookController interface {
	HandleGitHubPush(ctx *fiber.Ctx) error
}
