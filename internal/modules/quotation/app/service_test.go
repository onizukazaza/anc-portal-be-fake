package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/quotation/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/pagination"
)

// ─── GetByID ───

func TestGetByID(t *testing.T) {
	dbErr := errors.New("connection refused")

	tests := []struct {
		name    string
		repo    *fakeQuotationRepo
		id      string
		wantErr error
		wantID  string
	}{
		{
			name:   "success",
			repo:   &fakeQuotationRepo{quotation: &domain.Quotation{ID: "q1", DocNo: "DOC-001", Status: "active", CreatedAt: time.Now()}},
			id:     "q1",
			wantID: "q1",
		},
		{
			name:    "not found",
			repo:    &fakeQuotationRepo{quotation: nil},
			id:      "missing",
			wantErr: ErrNotFound,
		},
		{
			name:    "repo error",
			repo:    &fakeQuotationRepo{err: dbErr},
			id:      "q1",
			wantErr: dbErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(tc.repo)
			result, err := svc.GetByID(context.Background(), tc.id)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error: want %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ID != tc.wantID {
				t.Fatalf("id: want %s, got %s", tc.wantID, result.ID)
			}
		})
	}
}

// ─── ListByCustomer ───

func TestListByCustomer(t *testing.T) {
	dbErr := errors.New("timeout")

	tests := []struct {
		name      string
		repo      *fakeQuotationRepo
		pg        pagination.Request
		wantErr   error
		wantLen   int
		wantTotal int
		wantNext  bool
	}{
		{
			name: "success",
			repo: &fakeQuotationRepo{
				list:  []domain.Quotation{{ID: "q1", CustomerID: "c1"}, {ID: "q2", CustomerID: "c1"}},
				total: 5,
			},
			pg:        pagination.Request{Page: 1, Limit: 2, Order: "desc"},
			wantLen:   2,
			wantTotal: 5,
			wantNext:  true,
		},
		{
			name:      "empty",
			repo:      &fakeQuotationRepo{list: nil, total: 0},
			pg:        pagination.Request{Page: 1, Limit: 10, Order: "desc"},
			wantLen:   0,
			wantTotal: 0,
			wantNext:  false,
		},
		{
			name:    "repo error",
			repo:    &fakeQuotationRepo{err: dbErr},
			pg:      pagination.Request{Page: 1, Limit: 10, Order: "desc"},
			wantErr: dbErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(tc.repo)
			resp, err := svc.ListByCustomer(context.Background(), "c1", tc.pg)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error: want %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(resp.Items) != tc.wantLen {
				t.Fatalf("items: want %d, got %d", tc.wantLen, len(resp.Items))
			}
			if resp.Total != tc.wantTotal {
				t.Fatalf("total: want %d, got %d", tc.wantTotal, resp.Total)
			}
			if resp.HasNext != tc.wantNext {
				t.Fatalf("hasNext: want %v, got %v", tc.wantNext, resp.HasNext)
			}
		})
	}
}
