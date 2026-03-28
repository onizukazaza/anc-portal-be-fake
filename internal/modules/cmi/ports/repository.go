package ports

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/domain"
)

// CMIPolicyRepository ดึงข้อมูล CMI policy จาก external database
type CMIPolicyRepository interface {
	// JobExists ตรวจสอบว่า job_id มีอยู่ในระบบหรือไม่
	JobExists(ctx context.Context, jobID string) (bool, error)

	// FindPolicyByJobID ดึงข้อมูล CMI policy ทั้งหมดตาม job_id
	FindPolicyByJobID(ctx context.Context, jobID string) (*domain.CMIPolicy, error)
}
