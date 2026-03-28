package app

import (
	"context"
	"errors"
	"strings"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/ports"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type Service struct {
	users  ports.UserRepository
	tokens ports.TokenSigner
}

func NewService(users ports.UserRepository, tokens ports.TokenSigner) *Service {
	return &Service{users: users, tokens: tokens}
}

func (s *Service) Login(ctx context.Context, username string, password string) (*domain.Session, error) {
	ctx, span := appOtel.Tracer(appOtel.TracerAuthService).Start(ctx, "Login")
	defer span.End()

	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	if user == nil || !verifyPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.tokens.SignAccessToken(ctx, user.ID, user.Roles)
	if err != nil {
		return nil, err
	}

	return &domain.Session{
		AccessToken: accessToken,
		UserID:      user.ID,
		Roles:       user.Roles,
	}, nil
}

func verifyPassword(storedPassword string, rawPassword string) bool {
	if storedPassword == "" {
		return false
	}

	if isBcryptHash(storedPassword) {
		return bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(rawPassword)) == nil
	}

	return storedPassword == rawPassword
}

func isBcryptHash(value string) bool {
	return strings.HasPrefix(value, "$2a$") ||
		strings.HasPrefix(value, "$2b$") ||
		strings.HasPrefix(value, "$2y$")
}
