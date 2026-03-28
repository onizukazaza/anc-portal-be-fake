package sync

import (
	"context"
	"time"
)

// Mode กำหนดวิธี sync ข้อมูล.
type Mode string

const (
	ModeFull        Mode = "full"        // ลบ + insert ใหม่ทั้งหมด
	ModeIncremental Mode = "incremental" // sync เฉพาะ row ที่เปลี่ยนแปลง
)

// Syncer คือ interface สำหรับ sync ข้อมูลจาก source → destination.
// เพิ่มตารางใหม่แค่ implement interface นี้ + register ใน registry.
type Syncer interface {
	// Name ชื่อของ syncer (ใช้เป็น key ใน registry)
	Name() string

	// Sync ทำการ sync ข้อมูลตาม request ที่กำหนด.
	Sync(ctx context.Context, req SyncRequest) (*SyncResult, error)
}

// SyncRequest กำหนด parameter สำหรับ sync.
type SyncRequest struct {
	Mode      Mode      // full หรือ incremental
	BatchSize int       // จำนวน row ต่อ batch (default: 500)
	Since     time.Time // ใช้กับ incremental — sync row ที่ updated_at > Since
}

// SyncResult สรุปผลการ sync.
type SyncResult struct {
	Table     string        `json:"table"`
	Mode      Mode          `json:"mode"`
	Total     int           `json:"total"`
	Inserted  int           `json:"inserted"`
	Updated   int           `json:"updated"`
	Skipped   int           `json:"skipped"`
	Errors    int           `json:"errors"`
	Duration  time.Duration `json:"duration"`
	StartedAt time.Time     `json:"startedAt"`
}
