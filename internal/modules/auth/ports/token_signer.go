package ports

import "context"

type TokenSigner interface {
	SignAccessToken(ctx context.Context, userID string, roles []string) (string, error)
}
