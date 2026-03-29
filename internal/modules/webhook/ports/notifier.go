package ports

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/webhook/domain"
)

// Notifier sends notifications for GitHub events.
type Notifier interface {
	NotifyPush(ctx context.Context, event *domain.GitHubPushEvent) error
}
