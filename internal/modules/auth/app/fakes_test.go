package app

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/domain"
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
	token string
	err   error
}

func (f *fakeTokenSigner) SignAccessToken(_ context.Context, _ string, _ []string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.token, nil
}
