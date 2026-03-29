package discord

import (
	"context"
	"fmt"
	"strings"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/webhook/domain"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/discord"
)

// Notifier sends GitHub event notifications to Discord.
type Notifier struct {
	client *discord.Client
}

// NewNotifier creates a Discord notifier.
func NewNotifier(client *discord.Client) *Notifier {
	return &Notifier{client: client}
}

// NotifyPush sends a formatted push event notification to Discord.
func (n *Notifier) NotifyPush(ctx context.Context, event *domain.GitHubPushEvent) error {
	ctx, span := appOtel.Tracer(appOtel.TracerWebhookNotifier).Start(ctx, "Discord.NotifyPush")
	defer span.End()

	embed := n.buildPushEmbed(event)
	return n.client.SendEmbed(ctx, embed)
}

func (n *Notifier) buildPushEmbed(event *domain.GitHubPushEvent) discord.Embed {
	branch := event.BranchName()
	repo := event.Repository.FullName
	pusher := event.Pusher.Name
	commitCount := len(event.Commits)

	title := fmt.Sprintf("🚀 New push to %s", branch)

	embed := discord.NewEmbed(title, n.branchColor(branch)).
		WithURL(event.Compare).
		WithField("Repository", fmt.Sprintf("[%s](%s)", repo, event.Repository.HTMLURL), true).
		WithField("Branch", fmt.Sprintf("`%s`", branch), true).
		WithField("Pushed by", pusher, true).
		WithField("Commits", fmt.Sprintf("%d commit(s)", commitCount), true).
		WithFooter("ANC Portal — GitHub Webhook")

	// Add sender avatar if available
	if event.Sender.AvatarURL != "" {
		embed = embed.WithAuthor(
			event.Sender.Login,
			event.Sender.HTMLURL,
			event.Sender.AvatarURL,
		)
	}

	// Show commit details (max 5)
	if len(event.Commits) > 0 {
		var lines []string
		limit := len(event.Commits)
		if limit > 5 {
			limit = 5
		}
		for _, c := range event.Commits[:limit] {
			msg := c.Message
			if idx := strings.Index(msg, "\n"); idx > 0 {
				msg = msg[:idx]
			}
			if len(msg) > 72 {
				msg = msg[:72] + "..."
			}
			lines = append(lines, fmt.Sprintf("[`%s`](%s) %s — %s",
				domain.ShortSHA(c.ID), c.URL, msg, c.Author.Username))
		}
		if len(event.Commits) > 5 {
			lines = append(lines, fmt.Sprintf("... and %d more", len(event.Commits)-5))
		}
		embed = embed.WithField("Changes", strings.Join(lines, "\n"), false)
	}

	return embed
}

func (n *Notifier) branchColor(branch string) int {
	switch branch {
	case "main", "master":
		return discord.ColorRed
	case "develop":
		return discord.ColorGreen
	default:
		return discord.ColorBlue
	}
}
