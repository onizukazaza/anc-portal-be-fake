# Microservice Readiness — ANC Portal Backend

> **v3.0** — Last updated: March 2026
>
> วิเคราะห์ว่าโครงสร้างปัจจุบัน **พร้อมแค่ไหน** สำหรับ Microservices
> ครอบคลุม: Microservice คืออะไร, เหมาะอย่างไร, ช่องว่าง, แผนการ extract
>
> **v3.0 changes:** Multi-driver DB (postgres+mysql), Route-level middleware (JWT+APIKey), Module-owned auth

---

## สารบัญ

1. [Microservice คืออะไร](#1-microservice-คืออะไร)
2. [เราอยู่ตรงไหน — Modular Monolith](#2-เราอยู่ตรงไหน--modular-monolith)
3. [Readiness Score — พร้อมแค่ไหน](#3-readiness-score--พร้อมแค่ไหน)
4. [จุดแข็งที่มีแล้ว (ทำถูก)](#4-จุดแข็งที่มีแล้ว-ทำถูก)
5. [ช่องว่างที่ยังขาด](#5-ช่องว่างที่ยังขาด)
6. [Monolith vs Microservice — เปรียบเทียบตรงๆ](#6-monolith-vs-microservice--เปรียบเทียบตรงๆ)
7. [แผนการ Extract — 4 Phases](#7-แผนการ-extract--4-phases)
8. [Module ไหนควร Extract ก่อน](#8-module-ไหนควร-extract-ก่อน)
9. [สิ่งที่ต้องทำก่อน Extract](#9-สิ่งที่ต้องทำก่อน-extract)
10. [Anti-Patterns ที่ต้องหลีกเลี่ยง](#10-anti-patterns-ที่ต้องหลีกเลี่ยง)
11. [สรุป](#11-สรุป)

---

## 1. Microservice คืออะไร

### คำอธิบายง่ายๆ

**Microservice** = สถาปัตยกรรมที่แบ่งระบบออกเป็น **service ย่อยๆ** แต่ละตัวทำงานอิสระ
deploy อิสระ scale อิสระ และสื่อสารกันผ่าน **network** (HTTP/gRPC/Event)

```
┌─────────────────── Monolith ───────────────────┐
│                                                │
│  ┌──────┐ ┌──────────┐ ┌─────┐ ┌───────────┐  │
│  │ Auth │ │Quotation │ │ CMI │ │ExternalDB │  │
│  └──────┘ └──────────┘ └─────┘ └───────────┘  │
│              (1 binary, 1 process)             │
└────────────────────────────────────────────────┘

                      vs

┌─────────────── Microservices ──────────────────┐
│                                                │
│  ┌──────┐   ┌──────────┐   ┌─────┐   ┌──────┐ │
│  │ Auth │   │Quotation │   │ CMI │   │ExtDB │ │
│  │ :3001│   │  :3002   │   │:3003│   │:3004 │ │
│  └──┬───┘   └────┬─────┘   └──┬──┘   └──┬───┘ │
│     │            │             │          │     │
│     └────────────┴─────────────┴──────────┘     │
│            HTTP / gRPC / Kafka Events           │
└────────────────────────────────────────────────┘
```

### ลักษณะหลัก 5 ข้อ

| ลักษณะ | คำอธิบาย |
|--------|---------|
| **1. Single Responsibility** | 1 service ทำ 1 งาน (Auth ดูแลเฉพาะ authentication) |
| **2. Independent Deploy** | deploy Auth ใหม่โดยไม่ต้อง deploy Quotation |
| **3. Own Database** | แต่ละ service มี DB ของตนเอง ไม่ share schema |
| **4. Communication via Network** | คุยกันผ่าน HTTP, gRPC, หรือ Kafka event |
| **5. Independent Scale** | Auth มี traffic สูง → scale Auth เพิ่ม โดย CMI ไม่ต้องเพิ่ม |

### เมื่อไหร่ควรใช้ Microservice?

```
ใช้เมื่อ:                              ยังไม่ต้องใช้เมื่อ:
─────────                              ──────────────────
✓ ทีม 5+ คน ทำงานคนละ module           ✗ ทีม 1-3 คน
✓ Module มี traffic ต่างกันมาก          ✗ Traffic สม่ำเสมอ
✓ ต้อง deploy บ่อย แยก module           ✗ Deploy ทีเดียวทั้งระบบ
✓ ต้อง scale เฉพาะบาง module            ✗ Scale ทั้งระบบก็พอ
✓ ต้องใช้ tech stack ต่างกัน            ✗ Go ตัวเดียวพอ
```

---

## 2. เราอยู่ตรงไหน — Modular Monolith

### ปัจจุบัน: **Modular Monolith + Hexagonal Architecture**

```
                    ┌─── เราอยู่ตรงนี้ ───┐
                    │                     │
                    ▼                     │
 ┌──────────┐  ┌───────────┐  ┌──────────────────┐  ┌──────────────┐
 │ Monolith │→ │ Modular   │→ │ Microservice     │→ │ Event-Driven │
 │ (สปาเกตตี้)│  │ Monolith  │  │ (Service ย่อยๆ)  │  │ (Event       │
 │          │  │ (Module   │  │                  │  │  Sourcing)   │
 │          │  │  แยกชัด)   │  │                  │  │              │
 └──────────┘  └───────────┘  └──────────────────┘  └──────────────┘
```

### ทำไมถึงอยู่ตำแหน่งที่ดี?

**Modular Monolith** คือจุดเริ่มต้นที่ดีที่สุดสำหรับ Microservices:

- Module แยกชัดเจน → extract เป็น service ได้
- Business logic ไม่ผูก framework → ย้ายไปไหนก็ได้
- Interface (Ports) กำหนดขอบเขตชัด → ตัด dependency ได้ง่าย
- Deploy แบบ monolith ยังได้ → ค่อย extract เมื่อพร้อม

### โครงสร้าง Module ปัจจุบัน

```
internal/modules/
├── auth/            ← ✅ Hexagonal (domain + ports + app + adapters)
│   ├── domain/      ← Pure Go business entities
│   ├── ports/       ← Interface contracts
│   ├── app/         ← Use cases / service layer
│   ├── adapters/    ← HTTP handlers + PostgreSQL repos
│   └── module.go    ← Register(router, deps)
├── quotation/       ← ✅ Hexagonal
├── cmi/             ← ✅ Hexagonal
├── externaldb/      ← ✅ Hexagonal
├── payment/         ← ⬜ Placeholder
├── job/             ← ⬜ Placeholder
├── notification/    ← ⬜ Placeholder
└── document/        ← ⬜ Placeholder
```

### Dependency Injection — จุดเชื่อมต่อเดียว

```go
// internal/shared/module/deps.go
type Middleware struct {
    JWTAuth    fiber.Handler // Bearer-token (JWT) verification
    APIKeyAuth fiber.Handler // X-API-Key header verification
}

type Deps struct {
    Config      *config.Config
    DB          database.Provider
    Cache       cache.Cache
    LocalCache  localcache.Cache
    HybridCache *localcache.Hybrid
    Middleware  Middleware          // ← NEW: route-level auth
}
```

ทุก module รับ `Deps` struct เดียว — รวมถึง middleware handlers
→ **module ตัดสินใจเองว่า route ไหนใช้ auth แบบไหน**

```go
// ตัวอย่างใน module — เลือก auth ต่อ endpoint
group.Get("/public", ctrl.Public)                              // ไม่ต้อง auth
group.Get("/:id", deps.Middleware.JWTAuth, ctrl.GetByID)       // JWT
group.Post("/hook", deps.Middleware.APIKeyAuth, ctrl.Hook)     // API Key
```

---

## 3. Readiness Score — พร้อมแค่ไหน

### คะแนนรวม: **8.1 / 10 — พร้อมสำหรับ Phase 1 Extraction**

| ด้าน | คะแนน | สถานะ |
|------|-------|-------|
| Module Boundary Separation | 9/10 | ✅ Hexagonal pattern ทุก module |
| Dependency Injection | 9/10 | ✅ `module.Deps` + `Middleware` struct |
| Route-Level Auth | 9/10 | ✅ JWT + API Key per endpoint |
| Event-Driven (Kafka) | 9/10 | ✅ Producer/Consumer/Router/DLQ/Tracing |
| Database Multi-Driver | 9/10 | ✅ Provider + ExternalConn (postgres/mysql) |
| Distributed Tracing | 9/10 | ✅ OTel + W3C propagation |
| Cache Strategy | 8/10 | ✅ Hybrid L1+L2 + feature toggles |
| Configuration (12-Factor) | 9/10 | ✅ Stage-aware, env-based |
| Graceful Degradation | 9/10 | ✅ Redis/Kafka/OTel optional |
| Testing Infrastructure | 7/10 | ⚠️ testkit ดี แต่ coverage ยังไม่ครบ |
| CI/CD Pipeline | 5/10 | ⚠️ ยังไม่มี pipeline อัตโนมัติ |
| gRPC / Service-to-Service | 4/10 | ❌ ยังไม่มี gRPC |
| Circuit Breaker | 8/10 | ✅ sony/gobreaker ใน pkg/httpclient |
| Service Discovery | 4/10 | ❌ ต้องใช้ K8s DNS หรือ Consul |

### แผนภาพ Readiness

```
Module Boundary   ████████████████████░  9/10
Dependency Inj.   ████████████████████░  9/10
Route-Level Auth  ████████████████████░  9/10  ← NEW
Event-Driven      ████████████████████░  9/10
Database Multi-DB ████████████████████░  9/10  ← UP (8→9)
Tracing           ████████████████████░  9/10
Cache             ████████████████░░░░░  8/10
Config            ████████████████████░  9/10
Degradation       ████████████████████░  9/10
Testing           ██████████████░░░░░░░  7/10
CI/CD             ██████████░░░░░░░░░░░  5/10
gRPC              ████████░░░░░░░░░░░░░  4/10
Circuit Breaker   ████████████████░░░░░  8/10
Service Discovery ████████░░░░░░░░░░░░░  4/10
                  ─────────────────────
                  Average: 8.1/10
```

---

## 4. จุดแข็งที่มีแล้ว (ทำถูก)

### 4.1 Hexagonal Architecture — ขอบเขต Module ชัด

```
      ┌─── Module Boundary ───┐
      │                       │
      │   ┌──── domain ────┐  │    ← Pure Go (business rules)
      │   └───────┬────────┘  │
      │           │           │
      │   ┌──── ports ─────┐  │    ← Interface contracts
      │   └──┬─────────┬───┘  │
      │      │         │      │
      │  ┌───▼──┐  ┌───▼──┐  │
      │  │ app  │  │adapt.│  │    ← Use cases + implementations
      │  └──────┘  └──────┘  │
      │                       │
      └───────────────────────┘
            ↕ (ตัดตรงนี้)
        Become its own service
```

ทุก module มีขอบเขตชัด ไม่ import ข้าม module โดยตรง
→ **ตัดออกเป็น service ได้โดยไม่ต้อง refactor**

### 4.2 Event-Driven Architecture — Kafka พร้อม

```go
// producer — ส่ง event
producer.PublishMessage(ctx, kafka.Message{
    Type:    "quotation.created",
    Key:     quotationID,
    Payload: jsonData,
})

// consumer — รับ event (Event Router pattern)
router.Register("quotation.created", handleQuotationCreated)
router.Register("notification.send", handleNotification)
```

สิ่งที่มีแล้ว:
- **Producer/Consumer** แยก binary (`cmd/api` vs `cmd/worker`)
- **Event Router** — dispatch ตาม event type
- **Dead Letter Queue** — failed messages ไม่หาย
- **Retry + Exponential Backoff** — via `pkg/retry`
- **W3C Trace Propagation** — tracing ข้าม service ผ่าน Kafka headers

### 4.3 Database Provider — Multi-Driver (postgres + mysql)

```go
// Provider — module ขึ้นกับ interface นี้เท่านั้น
type Provider interface {
    Main() *pgxpool.Pool                          // Internal DB (always postgres)
    External(name string) (ExternalConn, error)   // External/Legacy DBs (any driver)
    Read() *pgxpool.Pool                          // Read replica (future)
    Write() *pgxpool.Pool                         // Write master (future)
    HealthCheck(ctx context.Context) error
    Close()
}

// ExternalConn — driver-agnostic interface
type ExternalConn interface {
    Health(ctx context.Context) error
    Close()
    Driver() string                                // "postgres" | "mysql"
    Diagnostic(ctx context.Context) (dbName, version string, err error)
}
```

- `External()` คืน `ExternalConn` — ไม่ผูกกับ driver ใด driver หนึ่ง
- Module ใช้ type-safe helpers: `database.PgxPool(conn)` หรือ `database.SQLDB(conn)`
- รองรับ **postgres + mysql** — เพิ่ม driver ใหม่ได้ผ่าน switch-case ใน Manager
- Read/Write split เตรียมไว้แล้ว

```go
// ตัวอย่างใน module
conn, _ := deps.DB.External("meprakun")
pool, _ := database.PgxPool(conn)    // postgres → *pgxpool.Pool
db, _   := database.SQLDB(conn)      // mysql   → *sql.DB
```

### 4.4 Distributed Tracing — สมบูรณ์

```
API Service (Fiber)          Worker Service (Kafka Consumer)
    │                              │
    │  traceparent: 00-abc...      │
    │ ─────────────────────────▶   │
    │         Kafka Headers        │
    │                              │
    └── Span: api.handler ──┐     └── Span: worker.handler ──┐
                            │                                 │
                     ┌──────▼──────┐                  ┌───────▼──────┐
                     │   Tempo     │◄─────────────────│   Tempo      │
                     │ (Trace DB)  │  same trace ID   │ (Trace DB)   │
                     └─────────────┘                  └──────────────┘
```

- **OTel SDK** ครบทุก layer (HTTP → Service → Repository → Kafka)
- **Central Tracer Registry** — ชื่อ tracer เป็น constant, ไม่ hardcode string
- **W3C propagation** — ต่อ trace ข้าม service ผ่าน Kafka headers

### 4.5 Infrastructure Optional — Graceful Degradation

```yaml
# ทุก infra ปิดได้โดยไม่พัง:
REDIS_ENABLED=false          # cache off → direct DB
KAFKA_ENABLED=false          # event off → synchronous
OTEL_ENABLED=false           # tracing off → no overhead
LOCAL_CACHE_ENABLED=false    # L1 cache off → L2 only
SWAGGER_ENABLED=false        # docs off → production
```

สิ่งนี้สำคัญมากสำหรับ Microservices เพราะ:
- Dev สามารถ run service เดียวโดยไม่ต้อง setup ทั้ง stack
- ถ้า Redis ล่ม → service ยัง run ได้ (แค่ช้าลง)
- ถ้า Kafka ล่ม → API ยัง serve ได้ (event ส่งไม่ได้แต่ไม่ crash)

### 4.6 Route-Level Auth — ยืดหยุ่นระดับ Endpoint

Middleware ไม่ได้ mount global แล้ว — แต่ละ module เลือกเองว่า route ไหนใช้ auth แบบไหน:

```go
// server.go — สร้าง middleware แล้วส่งผ่าน Deps
deps := module.Deps{
    Middleware: module.Middleware{
        JWTAuth:    mw.Auth(mw.AuthConfig{TokenSigner: tokenSigner}),
        APIKeyAuth: mw.APIKey(mw.APIKeyConfig{ValidKeys: cfg.Server.APIKeys.Internal}),
    },
}

// module เลือกใช้เอง
group.Post("/login", ctrl.Login)                                // public
group.Get("/:id", deps.Middleware.JWTAuth, ctrl.GetByID)        // JWT
group.Post("/webhook", deps.Middleware.APIKeyAuth, ctrl.Hook)   // API Key
```

ข้อดีสำหรับ Microservices:
- **Extract module แล้ว auth logic ไปด้วย** — ไม่ต้องแก้ route config ที่ server
- **Service-to-service** ใช้ API Key, user-facing ใช้ JWT — config อยู่ใน module
- **Constant-time comparison** สำหรับ API Key — ป้องกัน timing attack

### 4.7 แยก Binary แล้ว — 6 Entry Points

```
cmd/
├── api/     → HTTP Server (port 20000)     ← extract เป็น API Gateway ได้
├── worker/  → Kafka Consumer               ← แยก service ได้ทันที
├── migrate/ → Schema Migration             ← ใช้ร่วมกับ CI/CD
├── seed/    → Data Seeding                 ← dev tooling
├── import/  → CSV Import                   ← batch job service ได้
└── sync/    → External DB Sync             ← data pipeline service ได้
```

---

## 5. ช่องว่างที่ยังขาด

### 5.1 ยังไม่มี gRPC

```
ตอนนี้:  Module A ──(function call)──▶ Module B    (in-process)
ต้องเป็น: Service A ──(gRPC/HTTP)────▶ Service B   (over network)
```

**ต้องทำ:**
- สร้าง `.proto` definitions สำหรับ inter-service API
- หรือใช้ REST + OpenAPI schema ระหว่าง service

### 5.2 Circuit Breaker ✅ ทำแล้ว

```
ตอนนี้:  Service A ──▶ Circuit Breaker (sony/gobreaker) ──▶ fail fast (50ms) ──▶ fallback
```

**สิ่งที่ทำแล้ว:**
- `pkg/httpclient` มี `WithCircuitBreaker(name)` option
- sony/gobreaker/v2 v2.4.0 — defaults: MaxRequests=5, Interval=30s, Timeout=10s, ConsecutiveFailures≥5
- `IsCircuitOpen(err)` helper ตรวจสถานะ circuit
- Execution chain: traced → circuit breaker → retry

### 5.3 ยังไม่มี Service Discovery

```
ตอนนี้:   hardcode URL ใน config
ควรเป็น:  K8s DNS (auth-service.default.svc.cluster.local)
          หรือ Consul / etcd
```

### 5.4 ยังไม่มี Distributed Transaction (SAGA)

```
ตอนนี้:   1 DB transaction ครอบทุก module
ควรเป็น:  SAGA pattern (event-driven compensating transactions)

ตัวอย่าง:
  1. Quotation Service → create quote     ✅
  2. Kafka event → Payment Service         ✅
  3. Payment fails → compensate Quotation  ← ต้องสร้าง pattern นี้
```

### 5.5 Database ยังไม่แยก Schema ต่อ Service

```
ตอนนี้:   ทุก module ใช้ 1 DB (anc_portal)
ควรเป็น:  auth_db, quotation_db, cmi_db (แยก schema/DB ต่อ service)
```

### 5.6 Event Schema Versioning

```
ตอนนี้:   Kafka message มี Type + Payload (JSON)
ควรเป็น:  Type + Version + Schema Registry
          ป้องกัน breaking changes ใน event format
```

---

## 6. Monolith vs Microservice — เปรียบเทียบตรงๆ

### ด้าน Development

| หัวข้อ | Modular Monolith (ปัจจุบัน) | Microservices |
|--------|---------------------------|---------------|
| **Build** | `go build ./cmd/api` (1 binary) | N binaries + Docker images |
| **Run locally** | `.\run.ps1 dev` จบ | docker-compose 10+ services |
| **Debug** | Go debugger ปกติ | Debug ข้าม service ยาก |
| **Refactor** | compiler ช่วยหา error | ต้อง contract testing |
| **New feature** | เพิ่ม module + register | เพิ่ม service + deploy + routing |
| **Module ใช้ data ร่วมกัน** | function call (0 latency) | HTTP/gRPC call (1-5ms) |

### ด้าน Operations

| หัวข้อ | Modular Monolith (ปัจจุบัน) | Microservices |
|--------|---------------------------|---------------|
| **Deploy** | 1 Docker image | N Docker images + orchestration |
| **Scale** | scale ทั้ง binary | scale เฉพาะ service ที่ต้องการ |
| **Monitoring** | 1 service logs | N service logs + distributed tracing |
| **Failure blast radius** | 1 bug = ทั้งระบบ down | 1 bug = 1 service down (ถ้าทำดี) |
| **Network latency** | 0 (in-process) | 1-50ms ต่อ hop |
| **Data consistency** | DB transaction | SAGA / eventual consistency |
| **Infra cost** | 1 pod + 1 DB | N pods + N DBs + message broker |

### สรุปง่ายๆ

```
Modular Monolith                     Microservices
──────────────────                   ──────────────
✅ ง่ายกว่า                           ✅ Scale อิสระ
✅ Debug ง่าย                         ✅ Deploy อิสระ
✅ ค่าใช้จ่ายน้อย                      ✅ Fault isolation
✅ Transaction ง่าย                   ✅ Tech diversity
❌ Scale ทั้งก้อน                      ❌ ซับซ้อนกว่า 10 เท่า
❌ Deploy ทั้งก้อน                     ❌ Network overhead
❌ 1 bug ล่มทั้งระบบ                  ❌ Distributed debugging
                                     ❌ ค่าใช้จ่ายสูงกว่า
```

---

## 7. แผนการ Extract — 4 Phases

```
         ตอนนี้              Phase 1          Phase 2          Phase 3           Phase 4
      ┌──────────┐       ┌──────────┐     ┌──────────┐    ┌──────────┐     ┌──────────┐
      │ Modular  │       │  แยก     │     │  แยก     │    │  แยก     │     │  Full    │
      │ Monolith │──────▶│  Worker  │────▶│ Auth +   │───▶│ CMI +    │────▶│  Event   │
      │          │       │  Service │     │ Quotation│    │ ExtDB   │     │ Sourcing │
      └──────────┘       └──────────┘     └──────────┘    └──────────┘     └──────────┘
          NOW             1-2 เดือน        3-6 เดือน       6-12 เดือน        1 ปี+
```

### Phase 1 — แยก Worker Service (ง่ายสุด)

**ทำไมง่าย:** `cmd/worker/` **แยก binary อยู่แล้ว** ไม่ share state กับ API

```
Before:                              After:
┌────────────────────┐               ┌────────────┐   ┌────────────┐
│    Single Deploy   │               │ API Pod    │   │ Worker Pod │
│  ┌─────┐ ┌──────┐ │               │  :20000    │   │  (Kafka    │
│  │ API │ │Worker│ │       →       │            │   │  Consumer) │
│  └─────┘ └──────┘ │               └──────┬─────┘   └──────┬─────┘
└────────────────────┘                     │                 │
                                           └────── Kafka ────┘
```

สิ่งที่ต้องทำ:
1. แยก Docker image (`Dockerfile.worker` มีแล้ว)
2. แยก K8s deployment (`worker-deployment.yaml` มีแล้ว)
3. ตรวจสอบ config + env vars แยก
4. Test ว่า worker run อิสระได้

### Phase 2 — แยก Auth + Quotation

```
┌────────────┐    gRPC/REST    ┌──────────────┐
│  API       │ ◄──────────────▶│ Auth Service │
│  Gateway   │                 │  :3001       │
│  :20000    │    Kafka Event  ├──────────────┤
│            │ ────────────────▶│ Quotation   │
└────────────┘                 │  Service    │
                               │  :3002       │
                               └──────────────┘
```

สิ่งที่ต้องทำ:
1. สร้าง gRPC definitions (`.proto`) สำหรับ Auth API
2. แยก DB schema (`auth_*` tables → auth_db)
3. ลบ auth module จาก API binary
4. API เรียก Auth ผ่าน gRPC client

### Phase 3 — แยก CMI + ExternalDB

```
┌────────────┐         ┌──────┐    ┌──────────┐
│ API        │◄────────│ Auth │    │  CMI     │
│ Gateway    │         └──────┘    │ Service  │
│            │◄──Kafka──────────── │          │
└────────────┘         ┌──────────┐└──────────┘
                       │ ExtDB   │
                       │ Service │──▶ Partner DBs
                       └─────────┘
```

### Phase 4 — Event Sourcing (Long-term)

```
┌──────────────────────────────────────────────────────┐
│                    Service Mesh (Istio)               │
│                                                      │
│  ┌──────┐  ┌──────────┐  ┌─────┐  ┌──────┐  ┌─────┐│
│  │ Auth │  │Quotation │  │ CMI │  │ExtDB │  │Notif││
│  │      │  │          │  │     │  │      │  │     ││
│  └──┬───┘  └────┬─────┘  └──┬──┘  └──┬───┘  └──┬──┘│
│     │           │            │        │         │   │
│     └───────────┴────────────┴────────┴─────────┘   │
│                    Kafka (Event Store)                │
│                                                      │
│  ┌───────┐  ┌──────────┐  ┌────────┐  ┌──────────┐  │
│  │auth_db│  │quotation │  │cmi_db  │  │extdb_fed │  │
│  │       │  │   _db    │  │        │  │          │  │
│  └───────┘  └──────────┘  └────────┘  └──────────┘  │
└──────────────────────────────────────────────────────┘
```

---

## 8. Module ไหนควร Extract ก่อน

### การให้คะแนน Extract Priority

| Module | ขอบเขตชัด | Traffic แยก | Business Value | Complexity | **Priority** |
|--------|----------|------------|---------------|-----------|-------------|
| **Worker** | 10/10 | 10/10 | 8/10 | 2/10 (ง่ายมาก) | **#1** |
| **Auth** | 9/10 | 8/10 | 9/10 | 5/10 | **#2** |
| **Quotation** | 8/10 | 7/10 | 8/10 | 6/10 | **#3** |
| **CMI** | 7/10 | 5/10 | 7/10 | 7/10 | **#4** |
| **ExternalDB** | 6/10 | 4/10 | 6/10 | 8/10 | **#5** |

### ทำไม Worker ก่อน?

1. **แยก binary อยู่แล้ว** (`cmd/worker/main.go`)
2. **ไม่มี synchronous dependency** กับ API (คุยผ่าน Kafka เท่านั้น)
3. **Dockerfile แยกอยู่แล้ว** (`Dockerfile.worker`)
4. **K8s deployment แยกอยู่แล้ว** (`worker-deployment.yaml`)
5. **Risk ต่ำ** — ถ้าพังก็เฉพาะ background job

### ทำไม Auth ที่สอง?

1. **Hexagonal ครบสมบูรณ์** — domain, ports, app, adapters
2. **Interface เล็ก** (UserRepository + TokenSigner = 2 interfaces, 2 methods)
3. **ใช้ร่วมหลาย service** — ทุก service ต้อง validate token
4. **Extract แล้วได้ประโยชน์ทันที** — centralized auth

---

## 9. สิ่งที่ต้องทำก่อน Extract

### Checklist ก่อน Extract Module เป็น Service

```
□  1. ไม่มี import ข้าม module โดยตรง
     ตรวจ: grep -r "internal/modules/auth" internal/modules/quotation/
     ต้องไม่พบ

□  2. มี Port Interface ครบ
     ทุก dependency ใช้ interface ไม่ใช่ concrete type

□  3. มี Unit Test ครอบคลุม
     Extract แล้ว test เดิมต้อง pass ไม่แก้

□  4. API Contract ชัดเจน
     HTTP endpoints + request/response schema documented

□  5. ไม่ share DB transaction ข้าม module
     ตรวจ: ไม่มี module A เรียก module B ภายใน tx เดียวกัน

□  6. Event contract defined
     Kafka message type + payload schema documented

□  7. Health check endpoint มี
     GET /healthz + GET /ready

□  8. Configuration แยกได้
     module config ไม่ผูกกับ config ของ module อื่น

□  9. Graceful shutdown
     SIGTERM → stop accepting → drain requests → close DB

□  10. Monitoring ready
      OTel spans + metrics + structured logs
```

### โปรเจกต์นี้ผ่านกี่ข้อ?

| # | ข้อ | สถานะ |
|---|-----|-------|
| 1 | ไม่ import ข้าม module | ✅ ผ่าน |
| 2 | Port Interface ครบ | ✅ ผ่าน |
| 3 | Unit Test ครอบคลุม | ⚠️ มีแต่ไม่ครบทุก layer |
| 4 | API Contract ชัด | ✅ Swagger docs |
| 5 | ไม่ share DB tx | ✅ ผ่าน |
| 6 | Event contract | ⚠️ มี Kafka แต่ยังไม่มี schema registry |
| 7 | Health check | ✅ `/healthz` + `/ready` |
| 8 | Config แยกได้ | ✅ ผ่าน |
| 9 | Graceful shutdown | ✅ signal handling |
| 10 | Monitoring | ✅ OTel + zerolog |

**ผ่าน 8/10 ข้อ** — ขาดแค่ coverage ครบ + schema registry

---

## 10. Anti-Patterns ที่ต้องหลีกเลี่ยง

### 10.1 Distributed Monolith

```
❌ ผิด: แยก service แต่ต้อง deploy พร้อมกันทุกตัว
        → ได้แค่ complexity ไม่ได้ independence

✅ ถูก: service A deploy ได้โดย service B ไม่สนใจ
```

**ป้องกัน:** ใช้ versioned API + backward compatibility

### 10.2 Shared Database

```
❌ ผิด: Auth Service + Quotation Service → shared anc_portal DB
        → coupled ผ่าน schema changes

✅ ถูก: Auth Service → auth_db
       Quotation Service → quotation_db
       สื่อสารผ่าน API/Event
```

### 10.3 Synchronous Chain

```
❌ ผิด: A → B → C → D  (latency = sum ทุก hop)
        A ล่มถ้า D ล่ม

✅ ถูก: A → Kafka → B processes async
       A ตอบ client ทันที, B ทำงาน background
```

**โปรเจกต์นี้ทำถูกแล้ว** — API → Kafka → Worker pattern

### 10.4 premature Extraction

```
❌ ผิด: เริ่มโปรเจกต์ด้วย 10 microservices เลย
        → ขอบเขตยังไม่ชัด → แก้ไม่ถูก → refactor ข้าม service ยากมาก

✅ ถูก: เริ่มด้วย Modular Monolith → extract เมื่อมี pain point จริง
```

**โปรเจกต์นี้ทำถูกแล้ว** — Modular Monolith ก่อน, extract เมื่อพร้อม

### 10.5 No Observability

```
❌ ผิด: แยก service แต่ไม่มี distributed tracing
        → debug production incident ใช้เวลาเป็นวัน

✅ ถูก: OTel tracing ข้าม service + Kafka propagation
```

**โปรเจกต์นี้ทำถูกแล้ว** — OTel + W3C propagation ข้าม Kafka

---

## 11. สรุป

### โครงสร้างนี้เหมาะกับ Microservices ไหม?

**ใช่ เหมาะมาก** — แต่ยังไม่ต้องทำตอนนี้

### เหตุผล:

| ทำถูกแล้ว | ทำไมสำคัญ |
|-----------|----------|
| Hexagonal Architecture | ตัด module ออกเป็น service ได้ทันที |
| Event-Driven (Kafka) | service คุยกันผ่าน event ไม่ต้อง function call |
| Multi-Driver Database | ExternalConn interface รองรับ postgres + mysql |
| Route-Level Auth (JWT + API Key) | module ตัดสินใจ auth เอง → extract แล้วไม่ต้องแก้ |
| No cross-module imports | ไม่มี coupling ข้าม module |
| OTel Distributed Tracing | trace ข้าม service ได้ทันที |
| Circuit Breaker (httpclient) | ป้องกัน cascade failure ข้าม service |
| Feature Toggles | ปิด infra ที่ไม่ต้องการแต่ละ service ได้ |
| Worker แยก binary + health probe | extract เป็น service แยกได้ใน 1-2 วัน |

### คำแนะนำ:

> **อยู่กับ Modular Monolith จนกว่าจะมี pain point จริง**
> (ทีมโตจนชนกัน, scale ไม่พอ, deploy ถี่จนชนกัน)
> แล้วค่อย extract ตาม Phase 1 → 2 → 3 → 4
>
> **ไม่มี penalty** ที่จะ extract ทีหลัง เพราะโครงสร้างพร้อมแล้ว

---

> v3.0 — March 2026 | ANC Portal Backend Team
