// pkg/localcache — In-memory cache (otter) สำหรับ hot data ภายใน process เดียว
//
// แยกจาก Redis (pkg/cache) อย่างชัดเจน:
//   - pkg/cache      = shared cache ข้าม instance (L2)
//   - pkg/localcache = in-process cache เครื่องเดียว (L1)
//
// ใช้ otter: lock-free, generics, S3-FIFO eviction
//
// วิธีใช้:
//
//	lc, _ := localcache.New(localcache.Config{MaxSize: 10_000, TTL: 5 * time.Minute})
//	lc.SetJSON("user:1", user)
//	lc.GetJSON("user:1", &user)
package localcache

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/maypok86/otter"
)

// Cache — interface สำหรับ in-memory cache operations
type Cache interface {
	// Raw bytes
	Get(key string) ([]byte, bool)
	Set(key string, value []byte)
	Delete(key string)
	Has(key string) bool
	Clear()
	Close()

	// JSON helpers
	GetJSON(key string, dest any) bool
	SetJSON(key string, value any)
}

// ------------------------------
// Config
// ------------------------------

const (
	defaultMaxSize = 10_000
	defaultTTL     = 5 * time.Minute
)

// Config กำหนดค่า local cache
type Config struct {
	MaxSize int           // จำนวน entry สูงสุด
	TTL     time.Duration // default TTL ต่อ entry
}

func (c Config) resolveMaxSize() int {
	if c.MaxSize <= 0 {
		return defaultMaxSize
	}
	return c.MaxSize
}

func (c Config) resolveTTL() time.Duration {
	if c.TTL <= 0 {
		return defaultTTL
	}
	return c.TTL
}

// ------------------------------
// Client
// ------------------------------

// Client wrap otter cache พร้อม default TTL
type Client struct {
	store      otter.Cache[string, []byte]
	defaultTTL time.Duration
}

// New สร้าง local cache instance
func New(cfg Config) (*Client, error) {
	ttl := cfg.resolveTTL()

	store, err := otter.MustBuilder[string, []byte](cfg.resolveMaxSize()).
		CollectStats().
		WithTTL(ttl).
		Build()
	if err != nil {
		return nil, fmt.Errorf("localcache: build failed: %w", err)
	}

	return &Client{store: store, defaultTTL: ttl}, nil
}

// ------------------------------
// Raw bytes operations
// ------------------------------

// Get — ดึง value จาก key
func (c *Client) Get(key string) ([]byte, bool) {
	return c.store.Get(key)
}

// Set — เขียน value ด้วย default TTL
func (c *Client) Set(key string, value []byte) {
	c.store.Set(key, value)
}

// Delete — ลบ key
func (c *Client) Delete(key string) {
	c.store.Delete(key)
}

// Has — ตรวจว่า key มีอยู่
func (c *Client) Has(key string) bool {
	return c.store.Has(key)
}

// Clear — ลบทุก entry
func (c *Client) Clear() {
	c.store.Clear()
}

// Close — ปิด cache และคืน resource
func (c *Client) Close() {
	c.store.Close()
}

// ------------------------------
// JSON helpers
// ------------------------------

// SetJSON — marshal เป็น JSON แล้วเก็บด้วย default TTL
func (c *Client) SetJSON(key string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	c.store.Set(key, data)
}

// GetJSON — unmarshal JSON จาก cache เข้า dest, return false ถ้าไม่พบ
func (c *Client) GetJSON(key string, dest any) bool {
	data, ok := c.store.Get(key)
	if !ok {
		return false
	}
	return json.Unmarshal(data, dest) == nil
}

// ------------------------------
// Stats
// ------------------------------

// Stats — สถิติ hit/miss ratio
func (c *Client) Stats() otter.Stats {
	return c.store.Stats()
}

// DefaultTTL — คืนค่า TTL ที่ใช้เป็น default
func (c *Client) DefaultTTL() time.Duration {
	return c.defaultTTL
}
