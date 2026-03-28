package app

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/domain"
)

// ─── Fake CMIPolicyRepository ───

type fakeCMIRepo struct {
	exists   bool
	existErr error
	policy   *domain.CMIPolicy
	findErr  error
}

func (f *fakeCMIRepo) JobExists(_ context.Context, _ string) (bool, error) {
	return f.exists, f.existErr
}

func (f *fakeCMIRepo) FindPolicyByJobID(_ context.Context, _ string) (*domain.CMIPolicy, error) {
	return f.policy, f.findErr
}
