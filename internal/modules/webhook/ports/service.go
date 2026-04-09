package ports

import "context"

// WebhookService handles webhook processing logic.
type WebhookService interface {
	// HandlePush processes a GitHub push webhook event.
	HandlePush(ctx context.Context, rawBody []byte, signatureHeader string) error
}
