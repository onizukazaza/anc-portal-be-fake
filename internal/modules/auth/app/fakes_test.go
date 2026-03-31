package app

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/ports"
)

// ─── Fake UserRepository ───

type fakeUserRepo struct {
	user *domain.User
	err  error
}

func (f *fakeUserRepo) FindByUsername(_ context.Context, _ string) (*domain.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.user, nil
}

// ─── Fake TokenSigner ───

type fakeTokenSigner struct {
	token  string
	err    error
	claims *ports.Claims
}

func (f *fakeTokenSigner) SignAccessToken(_ context.Context, _ string, _ []string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.token, nil
}

func (f *fakeTokenSigner) VerifyAccessToken(_ context.Context, _ string) (*ports.Claims, error) {
	return f.claims, f.err
}
