# Redis Cache Integration Guide

> **v2.0** — Last updated: March 2026
>
> ออกแบบด้วยแนวคิด Interface-First — module ใดก็ inject `cache.Cache` interface ได้ทันที
> ไม่ผูกกับ Redis โดยตรง

---

## สารบัญ

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Configuration](#configuration)
4. [Cache Interface](#cache-interface)
5. [การใช้งาน](#การใช้งาน)
6. [Key Prefix](#key-prefix)
7. [การ Inject เข้า Module](#การ-inject-เข้า-module)
8. [การใช้กรณี Cache เป็น nil](#การใช้กรณี-cache-เป็น-nil)
9. [Health Check](#health-check)
10. [Unit Testing](#unit-testing)
11. [TTL Guidelines](#ttl-guidelines)
12. [Redis CLI Quick Reference](#redis-cli-quick-reference)

---

## Overview

ระบบ Cache ถูกออกแบบด้วยแนวคิด **Interface-First** เพื่อให้:

- **ยืดหยุ่น** — module ไหนก็ inject ได้ผ่าน `cache.Cache` interface
- **ทดสอบง่าย** — mock ได้ทันที ไม่ต้องพึ่ง Redis จริง
- **ปลอดภัย** — key prefix อัตโนมัติ ป้องกัน key ชนกันข้าม service
- **optional** — เปิด/ปิดผ่าน `REDIS_ENABLED` ได้ ถ้าปิดระบบยังทำงานได้ปกติ

---

## Architecture

```
cmd/api/main.go          ← สร้าง cache.Client ตอน bootstrap
    │
    ▼
server.Server            ← เก็บ cache.Cache ไว้ใน struct
    │
    ├── /healthz          ← Ping Redis ตรวจสอบ connectivity
    │
    └── module routers    ← ส่ง cache.Cache เข้า service ของแต่ละ module
         │
         ▼
    app.Service           ← ใช้ cache.Cache ใน business logic
```

---

## Package Structure

```
pkg/cache/
├── cache.go      # Cache interface + Client implementation + JSON helpers
└── errors.go     # ErrCacheMiss sentinel error
```

---

## Configuration

### Environment Variables

| Variable | Type | Default | Description |
|---|---|---|---|
| `REDIS_ENABLED` | bool | `false` | เปิด/ปิด Redis cache |
| `REDIS_HOST` | string | _(required if enabled)_ | Redis server host |
| `REDIS_PORT` | int | `6379` | Redis server port |
| `REDIS_PASSWORD` | string | _(empty)_ | Redis password |
| `REDIS_DB` | int | `0` | Redis database number |
| `REDIS_KEY_PREFIX` | string | `anc:` | Prefix ที่จะนำหน้า key ทุกตัวอัตโนมัติ |

### ตัวอย่าง .env.local

```env
REDIS_ENABLED=true
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_KEY_PREFIX=anc:
```

---

## Cache Interface

```go
type Cache interface {
    // >> Basic Operations
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key string, value any, ttl time.Duration) error
    Del(ctx context.Context, keys ...string) error
    Exists(ctx context.Context, key string) (bool, error)

    // >> JSON Helpers — marshal/unmarshal ให้อัตโนมัติ
    GetJSON(ctx context.Context, key string, dest any) error
    SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error

    // >> Health & Lifecycle
    Ping(ctx context.Context) error
    Close() error
}
```

module ใดก็ตามควร **depend on `cache.Cache` interface** ไม่ใช่ `*cache.Client` โดยตรง  
สิ่งนี้ทำให้ mock ง่ายใน unit test

---

## การใช้งาน

### 1. Cache-Aside Pattern (แนะนำ)

Pattern ที่ใช้บ่อยที่สุด — **อ่านจาก cache ก่อน ถ้าไม่มีค่อย query จาก database แล้วเก็บ cache ไว้**

```go
func (s *Service) GetUser(ctx context.Context, id string) (*User, error) {
    cacheKey := "user:" + id

    // >> Try cache first
    var user User
    if err := s.cache.GetJSON(ctx, cacheKey, &user); err == nil {
        return &user, nil // cache hit
    }

    // >> Cache miss — query database
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // >> Store in cache for next time
    _ = s.cache.SetJSON(ctx, cacheKey, user, 5*time.Minute)
    return user, nil
}
```

### 2. Cache Invalidation

เมื่อข้อมูลเปลี่ยน ต้องลบ cache ที่เกี่ยวข้องเสมอ:

```go
func (s *Service) UpdateUser(ctx context.Context, id string, input UpdateInput) error {
    // >> Update database
    if err := s.repo.Update(ctx, id, input); err != nil {
        return err
    }

    // >> Invalidate cache
    _ = s.cache.Del(ctx, "user:"+id)
    return nil
}
```

### 3. ตรวจสอบ Cache Miss

ใช้ `errors.Is` เพื่อแยก "key ไม่มี" กับ "Redis error จริง":

```go
import "github.com/onizukazaza/anc-portal-be-fake/pkg/cache"

val, err := s.cache.Get(ctx, "some-key")
if errors.Is(err, cache.ErrCacheMiss) {
    // key ไม่มีใน cache — ไม่ใช่ error จริง
}
if err != nil && !errors.Is(err, cache.ErrCacheMiss) {
    // Redis error จริง เช่น connection refused
    return err
}
```

### 4. Set ค่าแบบไม่มี Expiry

```go
// ttl = 0 หมายถึง ไม่หมดอายุ
s.cache.Set(ctx, "config:feature-flags", flagJSON, 0)
```

### 5. ลบหลาย Key พร้อมกัน

```go
s.cache.Del(ctx, "user:1", "user:2", "user:3")
```

---

## Key Prefix

ระบบใช้ **key prefix อัตโนมัติ** ที่ตั้งค่าผ่าน `REDIS_KEY_PREFIX`

ตัวอย่าง: ถ้า `REDIS_KEY_PREFIX=anc:`

| Code Key | Redis Key จริง |
|---|---|
| `"user:1"` | `"anc:user:1"` |
| `"session:abc"` | `"anc:session:abc"` |
| `"config:flags"` | `"anc:config:flags"` |

ประโยชน์:
- ป้องกัน key ชนกันถ้าใช้ Redis server ร่วมกับ service อื่น
- ง่ายต่อการ debug ด้วย `redis-cli KEYS "anc:*"`

---

## การ Inject เข้า Module

### วิธีที่ 1: ผ่าน Service Constructor

```go
// internal/modules/quotation/app/service.go
type Service struct {
    repo  ports.QuotationRepository
    cache cache.Cache
}

func NewService(repo ports.QuotationRepository, cache cache.Cache) *Service {
    return &Service{repo: repo, cache: cache}
}
```

### วิธีที่ 2: ผ่าน Router Wiring

```go
// server/quotation_router.go
func (s *Server) initQuotationRouter(api fiber.Router) {
    repo := quotationpostgres.NewRepository(s.db.Main())
    service := quotationapp.NewService(repo, s.cache)  // inject cache
    controller := quotationhttp.NewController(service)

    router := api.Group("/quotation")
    router.Get("/:id", controller.GetByID)
}
```

---

## การใช้กรณี Cache เป็น nil

เมื่อ `REDIS_ENABLED=false` ค่า `cacheClient` จะเป็น `nil`  
module ต้องตรวจสอบก่อนใช้:

```go
func (s *Service) GetUser(ctx context.Context, id string) (*User, error) {
    // >> Try cache first (skip if cache not available)
    if s.cache != nil {
        var user User
        if err := s.cache.GetJSON(ctx, "user:"+id, &user); err == nil {
            return &user, nil
        }
    }

    // >> Query database
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // >> Store in cache (skip if cache not available)
    if s.cache != nil {
        _ = s.cache.SetJSON(ctx, "user:"+id, user, 5*time.Minute)
    }
    return user, nil
}
```

---

## Health Check

Redis ถูก integrate เข้ากับ `/healthz` และ `/ready` endpoints อัตโนมัติ:

```
GET /healthz
```

ถ้า Redis เชื่อมต่อไม่ได้:

```json
{
  "status": "degraded",
  "error": "redis: dial tcp 127.0.0.1:6379: connect: connection refused"
}
```

ใช้สำหรับ:
- **Kubernetes liveness/readiness probes**
- **Load balancer health checks**
- **Monitoring & alerting**

---

## Unit Testing

### Mock Cache สำหรับ Test

```go
type mockCache struct {
    store map[string]string
}

func newMockCache() *mockCache {
    return &mockCache{store: make(map[string]string)}
}

func (m *mockCache) Get(_ context.Context, key string) (string, error) {
    v, ok := m.store[key]
    if !ok {
        return "", cache.ErrCacheMiss
    }
    return v, nil
}

func (m *mockCache) Set(_ context.Context, key string, value any, _ time.Duration) error {
    m.store[key] = fmt.Sprintf("%v", value)
    return nil
}

func (m *mockCache) Del(_ context.Context, keys ...string) error {
    for _, k := range keys {
        delete(m.store, k)
    }
    return nil
}

func (m *mockCache) Exists(_ context.Context, key string) (bool, error) {
    _, ok := m.store[key]
    return ok, nil
}

func (m *mockCache) GetJSON(_ context.Context, key string, dest any) error {
    v, ok := m.store[key]
    if !ok {
        return cache.ErrCacheMiss
    }
    return json.Unmarshal([]byte(v), dest)
}

func (m *mockCache) SetJSON(_ context.Context, key string, value any, _ time.Duration) error {
    data, _ := json.Marshal(value)
    m.store[key] = string(data)
    return nil
}

func (m *mockCache) Ping(_ context.Context) error { return nil }
func (m *mockCache) Close() error                 { return nil }
```

### ตัวอย่าง Test

```go
func TestGetUser_CacheHit(t *testing.T) {
    mc := newMockCache()
    _ = mc.SetJSON(context.Background(), "user:1", &User{ID: "1", Name: "Alice"}, 0)

    svc := NewService(nil, mc) // repo = nil เพราะไม่ควรถูกเรียก
    user, err := svc.GetUser(context.Background(), "1")

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if user.Name != "Alice" {
        t.Fatalf("want Alice, got %s", user.Name)
    }
}
```

---

## TTL Guidelines

| ประเภทข้อมูล | TTL แนะนำ | เหตุผล |
|---|---|---|
| User profile | 5 min | เปลี่ยนไม่บ่อย แต่ต้อง fresh พอสมควร |
| Config / Feature flags | 1 min | ต้องการ update เร็ว |
| Session data | 30 min | ตาม session timeout |
| Quotation draft | 15 min | ข้อมูลชั่วคราวระหว่าง user กรอก |
| Static reference data | 1 hour | เปลี่ยนน้อยมาก |
| Rate limit counters | ตาม window | เช่น 1 min สำหรับ rate limit per minute |

---

## Redis CLI Quick Reference

คำสั่งที่ใช้บ่อยสำหรับ debug:

```bash
# เชื่อมต่อ Redis
redis-cli -h localhost -p 6379

# ดู keys ทั้งหมดของ anc service
KEYS "anc:*"

# ดู value ของ key
GET "anc:user:1"

# ดู TTL เหลือกี่วินาที
TTL "anc:user:1"

# ลบ key
DEL "anc:user:1"

# ดูข้อมูล server
INFO server

# ดูจำนวน key ทั้งหมด
DBSIZE

# ล้าง database ปัจจุบัน (ระวัง!)
FLUSHDB
```

---

> **v2.0** — March 2026 | ANC Portal Backend Team

## Flow Diagram

```
┌──────────┐     cache hit      ┌────────────┐
│  Client   │ ──── GET ────────▶│   Redis    │
│ (Fiber)   │                   │  (cache)   │
└─────┬─────┘                   └─────┬──────┘
      │                               │
      │  cache miss                    │ return cached
      ▼                               ▼
┌──────────┐                   ┌─────────────┐
│  Service  │ ── Query ──────▶│  PostgreSQL  │
│ (app)     │                  │  (database)  │
└─────┬─────┘                  └──────────────┘
      │
      │ SetJSON (store for next time)
      ▼
┌────────────┐
│   Redis    │
│  (cache)   │
└────────────┘
```
