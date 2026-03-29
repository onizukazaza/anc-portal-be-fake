package discord

import "errors"

// ErrWebhookNotConfigured indicates the Discord webhook URL is empty.
var ErrWebhookNotConfigured = errors.New("discord: webhook URL not configured")
