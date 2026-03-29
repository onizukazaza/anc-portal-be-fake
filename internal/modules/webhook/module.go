package webhook

import (
	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/webhook/adapters/discord"
	webhookhttp "github.com/onizukazaza/anc-portal-be-fake/internal/modules/webhook/adapters/http"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/webhook/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/module"
	pkgdiscord "github.com/onizukazaza/anc-portal-be-fake/pkg/discord"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

// Register wires webhook module dependencies and mounts routes.
func Register(router fiber.Router, deps module.Deps) {
	cfg := deps.Config.Webhook
	if !cfg.Enabled {
		log.L().Info().Msg("webhook module disabled")
		return
	}

	if cfg.DiscordWebhookURL == "" {
		log.L().Warn().Msg("webhook module enabled but DISCORD_WEBHOOK_URL is empty — skipping")
		return
	}

	discordClient := pkgdiscord.NewClient(cfg.DiscordWebhookURL)
	notifier := discord.NewNotifier(discordClient)
	service := app.NewService(cfg.GitHubSecret, notifier)
	controller := webhookhttp.NewWebhookController(service)

	group := router.Group("/webhooks")
	group.Post("/github", controller.HandleGitHubPush)
}
