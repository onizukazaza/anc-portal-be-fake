package cmi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	cmipg "github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/adapters/postgres"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/app"
)

// ============================================================
// Integration test — ต่อ DB จริง + dump JSON ไว้ตรวจสอบ
//
// วิธีใช้:
//   CMI_TEST_DSN=postgres://user:pass@localhost:5432/meprakun_local_v2 \
//   CMI_TEST_JOB_ID=xxx \
//   go test ./internal/modules/cmi/... -run TestIntegration -v
//
// ⚠️  ถ้าไม่ตั้ง CMI_TEST_DSN จะ skip อัตโนมัติ (ไม่ fail ใน CI)
// ============================================================

const outputDir = "testdata/cmi"

func TestIntegrationGetPolicyByJobID(t *testing.T) {
	dsn := os.Getenv("CMI_TEST_DSN")
	if dsn == "" {
		t.Skip("skip: CMI_TEST_DSN not set")
	}

	jobID := os.Getenv("CMI_TEST_JOB_ID")
	if jobID == "" {
		t.Skip("skip: CMI_TEST_JOB_ID not set")
	}

	// -- connect
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect DB: %v", err)
	}
	defer pool.Close()

	// -- query
	repo := cmipg.NewCMIPolicyRepository(pool)
	svc := app.NewService(repo)

	policy, err := svc.GetPolicyByJobID(ctx, jobID)
	if err != nil {
		t.Fatalf("GetPolicyByJobID: %v", err)
	}

	// -- print JSON
	data, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}

	t.Logf("CMI Policy (job_id=%s):\n%s", jobID, data)

	// -- save to testdata/cmi/
	saveJSON(t, jobID, data)
}

func saveJSON(t *testing.T, jobID string, data []byte) {
	t.Helper()

	dir := outputDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}

	filename := fmt.Sprintf("policy_%s.json", jobID)
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}

	t.Logf("saved → %s", path)
}
