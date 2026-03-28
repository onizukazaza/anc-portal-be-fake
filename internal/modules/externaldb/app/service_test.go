package app

import (
	"context"
	"errors"
	"testing"

	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/enum"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── CheckAll ───

func TestCheckAll(t *testing.T) {
	tests := []struct {
		name       string
		provider   *fakeDBProvider
		dbNames    []string
		wantLen    int
		wantStatus string
	}{
		{
			name:     "empty names",
			provider: &fakeDBProvider{},
			dbNames:  nil,
			wantLen:  0,
		},
		{
			name:       "pool not found returns error status",
			provider:   &fakeDBProvider{pools: map[string]*pgxpool.Pool{}},
			dbNames:    []string{"db1", "db2"},
			wantLen:    2,
			wantStatus: enum.DBError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(tc.provider, tc.dbNames)
			results := svc.CheckAll(context.Background())

			if len(results) != tc.wantLen {
				t.Fatalf("len: want %d, got %d", tc.wantLen, len(results))
			}
			for _, r := range results {
				if tc.wantStatus != "" && r.Status != tc.wantStatus {
					t.Fatalf("name=%s: status want %q, got %q", r.Name, tc.wantStatus, r.Status)
				}
			}
		})
	}
}

// ─── CheckByName ───

func TestCheckByName(t *testing.T) {
	tests := []struct {
		name       string
		provider   *fakeDBProvider
		dbNames    []string
		checkName  string
		wantStatus string
		wantError  string
	}{
		{
			name:       "external fails",
			provider:   &fakeDBProvider{err: errors.New("connection refused")},
			dbNames:    []string{"meprakun"},
			checkName:  "meprakun",
			wantStatus: enum.DBError,
			wantError:  "connection refused",
		},
		{
			name:       "unknown db",
			provider:   &fakeDBProvider{pools: map[string]*pgxpool.Pool{}},
			dbNames:    []string{},
			checkName:  "unknown_db",
			wantStatus: enum.DBError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(tc.provider, tc.dbNames)
			result := svc.CheckByName(context.Background(), tc.checkName)

			if result.Status != tc.wantStatus {
				t.Fatalf("status: want %q, got %q", tc.wantStatus, result.Status)
			}
			if tc.wantError != "" && result.Error != tc.wantError {
				t.Fatalf("error: want %q, got %q", tc.wantError, result.Error)
			}
		})
	}
}

// NOTE: healthy/unhealthy paths ต้องใช้ real DB หรือ pgxmock
// เพราะ check() เรียก pool.QueryRow() ซึ่งเป็น concrete type
// ถ้าต้องการ test ส่วนนั้น → เพิ่ม integration test แยก
