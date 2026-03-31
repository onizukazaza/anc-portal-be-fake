package ports

import "context"

// Claims holds the decoded JWT payload passed through middleware.
type Claims struct {
	UserID string
	Roles  []string
}

// TokenSigner signs and verifies access tokens.
type TokenSigner interface {
	SignAccessToken(ctx context.Context, userID string, roles []string) (string, error)
	VerifyAccessToken(ctx context.Context, tokenString string) (*Claims, error)
}
