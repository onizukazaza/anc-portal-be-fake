package http

import (
	"context"
	"errors"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/ports"
)

// fakeService wraps a fake repo so we can inject it into Handler
// without exporting Service fields.
// We build a real app.Service via NewService, but for handler tests
// we need to control the service behavior → use fakeServiceAdapter.

// fakeRepo implements ports.CMIPolicyRepository for handler-level tests.
type fakeRepo struct {
	exists   bool
	existErr error
	policy   *domain.CMIPolicy
	findErr  error
}

var _ ports.CMIPolicyRepository = (*fakeRepo)(nil)

func (f *fakeRepo) JobExists(_ context.Context, _ string) (bool, error) {
	return f.exists, f.existErr
}

func (f *fakeRepo) FindPolicyByJobID(_ context.Context, _ string) (*domain.CMIPolicy, error) {
	return f.policy, f.findErr
}

// helper errors for tests
var errDB = errors.New("db connection refused")
