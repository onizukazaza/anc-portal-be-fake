package app

import (
	"context"
	"errors"
	"testing"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/domain"
)

func TestGetPolicyByJobID(t *testing.T) {
	dbErr := errors.New("connection refused")
	samplePolicy := &domain.CMIPolicy{
		JobID:     "job-001",
		JobType:   "cmi_only",
		JobStatus: "quotations",
		AgentID:   "agent-001",
		Motor:     &domain.MotorInfo{Year: "2025", Brand: "Toyota", Model: "Camry"},
	}

	tests := []struct {
		name    string
		repo    *fakeCMIRepo
		jobID   string
		wantErr error
		wantID  string
	}{
		{
			name:   "success",
			repo:   &fakeCMIRepo{exists: true, policy: samplePolicy},
			jobID:  "job-001",
			wantID: "job-001",
		},
		{
			name:    "job not found",
			repo:    &fakeCMIRepo{exists: false},
			jobID:   "missing",
			wantErr: ErrJobNotFound,
		},
		{
			name:    "repo error on JobExists",
			repo:    &fakeCMIRepo{existErr: dbErr},
			jobID:   "job-001",
			wantErr: dbErr,
		},
		{
			name:    "repo error on FindPolicy",
			repo:    &fakeCMIRepo{exists: true, findErr: dbErr},
			jobID:   "job-001",
			wantErr: dbErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(tc.repo)
			policy, err := svc.GetPolicyByJobID(context.Background(), tc.jobID)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error: want %v, got %v", tc.wantErr, err)
				}
				if policy != nil {
					t.Fatalf("policy: want nil, got %+v", policy)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if policy.JobID != tc.wantID {
				t.Fatalf("jobID: want %s, got %s", tc.wantID, policy.JobID)
			}
		})
	}
}
