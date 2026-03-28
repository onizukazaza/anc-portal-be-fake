package localcache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/onizukazaza/anc-portal-be-fake/pkg/cache"
)

// Hybrid — L1 (otter in-memory) + L2 (Redis) cache
//
// อ่าน: ดู L1 ก่อน → miss แล้วค่อยไป L2 → เจอก็เขียนกลับ L1
// เขียน: เขียนทั้ง L1 + L2 พร้อมกัน
// ลบ: ลบทั้ง L1 + L2
type Hybrid struct {
	local Cache       // L1: in-memory (fast, per-instance)
	redis cache.Cache // L2: Redis (shared, cross-instance)
}

// NewHybrid สร้าง hybrid cache จาก local + redis
func NewHybrid(local Cache, redis cache.Cache) *Hybrid {
	return &Hybrid{local: local, redis: redis}
}

// ------------------------------
// Read — L1 → L2 (read-through)
// ------------------------------

// Get — ดู L1 ก่อน, miss แล้วดู L2, เจอก็ backfill L1
func (h *Hybrid) Get(ctx context.Context, key string) ([]byte, error) {
	// L1 hit
	if data, ok := h.local.Get(key); ok {
		return data, nil
	}

	// L2 lookup
	val, err := h.redis.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// backfill L1
	h.local.Set(key, []byte(val))
	return []byte(val), nil
}

// GetJSON — ดู L1 ก่อน, miss แล้วดู L2, เจอก็ backfill L1
func (h *Hybrid) GetJSON(ctx context.Context, key string, dest any) error {
	// L1 hit
	if h.local.GetJSON(key, dest) {
		return nil
	}

	// L2 lookup
	if err := h.redis.GetJSON(ctx, key, dest); err != nil {
		return err
	}

	// backfill L1
	h.local.SetJSON(key, dest)
	return nil
}

// Has — ตรวจจาก L1 ก่อน, ถ้าไม่มีค่อยถาม L2
func (h *Hybrid) Has(ctx context.Context, key string) (bool, error) {
	if h.local.Has(key) {
		return true, nil
	}
	return h.redis.Exists(ctx, key)
}

// ------------------------------
// Write — L1 + L2 (write-through)
// ------------------------------

// Set — เขียนทั้ง L1 + L2 พร้อมกัน
func (h *Hybrid) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	h.local.Set(key, value)
	return h.redis.Set(ctx, key, value, ttl)
}

// SetJSON — เขียน JSON ทั้ง L1 + L2
func (h *Hybrid) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	h.local.SetJSON(key, value)
	return h.redis.SetJSON(ctx, key, value, ttl)
}

// SetLocal — เขียนเฉพาะ L1 (ไม่ต้องแชร์ข้าม instance)
func (h *Hybrid) SetLocal(key string, value []byte) {
	h.local.Set(key, value)
}

// SetLocalJSON — เขียน JSON เฉพาะ L1
func (h *Hybrid) SetLocalJSON(key string, value any) {
	h.local.SetJSON(key, value)
}

// ------------------------------
// Delete — L1 + L2
// ------------------------------

// Delete — ลบทั้ง L1 + L2
func (h *Hybrid) Delete(ctx context.Context, keys ...string) error {
	for _, k := range keys {
		h.local.Delete(k)
	}
	return h.redis.Del(ctx, keys...)
}

// ------------------------------
// Invalidate L1 only
// ------------------------------

// InvalidateLocal — ลบ key จาก L1 อย่างเดียว (ใช้เมื่อรู้ว่า data เปลี่ยนใน L2)
func (h *Hybrid) InvalidateLocal(keys ...string) {
	for _, k := range keys {
		h.local.Delete(k)
	}
}

// ClearLocal — ลบทั้งหมดจาก L1
func (h *Hybrid) ClearLocal() {
	h.local.Clear()
}

// ------------------------------
// Fetch — read-through + loader
// ------------------------------

// Fetch — ดู L1 → L2 → ถ้าไม่มีทั้งคู่ เรียก loader แล้วเขียนกลับทั้ง L1+L2
func (h *Hybrid) Fetch(ctx context.Context, key string, dest any, ttl time.Duration, loader func(ctx context.Context) (any, error)) error {
	// L1 hit
	if h.local.GetJSON(key, dest) {
		return nil
	}

	// L2 hit → backfill L1
	if err := h.redis.GetJSON(ctx, key, dest); err == nil {
		h.local.SetJSON(key, dest)
		return nil
	}

	// ทั้งคู่ miss → load from source
	val, err := loader(ctx)
	if err != nil {
		return err
	}

	// เขียนกลับ L1 + L2
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	h.local.Set(key, data)
	if err := h.redis.Set(ctx, key, data, ttl); err != nil {
		return err
	}

	// unmarshal เข้า dest
	return json.Unmarshal(data, dest)
}
