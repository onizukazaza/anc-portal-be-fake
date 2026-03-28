package app

import (
	"context"
	"errors"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/ports"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

var (
	ErrJobIDRequired = errors.New("job_id is required")
	ErrJobNotFound   = errors.New("job not found")
)

type Service struct {
	repo ports.CMIPolicyRepository
}

func NewService(repo ports.CMIPolicyRepository) *Service {
	return &Service{repo: repo}
}

// GetPolicyByJobID ดึงข้อมูล CMI policy โดยตรวจสอบ job ก่อน
func (s *Service) GetPolicyByJobID(ctx context.Context, jobID string) (*domain.CMIPolicy, error) {
	ctx, span := appOtel.Tracer(appOtel.TracerCMIService).Start(ctx, "GetPolicyByJobID")
	defer span.End()

	log.L().Info().Str("layer", "service").Str("job_id", jobID).Msg("→ CMI GetPolicyByJobID")

	exists, err := s.repo.JobExists(ctx, jobID)
	if err != nil {
		log.L().Error().Err(err).Str("layer", "service").Str("job_id", jobID).Msg("← CMI JobExists error")
		return nil, err
	}
	if !exists {
		log.L().Warn().Str("layer", "service").Str("job_id", jobID).Msg("← CMI job not found")
		return nil, ErrJobNotFound
	}

	policy, err := s.repo.FindPolicyByJobID(ctx, jobID)
	if err != nil {
		log.L().Error().Err(err).Str("layer", "service").Str("job_id", jobID).Msg("← CMI FindPolicy error")
		return nil, err
	}

	log.L().Info().Str("layer", "service").Str("job_id", jobID).Msg("← CMI GetPolicyByJobID OK")
	return policy, nil
}
