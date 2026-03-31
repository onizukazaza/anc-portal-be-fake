package external

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/ports"
)

// SimpleTokenSigner is a dev-only token signer that produces plaintext tokens.
// DO NOT use in production — tokens are not signed and can be forged.
type SimpleTokenSigner struct{}

func NewSimpleTokenSigner() *SimpleTokenSigner {
	return &SimpleTokenSigner{}
}

func (s *SimpleTokenSigner) SignAccessToken(_ context.Context, userID string, roles []string) (string, error) {
	return fmt.Sprintf("dev-token:%s:%s:%d", userID, strings.Join(roles, ","), time.Now().Unix()), nil
}

func (s *SimpleTokenSigner) VerifyAccessToken(_ context.Context, tokenString string) (*ports.Claims, error) {
	// format: dev-token:{userID}:{roles}:{timestamp}
	const prefix = "dev-token:"
	if !strings.HasPrefix(tokenString, prefix) {
		return nil, errors.New("invalid dev token format")
	}

	parts := strings.SplitN(tokenString[len(prefix):], ":", 3)
	if len(parts) < 2 {
		return nil, errors.New("invalid dev token format")
	}

	var roles []string
	if parts[1] != "" {
		roles = strings.Split(parts[1], ",")
	}

	return &ports.Claims{
		UserID: parts[0],
		Roles:  roles,
	}, nil
}
