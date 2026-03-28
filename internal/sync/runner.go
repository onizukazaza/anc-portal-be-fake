package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

// DefaultBatchSize ขนาด batch เริ่มต้น.
const DefaultBatchSize = 500

// Runner ควบคุมการรัน syncer ทั้งหมด.
type Runner struct {
	registry *Registry
}

// NewRunner สร้าง runner จาก registry.
func NewRunner(registry *Registry) *Runner {
	return &Runner{registry: registry}
}

// RunOne รัน syncer ตัวเดียวตามชื่อ.
func (r *Runner) RunOne(ctx context.Context, name string, req SyncRequest) (*SyncResult, error) {
	s, err := r.registry.Get(name)
	if err != nil {
		return nil, err
	}

	req = normalizeRequest(req)

	log.L().Info().
		Str("syncer", name).
		Str("mode", string(req.Mode)).
		Int("batch_size", req.BatchSize).
		Msg("sync started")

	result, err := s.Sync(ctx, req)
	if err != nil {
		log.L().Error().Err(err).Str("syncer", name).Msg("sync failed")
		return nil, fmt.Errorf("sync %s: %w", name, err)
	}

	log.L().Info().
		Str("syncer", name).
		Int("total", result.Total).
		Int("inserted", result.Inserted).
		Int("updated", result.Updated).
		Int("skipped", result.Skipped).
		Int("errors", result.Errors).
		Dur("duration", result.Duration).
		Msg("sync completed")

	return result, nil
}

// RunAll รัน syncer ทั้งหมดที่ลงทะเบียนไว้.
func (r *Runner) RunAll(ctx context.Context, req SyncRequest) ([]*SyncResult, error) {
	syncers := r.registry.All()
	if len(syncers) == 0 {
		return nil, fmt.Errorf("no syncers registered")
	}

	req = normalizeRequest(req)

	results := make([]*SyncResult, 0, len(syncers))
	for _, s := range syncers {
		if ctx.Err() != nil {
			return results, ctx.Err()
		}

		result, err := r.RunOne(ctx, s.Name(), req)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	return results, nil
}

// AvailableSyncers คืนชื่อ syncer ทั้งหมด.
func (r *Runner) AvailableSyncers() []string {
	return r.registry.Names()
}

func normalizeRequest(req SyncRequest) SyncRequest {
	if req.Mode == "" {
		req.Mode = ModeFull
	}
	if req.BatchSize <= 0 {
		req.BatchSize = DefaultBatchSize
	}
	if req.Mode == ModeIncremental && req.Since.IsZero() {
		// incremental แต่ไม่ระบุ since → fallback เป็น 24 ชม. ย้อนหลัง
		req.Since = time.Now().Add(-24 * time.Hour)
	}
	return req
}
