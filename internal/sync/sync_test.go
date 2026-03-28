package sync

import (
	"context"
	"testing"
	"time"
)

// ─── Fake Syncer ───

type fakeSyncer struct {
	name   string
	result *SyncResult
	err    error
	calls  int
}

func (f *fakeSyncer) Name() string { return f.name }
func (f *fakeSyncer) Sync(_ context.Context, req SyncRequest) (*SyncResult, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	r := *f.result
	r.Mode = req.Mode
	return &r, nil
}

func newFakeSyncer(name string, total int) *fakeSyncer {
	return &fakeSyncer{
		name: name,
		result: &SyncResult{
			Table:     name,
			Total:     total,
			Inserted:  total,
			Duration:  100 * time.Millisecond,
			StartedAt: time.Now(),
		},
	}
}

// ─── Registry ───

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	s := newFakeSyncer("orders", 10)
	reg.Register(s)

	got, err := reg.Get("orders")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name() != "orders" {
		t.Fatalf("want name 'orders', got %q", got.Name())
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Get("unknown")
	if err == nil {
		t.Fatal("want error for unknown syncer, got nil")
	}
}

func TestRegistryNames(t *testing.T) {
	reg := NewRegistry()
	reg.Register(newFakeSyncer("a", 1))
	reg.Register(newFakeSyncer("b", 2))

	names := reg.Names()
	if len(names) != 2 {
		t.Fatalf("want 2 names, got %d", len(names))
	}
}

func TestRegistryAll(t *testing.T) {
	reg := NewRegistry()
	reg.Register(newFakeSyncer("x", 5))
	reg.Register(newFakeSyncer("y", 10))

	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("want 2 syncers, got %d", len(all))
	}
}

// ─── Runner ───

func TestRunnerRunOneSuccess(t *testing.T) {
	reg := NewRegistry()
	s := newFakeSyncer("quotations", 50)
	reg.Register(s)

	runner := NewRunner(reg)
	result, err := runner.RunOne(context.Background(), "quotations", SyncRequest{Mode: ModeFull})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 50 {
		t.Fatalf("total: want 50, got %d", result.Total)
	}
	if result.Mode != ModeFull {
		t.Fatalf("mode: want %q, got %q", ModeFull, result.Mode)
	}
	if s.calls != 1 {
		t.Fatalf("calls: want 1, got %d", s.calls)
	}
}

func TestRunnerRunOneNotFound(t *testing.T) {
	reg := NewRegistry()
	runner := NewRunner(reg)

	_, err := runner.RunOne(context.Background(), "missing", SyncRequest{})
	if err == nil {
		t.Fatal("want error for missing syncer, got nil")
	}
}

func TestRunnerRunOneSyncError(t *testing.T) {
	reg := NewRegistry()
	s := &fakeSyncer{name: "broken", err: context.DeadlineExceeded}
	reg.Register(s)

	runner := NewRunner(reg)
	_, err := runner.RunOne(context.Background(), "broken", SyncRequest{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestRunnerRunAllSuccess(t *testing.T) {
	reg := NewRegistry()
	reg.Register(newFakeSyncer("a", 10))
	reg.Register(newFakeSyncer("b", 20))

	runner := NewRunner(reg)
	results, err := runner.RunAll(context.Background(), SyncRequest{Mode: ModeIncremental})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results: want 2, got %d", len(results))
	}
}

func TestRunnerRunAllEmpty(t *testing.T) {
	reg := NewRegistry()
	runner := NewRunner(reg)

	_, err := runner.RunAll(context.Background(), SyncRequest{})
	if err == nil {
		t.Fatal("want error for empty registry, got nil")
	}
}

func TestRunnerRunAllContextCancelled(t *testing.T) {
	reg := NewRegistry()
	reg.Register(newFakeSyncer("a", 10))
	reg.Register(newFakeSyncer("b", 20))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel ทันที

	runner := NewRunner(reg)
	_, err := runner.RunAll(ctx, SyncRequest{})
	if err == nil {
		t.Fatal("want error for cancelled context, got nil")
	}
}

func TestRunnerAvailableSyncers(t *testing.T) {
	reg := NewRegistry()
	reg.Register(newFakeSyncer("customers", 0))
	reg.Register(newFakeSyncer("orders", 0))

	runner := NewRunner(reg)
	names := runner.AvailableSyncers()
	if len(names) != 2 {
		t.Fatalf("want 2, got %d", len(names))
	}
}

// ─── normalizeRequest ───

func TestNormalizeRequestDefaults(t *testing.T) {
	req := normalizeRequest(SyncRequest{})

	if req.Mode != ModeFull {
		t.Fatalf("mode: want %q, got %q", ModeFull, req.Mode)
	}
	if req.BatchSize != DefaultBatchSize {
		t.Fatalf("batchSize: want %d, got %d", DefaultBatchSize, req.BatchSize)
	}
}

func TestNormalizeRequestIncrementalSince(t *testing.T) {
	req := normalizeRequest(SyncRequest{Mode: ModeIncremental})

	if req.Since.IsZero() {
		t.Fatal("since should be set for incremental mode")
	}
	// ต้องเป็นเวลาภายใน 25 ชม. ที่ผ่านมา (24h + tolerance)
	if time.Since(req.Since) > 25*time.Hour {
		t.Fatalf("since too old: %v", req.Since)
	}
}

func TestNormalizeRequestPreservesValues(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	req := normalizeRequest(SyncRequest{
		Mode:      ModeIncremental,
		BatchSize: 100,
		Since:     since,
	})

	if req.BatchSize != 100 {
		t.Fatalf("batchSize: want 100, got %d", req.BatchSize)
	}
	if !req.Since.Equal(since) {
		t.Fatalf("since: want %v, got %v", since, req.Since)
	}
}
