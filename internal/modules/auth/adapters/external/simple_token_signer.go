package external

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type SimpleTokenSigner struct{}

func NewSimpleTokenSigner() *SimpleTokenSigner {
	return &SimpleTokenSigner{}
}

func (s *SimpleTokenSigner) SignAccessToken(_ context.Context, userID string, roles []string) (string, error) {
	return fmt.Sprintf("dev-token:%s:%s:%d", userID, strings.Join(roles, ","), time.Now().Unix()), nil
}
