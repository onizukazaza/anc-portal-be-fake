package ports

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/domain"
)

// CMIPolicyService handles CMI policy business logic.
type CMIPolicyService interface {
	// GetPolicyByJobID ดึงข้อมูล CMI policy โดยตรวจสอบ job ก่อน
	GetPolicyByJobID(ctx context.Context, jobID string) (*domain.CMIPolicy, error)
}
