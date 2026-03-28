// pkg/cache — Redis cache client สำหรับ application ทุก module
//
// ออกแบบให้ยืดหยุ่น:
//   - Cache interface เป็น abstraction กลาง ที่ module ไหนก็ inject ได้
//   - รองรับ Get/Set/Del/Exists + JSON helper (GetJSON/SetJSON)
//   - Health check ในตัว เพื่อใช้กับ /healthz endpoint
//   - key prefix อัตโนมัติ ป้องกัน key ชนกันข้าม service
//
// วิธีใช้:
//
//	client, err := cache.New(cache.Config{...})
//	client.SetJSON(ctx, "user:1", user, 5*time.Minute)
//	client.GetJSON(ctx, "user:1", &user)
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// ------------------------------
// >> Cache Interface
// ------------------------------

// Cache defines the contract for cache operations.
// Module ใดก็ตามควร depend on interface นี้ ไม่ใช่ concrete client
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)

	GetJSON(ctx context.Context, key string, dest any) error
	SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error

	Ping(ctx context.Context) error
	Close() error
}

// ------------------------------
// >> Config
// ------------------------------

type Config struct {
	Host        string
	Port        int
	Password    string
	DB          int
	KeyPrefix   string // prefix ทุก key อัตโนมัติ เช่น "anc:" → key "user:1" กลายเป็น "anc:user:1"
	OtelEnabled bool   // เปิด tracing ทุก Redis command
}

func (c Config) addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// ------------------------------
// >> Client (concrete implementation)
// ------------------------------

type Client struct {
	rdb       *redis.Client
	keyPrefix string
}

// New สร้าง Redis client พร้อม ping ทดสอบ connectivity ทันที
func New(ctx context.Context, cfg Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// >> Enable OpenTelemetry tracing for Redis commands
	if cfg.OtelEnabled {
		if err := redisotel.InstrumentTracing(rdb); err != nil {
			return nil, fmt.Errorf("redis otel tracing: %w", err)
		}
		if err := redisotel.InstrumentMetrics(rdb); err != nil {
			return nil, fmt.Errorf("redis otel metrics: %w", err)
		}
	}

	// >> Verify connection on startup
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &Client{rdb: rdb, keyPrefix: cfg.KeyPrefix}, nil
}

// prefixed คืน key พร้อม prefix
func (c *Client) prefixed(key string) string {
	if c.keyPrefix == "" {
		return key
	}
	return c.keyPrefix + key
}

// ------------------------------
// >> Basic Operations
// ------------------------------

// Get ดึง value จาก key — return ErrCacheMiss ถ้าไม่พบ
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.rdb.Get(ctx, c.prefixed(key)).Result()
	if err == redis.Nil {
		return "", ErrCacheMiss
	}
	return val, err
}

// Set เขียน value ลง key พร้อม TTL — ttl = 0 หมายถึง no expiry
func (c *Client) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return c.rdb.Set(ctx, c.prefixed(key), value, ttl).Err()
}

// Del ลบ key หนึ่งตัวหรือหลายตัว
func (c *Client) Del(ctx context.Context, keys ...string) error {
	prefixed := make([]string, len(keys))
	for i, k := range keys {
		prefixed[i] = c.prefixed(k)
	}
	return c.rdb.Del(ctx, prefixed...).Err()
}

// Exists ตรวจว่า key มีอยู่หรือไม่
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.rdb.Exists(ctx, c.prefixed(key)).Result()
	return n > 0, err
}

// ------------------------------
// >> JSON Helpers
// ------------------------------

// SetJSON marshal struct เป็น JSON แล้วเก็บลง Redis
func (c *Client) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache json marshal: %w", err)
	}
	return c.Set(ctx, key, data, ttl)
}

// GetJSON ดึง JSON จาก Redis แล้ว unmarshal เข้า dest — return ErrCacheMiss ถ้าไม่พบ
func (c *Client) GetJSON(ctx context.Context, key string, dest any) error {
	val, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

// ------------------------------
// >> Health & Lifecycle
// ------------------------------

// Ping ตรวจสอบ connectivity กับ Redis server
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Close ปิด connection กับ Redis
func (c *Client) Close() error {
	return c.rdb.Close()
}
