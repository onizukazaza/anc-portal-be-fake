package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/webhook/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/webhook/ports"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrUnsupportedEvent = errors.New("unsupported event type")
)

// Service handles incoming GitHub webhook events.
type Service struct {
	secret   string // GitHub webhook secret for HMAC verification
	notifier ports.Notifier
}

// NewService creates a webhook service.
func NewService(secret string, notifier ports.Notifier) *Service {
	return &Service{secret: secret, notifier: notifier}
}

// HandlePush processes a GitHub push event.
func (s *Service) HandlePush(ctx context.Context, rawBody []byte, signatureHeader string) error {
	_, span := appOtel.Tracer(appOtel.TracerWebhookService).Start(ctx, "HandlePush")
	defer span.End()

	// Verify signature if secret is configured
	if s.secret != "" {
		if err := s.verifySignature(rawBody, signatureHeader); err != nil {
			return err
		}
	}

	var event domain.GitHubPushEvent
	if err := json.Unmarshal(rawBody, &event); err != nil {
		return fmt.Errorf("webhook: unmarshal push event: %w", err)
	}

	log.L().Info().
		Str("repo", event.Repository.FullName).
		Str("branch", event.BranchName()).
		Str("pusher", event.Pusher.Name).
		Int("commits", len(event.Commits)).
		Msg("received push event")

	// Send notification asynchronously — don't block GitHub's response.
	// GitHub has a 10s timeout; if Discord is slow, it would cause retries.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.L().Error().Interface("panic", r).Msg("panic in push notification goroutine")
			}
		}()
		notifyCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.notifier.NotifyPush(notifyCtx, &event); err != nil {
			log.L().Error().Err(err).Msg("failed to send push notification")
		}
	}()

	return nil
}

// verifySignature checks the X-Hub-Signature-256 header against the body.
func (s *Service) verifySignature(body []byte, signature string) error {
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return ErrInvalidSignature
	}

	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write(body)
	expected := prefix + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidSignature
	}
	return nil
}
