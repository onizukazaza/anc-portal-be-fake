# CMI Module — Interface Tracing & Testing Deep Dive

> เอกสารนี้เจาะลึก **การไล่ interface ทุก layer** ของ module CMI  
> ตั้งแต่ domain → ports → app → adapters → module.go  
> และเจาะลึก **ระบบ Fake Test** ที่เป็นหัวใจของการ test โดยไม่ต้องต่อ DB

---

## Table of Contents

- [Section 1: Fake Test — ทำไมต้องมี และ concept คืออะไร](#section-1-fake-test)
- [Section 2: Interface Layer — ไล่ตาม data flow](#section-2-interface-layer)
- [Section 3: Layer-by-Layer Deep Dive](#section-3-layer-by-layer)
- [Section 4: Dependency Wiring — module.go](#section-4-dependency-wiring)
- [Section 5: Test ทุกชั้น — ใครใช้ fake อะไร](#section-5-test-ทุกชั้น)

---

## Section 1: Fake Test

### 1.1 ปัญหา — ถ้าไม่มี Fake จะเกิดอะไร?

สมมุติต้อง test ว่า handler ส่ง 404 เมื่อ `job_id` ไม่เจอ:

```
❌ ถ้า test ต่อ DB จริง:

   1. ต้องมี PostgreSQL server running
   2. ต้อง migrate schema ให้ตรง
   3. ต้อง seed data (หรือลบ data เพื่อจำลอง "ไม่เจอ")
   4. ต้องรอ network round-trip
   5. test ช้า (วินาที ↗)
   6. test flaky (DB ล่ม → test พัง)
   7. CI ต้อง set DB service
```

```
✅ ถ้า test ใช้ Fake:

   1. ไม่ต้องมี DB — Fake return ค่าที่ control ได้
   2. ไม่ต้อง migrate / seed อะไร
   3. test เร็ว (milliseconds)
   4. test stable 100% (ไม่มี external dependency)
   5. CI ง่าย (go test ./... จบ)
```

### 1.2 Fake คืออะไร — อธิบายแบบเห็นภาพ

```
┌──────────────────────────────────────────────────────────────────────┐
│                         Production (ของจริง)                        │
│                                                                      │
│  Handler ──→ Service ──→ Repository (Postgres) ──→ PostgreSQL DB     │
│                              ↑                                       │
│                    implements ports.CMIPolicyRepository               │
└──────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────┐
│                         Test (จำลอง)                                 │
│                                                                      │
│  Handler ──→ Service ──→ fakeRepo (struct ธรรมดา) ──→ ไม่มี DB      │
│                              ↑                                       │
│                    implements ports.CMIPolicyRepository               │
│                    return ค่าที่ตั้งไว้ล่วงหน้า                       │
└──────────────────────────────────────────────────────────────────────┘
```

**ทั้ง Repository จริง และ fakeRepo implement interface เดียวกัน**  
Service ไม่รู้ (และไม่สน) ว่าข้างหลังเป็น Postgres หรือ Fake

### 1.3 Interface ที่เป็นตัวเชื่อม

```go
// ports/repository.go — "สัญญา" ที่ทุกฝ่ายต้องทำตาม
type CMIPolicyRepository interface {
    JobExists(ctx context.Context, jobID string) (bool, error)
    FindPolicyByJobID(ctx context.Context, jobID string) (*domain.CMIPolicy, error)
}
```

Interface นี้เปรียบเหมือน **ปลั๊กไฟ** — อะไรก็ได้ที่มีขาตรงก็เสียบได้:

```
┌───────────────────────┐
│  CMIPolicyRepository  │ ← ปลั๊กไฟ (interface)
│  ├─ JobExists()       │
│  └─ FindPolicyByJobID()│
└───────────┬───────────┘
            │
    ┌───────┴───────┐
    │               │
    ▼               ▼
┌────────┐    ┌──────────┐
│Postgres│    │ fakeRepo │  ← ขาปลั๊ก 2 แบบ (ทำงานเหมือนกัน)
│  Repo  │    │ (test)   │
└────────┘    └──────────┘
```

### 1.4 Fake ของ CMI — โค้ดจริง

#### Fake ใน `app/fakes_test.go` (Service level test)

```go
// ─── ใน app/fakes_test.go ───

type fakeCMIRepo struct {
    exists   bool              // ← control ว่า JobExists return อะไร
    existErr error             // ← control ว่า JobExists error หรือเปล่า
    policy   *domain.CMIPolicy // ← control ว่า FindPolicy return อะไร
    findErr  error             // ← control ว่า FindPolicy error หรือเปล่า
}

func (f *fakeCMIRepo) JobExists(_ context.Context, _ string) (bool, error) {
    return f.exists, f.existErr
    //     ↑ return ค่าที่ set ไว้ตรงๆ — ไม่ query SQL
}

func (f *fakeCMIRepo) FindPolicyByJobID(_ context.Context, _ string) (*domain.CMIPolicy, error) {
    return f.policy, f.findErr
    //     ↑ return ค่าที่ set ไว้ตรงๆ — ไม่ query SQL
}
```

**จุดสำคัญ:**
- `_ context.Context` — ไม่ใช้ context เพราะไม่มี network call
- `_ string` — ไม่สนว่า jobID อะไร เพราะ return ค่าคงที่
- struct fields เป็น **ปุ่มบังคับ** ผลลัพธ์ของ test

#### Fake ใน `adapters/http/fakes_test.go` (Handler level test)

```go
// ─── ใน adapters/http/fakes_test.go ───

type fakeRepo struct {
    exists   bool
    existErr error
    policy   *domain.CMIPolicy
    findErr  error
}

// Compile-time check — ถ้า interface เปลี่ยน จะ compile ไม่ผ่านทันที
var _ ports.CMIPolicyRepository = (*fakeRepo)(nil)

func (f *fakeRepo) JobExists(_ context.Context, _ string) (bool, error) {
    return f.exists, f.existErr
}

func (f *fakeRepo) FindPolicyByJobID(_ context.Context, _ string) (*domain.CMIPolicy, error) {
    return f.policy, f.findErr
}

var errDB = errors.New("db connection refused")
```

### 1.5 ทำไม Fake มี 2 ที่? (app/ กับ adapters/http/)

```
app/fakes_test.go              ← Fake สำหรับ test "Service logic"
adapters/http/fakes_test.go    ← Fake สำหรับ test "Handler HTTP behavior"
```

| ที่ Fake อยู่ | Test อะไร | สร้างอะไร | focus ที่ |
|---|---|---|---|
| `app/fakes_test.go` | `service_test.go` | `NewService(fakeRepo)` | business logic ถูกไหม? |
| `adapters/http/fakes_test.go` | `handler_test.go` | `NewService(fakeRepo) → NewHandler(svc)` | HTTP status/body ถูกไหม? |

```
app/service_test.go:
  ┌──────────────────────────────────────────────┐
  │ Test "บทบาทของ Service"                       │
  │                                                │
  │ fakeRepo → NewService(repo) → svc.Method()    │
  │                                                │
  │ ✓ ถ้า repo.JobExists = false → ErrJobNotFound │
  │ ✓ ถ้า repo error → return error               │
  │ ✓ ถ้า repo OK → return policy                 │
  └──────────────────────────────────────────────┘

adapters/http/handler_test.go:
  ┌──────────────────────────────────────────────┐
  │ Test "บทบาทของ Handler"                       │
  │                                                │
  │ fakeRepo → NewService(repo) → NewHandler(svc) │
  │ → fiber.Test(req) → assert status + JSON body │
  │                                                │
  │ ✓ success → 200 + {"status":"OK"}             │
  │ ✓ not found → 404 + trace_id                  │
  │ ✓ DB error → 500 + trace_id                   │
  └──────────────────────────────────────────────┘
```

### 1.6 Compile-Time Interface Check — `var _ Interface = (*Fake)(nil)`

```go
var _ ports.CMIPolicyRepository = (*fakeRepo)(nil)
```

**บรรทัดนี้สำคัญมาก** — มันบอก Go compiler ว่า:

> "fakeRepo ต้อง implement CMIPolicyRepository ทุก method  
> ถ้าไม่ครบ → **compile error ทันที** ไม่ต้องรอ run test"

```
สถานการณ์: เพิ่ม method ใหม่ใน interface

ports/repository.go:
  type CMIPolicyRepository interface {
      JobExists(...)
      FindPolicyByJobID(...)
+     DeletePolicy(...)          ← เพิ่ม method ใหม่
  }

ถ้า fakeRepo ไม่ implement DeletePolicy():

  $ go build
  ❌ compile error:
     "fakeRepo does not implement ports.CMIPolicyRepository
      (missing method DeletePolicy)"

→ รู้ทันทีว่าต้องไปเพิ่ม method ใน fake ด้วย
```

**ถ้าไม่มีบรรทัดนี้** → จะรู้ตอน runtime เท่านั้น (ตอน assign fake ให้ interface) ซึ่งช้ากว่ามาก

### 1.7 Fake vs Mock — ต่างกันยังไง?

| Feature | Fake (โปรเจกต์นี้ใช้) | Mock (testify, gomock) |
|---|---|---|
| **สร้างยังไง** | เขียน struct เอง | Library generate ให้ |
| **ควบคุมผลลัพธ์** | set struct fields | `.Return(value)` chain |
| **ตรวจ compile-time** | ✅ `var _ = (*fake)(nil)` | ❌ ตรวจตอน runtime |
| **external dependency** | ❌ ไม่มี | ✅ ต้อง import library |
| **อ่านง่าย** | ✅ Go ธรรมดา | ⚠️ ต้องรู้ API ของ library |
| **ใช้ใน project นี้** | ✅ เท่านี้เท่านั้น | ❌ ห้ามใด |

**กฎเหล็ก:** ห้ามใช้ `testify`, `gomock`, `mockery`, `ginkgo` ใช้ `internal/testkit` + hand-written fakes เท่านั้น

### 1.8 สรุป Fake — Mental Model

```
Fake = "ตัวแทน" ของ dependency ที่ควบคุมได้ 100%

┌─── Fake struct ───┐
│ exists:   true    │ ← "ปุ่มบังคับ" ผลลัพธ์
│ existErr: nil     │
│ policy:   &CMI{}  │
│ findErr:  nil     │
└───────────────────┘
        ↓
เมื่อ Service เรียก repo.JobExists()
        ↓
return true, nil    ← return ค่าที่ set ไว้ ไม่ query DB

ผลลัพธ์: test control ได้ทุก scenario โดยแค่เปลี่ยนค่าใน struct
```

---

## Section 2: Interface Layer — ไล่ตามสายข้อมูล

### 2.1 ภาพรวม: Interface อยู่ตรงไหนบ้าง?

```
Module CMI — Interface Chain

  ┌────────────────────────────────────────────────────────────────────┐
  │                                                                    │
  │  module.go (Composition Root)                                      │
  │  ├─ สร้าง: CMIPolicyRepository (concrete = Postgres)              │
  │  ├─ สร้าง: *Service           (inject repo interface)            │
  │  └─ สร้าง: CMIController      (inject service → return interface) │
  │                                                                    │
  │  ┌────────┐    ┌──────────┐    ┌─────────────┐    ┌────────────┐  │
  │  │domain/ │ ←──│ ports/   │ ←──│ app/        │ ←──│ adapters/  │  │
  │  │        │    │          │    │             │    │            │  │
  │  │CMIPolicy│   │CMIPolicy │    │Service      │    │Handler     │  │
  │  │MotorInfo│   │Repository│    │ .repo field │    │ .service   │  │
  │  │AssetInfo│   │(interface)│   │ (interface) │    │ (concrete) │  │
  │  │...      │   │          │    │             │    │            │  │
  │  └────────┘    └──────────┘    └─────────────┘    │Controller  │  │
  │                                                    │(interface) │  │
  │       純粋 struct     contract        logic        │Postgres    │  │
  │       ห้าม import   ห้าม impl     inject port    │Repo        │  │
  │                                                    │(concrete)  │  │
  │                                                    └────────────┘  │
  └────────────────────────────────────────────────────────────────────┘
```

### 2.2 Interface ที่มีในโปรเจกต์ CMI (ครบทุกตัว)

| # | Interface | ไฟล์ | Method(s) | ใครใช้ | ใคร Implement |
|---|---|---|---|---|---|
| 1 | `CMIPolicyRepository` | `ports/repository.go` | `JobExists`, `FindPolicyByJobID` | Service (app/) | `postgres.CMIPolicyRepository`, `fakeRepo`, `fakeCMIRepo` |
| 2 | `CMIController` | `adapters/http/controller.go` | `GetPolicyByJobID` | `module.go` | `Handler` |
| 3 | `pgx.Row` (external) | pgx library | `Scan` | `scanCMIPolicy()` | `fakeRow` (ใน repo test) |

### 2.3 แผนภาพการ Implement

```
                    ports.CMIPolicyRepository
                    ┌──────────────────────┐
                    │ JobExists()          │
                    │ FindPolicyByJobID()  │
                    └──────────┬───────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
              ▼                ▼                ▼
  ┌──────────────────┐ ┌──────────────┐ ┌──────────────┐
  │ postgres/        │ │ app/         │ │ http/        │
  │ CMIPolicyRepo    │ │ fakeCMIRepo  │ │ fakeRepo     │
  │                  │ │              │ │              │
  │ ★ ต่อ DB จริง    │ │ ★ test svc   │ │ ★ test HTTP  │
  │ ★ SQL queries   │ │ ★ ไม่มี DB   │ │ ★ ไม่มี DB   │
  │ ★ production    │ │ ★ unit test  │ │ ★ unit test  │
  └──────────────────┘ └──────────────┘ └──────────────┘
```

---

## Section 3: Layer-by-Layer Deep Dive

### 3.1 Domain Layer — Pure Data (ชั้นในสุด)

```
internal/modules/cmi/domain/cmi.go
```

```go
package domain

type CMIPolicy struct {
    JobID   string       `json:"job_id"`
    Motor   *MotorInfo   `json:"motor_info"`
    Asset   *AssetInfo   `json:"asset_info"`
    Insured *InsuredInfo  `json:"insured"`
    // ... 20+ fields
}

type MotorInfo struct {
    Year  string `json:"year"`
    Brand string `json:"brand"`
    Model string `json:"model"`
}
// ... AssetInfo, InsuredInfo, PolicyDate, AddressSet, AgentInfo, QuoteInfo
```

**กฎ Domain:**

```
✅ Pure struct — ไม่มี logic, ไม่มี method, ไม่มี I/O
✅ import ได้แค่ standard library (encoding/json, time)
❌ ห้าม import package อื่นใน project
❌ ห้าม import ports/ หรือ app/ หรือ adapters/
```

**ทำไม?** — Domain เป็น "ศูนย์กลาง" ที่ทุก layer ใช้ร่วมกัน ถ้ามัน import ชั้นอื่น → circular dependency ทันที

```
Dependency Direction:

  domain ←── ports ←── app ←── adapters
    ↑          ↑         ↑         ↑
    │          │         │         │
  ห้าม      import    import    import
  import    domain    domain    app
  ใคร                 + ports
```

### 3.2 Ports Layer — Contract (สัญญา)

```
internal/modules/cmi/ports/repository.go
```

```go
package ports

import (
    "context"
    "github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/domain"
)

type CMIPolicyRepository interface {
    JobExists(ctx context.Context, jobID string) (bool, error)
    FindPolicyByJobID(ctx context.Context, jobID string) (*domain.CMIPolicy, error)
}
```

**กฎ Ports:**

```
✅ Go interface เท่านั้น — ไม่มี struct, ไม่มี function ธรรมดา
✅ method แรกต้องเป็น context.Context เสมอ
✅ return type ใช้ domain types (ไม่ใช่ adapter types)
✅ interface เล็ก: 1-3 methods (Interface Segregation Principle)
❌ ห้ามมี implementation ใดๆ
❌ ห้าม import app/ หรือ adapters/
```

**ทำไม interface ต้องเล็ก?**

```
❌ Interface ใหญ่เกินไป (ยาก test, ยาก implement):

  type BigRepo interface {
      JobExists(...)
      FindPolicyByJobID(...)
      FindAll(...)
      Create(...)
      Update(...)
      Delete(...)
      Count(...)
      Search(...)     ← 8 methods = fake ต้องเขียน 8 methods
  }

✅ Interface เล็กพอดี:

  type CMIPolicyRepository interface {
      JobExists(...)
      FindPolicyByJobID(...)    ← 2 methods = fake เขียนง่าย
  }
```

### 3.3 App Layer — Business Logic

```
internal/modules/cmi/app/service.go
```

```go
package app

type Service struct {
    repo ports.CMIPolicyRepository    // ← ถือ interface ไม่ใช่ concrete type
}

func NewService(repo ports.CMIPolicyRepository) *Service {
    return &Service{repo: repo}         // ← constructor injection
}
```

**ไล่ logic ของ `GetPolicyByJobID`:**

```go
func (s *Service) GetPolicyByJobID(ctx context.Context, jobID string) (*domain.CMIPolicy, error) {
    // STEP 1: ตรวจว่า job มีจริงไหม
    exists, err := s.repo.JobExists(ctx, jobID)
    //              ↑ เรียกผ่าน interface — ไม่รู้ว่าเป็น Postgres หรือ Fake
    if err != nil {
        return nil, err       // ← early return (ไม่ใช้ else)
    }
    if !exists {
        return nil, ErrJobNotFound    // ← sentinel error
    }

    // STEP 2: ดึง policy
    policy, err := s.repo.FindPolicyByJobID(ctx, jobID)
    //              ↑ เรียกผ่าน interface เดียวกัน
    if err != nil {
        return nil, err
    }

    return policy, nil
}
```

**กฎ App:**

```
✅ Dependency ต้องเป็น interface (inject ผ่าน constructor)
✅ ประกาศ sentinel errors ที่นี่ (ErrJobNotFound, ErrJobIDRequired)
✅ ใช้ early return (ไม่ใช้ if-else)
❌ ห้าม import adapter types (postgres, fiber, pgx)
❌ ห้ามสร้าง concrete dependency ภายใน
```

**ทำไม Service ถือ interface?**

```
// ❌ ผูกกับ Postgres ตายตัว
type Service struct {
    repo *postgres.CMIPolicyRepository    // ← ถ้าเปลี่ยน DB ต้องแก้ service
}

// ✅ ยืดหยุ่น — ใส่อะไรก็ได้ที่ implement interface
type Service struct {
    repo ports.CMIPolicyRepository        // ← เปลี่ยน DB ไม่ต้องแก้ service
}
```

### 3.4 Adapters Layer — HTTP + Postgres

#### 3.4.1 HTTP Handler

```
internal/modules/cmi/adapters/http/handler.go
internal/modules/cmi/adapters/http/controller.go
```

```go
// controller.go — interface สำหรับ route registration
type CMIController interface {
    GetPolicyByJobID(ctx *fiber.Ctx) error
}

// handler.go — implementation
type Handler struct {
    service *app.Service    // ← ถือ concrete Service (ไม่ใช่ interface)
}

func NewHandler(service *app.Service) *Handler {
    return &Handler{service: service}   // ← return concrete
}

func NewCMIController(service *app.Service) CMIController {
    return &Handler{service: service}   // ← return interface
}
```

**Handler `GetPolicyByJobID` — ไล่ทีละบรรทัด:**

```go
func (h *Handler) GetPolicyByJobID(c *fiber.Ctx) error {
    // 1. เริ่ม OTel tracing span
    ctx, span := appOtel.Tracer(appOtel.TracerCMIHandler).Start(c.UserContext(), "GetPolicyByJobID")
    defer span.End()

    // 2. ดึง job_id จาก URL path
    jobID := c.Params("job_id")

    // 3. validate input
    if jobID == "" {
        return dto.ErrorWithTrace(c, 400, "job_id is required", dto.TraceCMIJobIdRequired)
        //                                                        ↑ trace_id = "cmi-job-id-required"
    }

    // 4. เรียก service (business logic)
    policy, err := h.service.GetPolicyByJobID(ctx, jobID)

    // 5. แปลง error → HTTP response
    if err != nil {
        if errors.Is(err, app.ErrJobNotFound) {
            return dto.ErrorWithTrace(c, 404, "job not found", dto.TraceCMIJobNotFound)
        }
        return dto.ErrorWithTrace(c, 500, "internal error", dto.TraceCMIInternalError)
    }

    // 6. success
    return dto.Success(c, 200, policy)
}
```

**Handler ทำอะไรบ้าง — ทำแค่ 3 อย่าง:**

```
┌─────────────────────────────────────────────────┐
│ Handler responsibilities:                        │
│                                                  │
│ 1. แปลง HTTP → Go type  (c.Params → string)    │
│ 2. เรียก Service         (h.service.Method())   │
│ 3. แปลง result → HTTP   (dto.Success/Error)     │
│                                                  │
│ ❌ ไม่มี SQL                                     │
│ ❌ ไม่มี business rule ("ถ้า X ให้ทำ Y")          │
│ ❌ ไม่มี data transform                          │
└─────────────────────────────────────────────────┘
```

#### 3.4.2 Postgres Repository

```
internal/modules/cmi/adapters/postgres/repository.go
```

```go
type CMIPolicyRepository struct {
    pool *pgxpool.Pool    // ← ถือ DB connection pool
}

func NewCMIPolicyRepository(pool *pgxpool.Pool) *CMIPolicyRepository {
    return &CMIPolicyRepository{pool: pool}
}
```

**Repository implement interface ของ ports:**

```
ports.CMIPolicyRepository (interface)
        ↑
        │ implements
        │
postgres.CMIPolicyRepository (struct)
        │
        ├─ JobExists()         → SELECT EXISTS(SELECT 1 FROM job WHERE id = $1)
        └─ FindPolicyByJobID() → SELECT ... 20+ LEFT JOINs ... WHERE j.id = $1
```

**SQL Architecture — แยก fragment:**

```go
func buildFindPolicyQuery() string {
    return fmt.Sprintf(`SELECT
        %s,    // ← sqlSelectJobFields()     → j.id, j.job_type, ...
        %s,    // ← sqlMotorInfo()           → jsonb_build_object(...)
        %s,    // ← sqlAssetInfo()           → jsonb_build_object(...)
        %s,    // ← sqlInsured()             → json_build_object(...)
        %s,    // ← sqlPolicyDates()         → jsonb_build_object(...)
        %s,    // ← sqlAddressSet()          → jsonb_build_object(...)
        %s,    // ← sqlAgentInfo()           → jsonb_build_object(...)
        %s,    // ← sqlProducts()            → COALESCE(jsonb_agg(...))
        %s,    // ← sqlPayments()            → COALESCE(jsonb_agg(...))
        %s,    // ← sqlInsuranceDocs()       → COALESCE(jsonb_agg(...))
        %s,    // ← sqlInsuredDocs()         → COALESCE(jsonb_agg(...))
        %s,    // ← sqlQuoteInfo()           → jsonb_build_object(...)
        j.created_datetime,
        j.updated_datetime
        %s`,   // ← sqlFromJoins()          → FROM job j LEFT JOIN ...
        ...)
}
```

**ทำไมแยก fragment?**

```
✅ อ่าน/แก้ได้ทีละส่วน — ไม่ต้อง scroll SQL 200 บรรทัด
✅ test ได้ทีละ fragment — ตรวจว่า SQL มี keyword ที่ต้องการ
✅ reuse ได้ — ถ้ามี query อื่นที่ใช้ motor info เหมือนกัน
```

---

## Section 4: Dependency Wiring — module.go

```
internal/modules/cmi/module.go
```

### 4.1 Wiring Chain — สร้างทุกอย่างทีละชั้น

```go
func Register(router fiber.Router, deps module.Deps) {
    // STEP 1: ดึง external DB connection
    conn, err := deps.DB.External("meprakun")
    pool, err := database.PgxPool(conn)

    // STEP 2: สร้าง Repository (concrete → ต่อ DB จริง)
    repo := cmipg.NewCMIPolicyRepository(pool)
    //      ↑ return *postgres.CMIPolicyRepository
    //      (implements ports.CMIPolicyRepository)

    // STEP 3: สร้าง Service (inject interface)
    service := app.NewService(repo)
    //                        ↑ repo ถูก cast เป็น ports.CMIPolicyRepository โดยอัตโนมัติ
    //                        เพราะ Go implicit interface satisfaction

    // STEP 4: สร้าง Controller (return interface)
    controller := cmihttp.NewCMIController(service)
    //           ↑ return CMIController (interface)
    //           ข้างในเป็น *Handler (concrete)

    // STEP 5: ผูก route
    group := router.Group("/cmi")
    group.Get("/:job_id/request-policy-single-cmi",
        deps.Middleware.JWTAuth,       // ← middleware
        controller.GetPolicyByJobID,   // ← handler method
    )
}
```

### 4.2 สิ่งที่เกิดขึ้นจริงๆ ตอน runtime

```
Startup:
  module.go สร้าง chain → Pool → Repo → Service → Handler

  ┌─────────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
  │ pgxpool.Pool│────→│ Postgres │────→│ Service  │────→│ Handler  │
  │             │     │ Repo     │     │          │     │          │
  │ ★ DB conn   │     │ ★ SQL    │     │ ★ logic  │     │ ★ HTTP   │
  └─────────────┘     └──────────┘     └──────────┘     └──────────┘

                      implements        uses             implements
                      CMIPolicy         CMIPolicy        CMI
                      Repository        Repository       Controller

Per-Request:
  GET /cmi/J001/request-policy-single-cmi
       ↓
  [JWTAuth middleware] → ok
       ↓
  Handler.GetPolicyByJobID(c)
       ↓
  h.service.GetPolicyByJobID(ctx, "J001")    ← Service (ตัวเดิมที่สร้างตอน startup)
       ↓
  s.repo.JobExists(ctx, "J001")              ← Postgres Repo (ตัวเดิม)
       ↓
  s.repo.FindPolicyByJobID(ctx, "J001")      ← Postgres Repo (ตัวเดิม)
       ↓
  dto.Success(c, 200, policy)
```

### 4.3 Implicit Interface Satisfaction — Go Magic

```go
// Go ไม่ต้อง "ประกาศ" ว่า implement interface
// แค่มี method ครบ → ถือว่า implement

// postgres.CMIPolicyRepository มี method:
//   - JobExists(ctx, string) (bool, error)         ✅ ตรง
//   - FindPolicyByJobID(ctx, string) (*CMIPolicy, error) ✅ ตรง
//
// ports.CMIPolicyRepository ต้องการ:
//   - JobExists(ctx, string) (bool, error)         ✅ ตรง
//   - FindPolicyByJobID(ctx, string) (*CMIPolicy, error) ✅ ตรง
//
// → Go compiler: "OK, postgres.CMIPolicyRepository implements ports.CMIPolicyRepository"

repo := cmipg.NewCMIPolicyRepository(pool)
service := app.NewService(repo)   // ← repo auto-cast เป็น interface
```

---

## Section 5: Test ทุกชั้น — ใครใช้ Fake อะไร

### 5.1 ภาพรวม Test ทั้ง Module

```
internal/modules/cmi/
├── app/
│   ├── service_test.go    ← TEST: Service logic
│   │   ใช้ fakeCMIRepo    ← FAKE: ports.CMIPolicyRepository
│   │
│   └── fakes_test.go      ← FAKE DEFINITION
│
├── adapters/http/
│   ├── handler_test.go    ← TEST: HTTP handler
│   │   ใช้ fakeRepo       ← FAKE: ports.CMIPolicyRepository
│   │
│   └── fakes_test.go      ← FAKE DEFINITION
│
├── adapters/postgres/
│   └── repository_test.go ← TEST: SQL scan logic
│       ใช้ fakeRow         ← FAKE: pgx.Row
│
└── integration_test.go    ← TEST: ต่อ DB จริง (skip ถ้าไม่มี DSN)
```

### 5.2 Service Test — ทดสอบ Business Logic

```go
// app/service_test.go

func TestGetPolicyByJobID(t *testing.T) {
    tests := []struct {
        name    string
        repo    *fakeCMIRepo     // ← fakeCMIRepo ที่ control ผลลัพธ์ได้
        jobID   string
        wantErr error
        wantID  string
    }{
        {
            name:   "success",
            repo:   &fakeCMIRepo{exists: true, policy: samplePolicy},
            //       ↑ JobExists return true, FindPolicy return samplePolicy
            wantID: "job-001",
        },
        {
            name:    "job not found",
            repo:    &fakeCMIRepo{exists: false},
            //       ↑ JobExists return false → Service ต้อง return ErrJobNotFound
            wantErr: ErrJobNotFound,
        },
        {
            name:    "repo error on JobExists",
            repo:    &fakeCMIRepo{existErr: dbErr},
            //       ↑ JobExists return error → Service ต้อง return error เดียวกัน
            wantErr: dbErr,
        },
        {
            name:    "repo error on FindPolicy",
            repo:    &fakeCMIRepo{exists: true, findErr: dbErr},
            //       ↑ JobExists OK แต่ FindPolicy error → Service ต้อง return error
            wantErr: dbErr,
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            svc := NewService(tc.repo)    // ← inject fake ผ่าน constructor
            policy, err := svc.GetPolicyByJobID(context.Background(), tc.jobID)
            // ... assert ...
        })
    }
}
```

**Test 4 scenarios — ครอบคลุมทุก branch ของ Service:**

```
GetPolicyByJobID logic:
                            ┌── existErr? ──→ return err   [test: "repo error on JobExists"]
                            │
JobExists(jobID) ───────────┤
                            │
                            └── !exists? ──→ return ErrJobNotFound  [test: "job not found"]
                                    │
                                  exists
                                    │
                                    ▼
                            ┌── findErr? ──→ return err   [test: "repo error on FindPolicy"]
                            │
FindPolicyByJobID(jobID) ──┤
                            │
                            └── success ──→ return policy  [test: "success"]
```

### 5.3 Handler Test — ทดสอบ HTTP Response

```go
// adapters/http/handler_test.go

// setup: สร้าง Fiber app + handler จาก fakeRepo
func setupApp(repo *fakeRepo) *fiber.App {
    fiberApp := fiber.New()
    svc := app.NewService(repo)    // ← fake inject ตรงนี้
    h := NewHandler(svc)
    fiberApp.Get("/cmi/:job_id/request-policy-single-cmi", h.GetPolicyByJobID)
    return fiberApp
}

// helper: ยิง request + parse response
func doRequest(t *testing.T, fiberApp *fiber.App, jobID string) (int, dto.ApiResponse) {
    req := httptest.NewRequest("GET", "/cmi/"+jobID+"/request-policy-single-cmi", nil)
    resp, _ := fiberApp.Test(req, -1)    // ← ใช้ Fiber test mode (ไม่ต้อง start server จริง)
    // ... read body, unmarshal JSON ...
    return statusCode, apiResp
}
```

**Test แต่ละ scenario:**

```go
// ✅ Success — 200 + OK
func TestGetPolicyByJobID_Success(t *testing.T) {
    repo := &fakeRepo{exists: true, policy: &domain.CMIPolicy{...}}
    fiberApp := setupApp(repo)
    statusCode, apiResp := doRequest(t, fiberApp, "job-001")

    testkit.Equal(t, statusCode, 200, "status code")
    testkit.Equal(t, apiResp.Status, "OK", "response status")
}

// ❌ Not Found — 404 + trace_id
func TestGetPolicyByJobID_JobNotFound(t *testing.T) {
    repo := &fakeRepo{exists: false}
    fiberApp := setupApp(repo)
    statusCode, apiResp := doRequest(t, fiberApp, "missing-job")

    testkit.Equal(t, statusCode, 404, "status code")
    testkit.Contains(t, extractTraceID(t, apiResp), "cmi-job-not-found", "trace_id")
}

// 💥 DB Error — 500 + trace_id
func TestGetPolicyByJobID_RepoError(t *testing.T) {
    repo := &fakeRepo{existErr: errDB}
    fiberApp := setupApp(repo)
    statusCode, apiResp := doRequest(t, fiberApp, "job-001")

    testkit.Equal(t, statusCode, 500, "status code")
    testkit.Contains(t, extractTraceID(t, apiResp), "cmi-internal-error", "trace_id")
}
```

### 5.4 Repository Test — ทดสอบ SQL Scan Logic

Repository test **ไม่ test SQL query** (ต้อง integration test ถึงจะ test SQL ได้)  
แต่ test **scanCMIPolicy()** — แปลง row → struct ถูกไหม

```go
// adapters/postgres/repository_test.go

// fakeRow — จำลอง pgx.Row (ผลลัพธ์ 1 row จาก DB)
type fakeRow struct {
    values []any    // ← ค่าที่ "แกล้ง" ว่า DB return มา
    err    error    // ← error ที่ "แกล้ง" ว่า scan พัง
}

func (f *fakeRow) Scan(dest ...any) error {
    if f.err != nil {
        return f.err    // ← จำลอง scan error
    }
    // type-switch: assign ค่าจาก f.values ไปยัง dest pointers
    for i, val := range f.values {
        switch d := dest[i].(type) {
        case *string: *d = val.(string)
        case *bool:   *d = val.(bool)
        case *[]byte: *d = val.([]byte)
        // ... 6 types
        }
    }
    return nil
}
```

**Test scenarios:**

```go
// ✅ Scan success — ค่าครบ, JSON valid
func TestScanCMIPolicy_Success(t *testing.T) {
    row := buildSuccessRow(t)    // ← fakeRow ที่มีค่า 25 columns
    pol, err := scanCMIPolicy(row)

    testkit.NoError(t, err)
    testkit.Equal(t, pol.JobID, "job-001", "JobID")
    testkit.Equal(t, pol.Motor.Brand, "Toyota", "Motor.Brand")
}

// ❌ Scan error
func TestScanCMIPolicy_ScanError(t *testing.T) {
    row := &fakeRow{err: errors.New("no rows")}
    pol, err := scanCMIPolicy(row)

    testkit.Error(t, err)
    testkit.Nil(t, pol, "policy")
}

// 🔹 NULL JSON fields — ไม่ crash
func TestScanCMIPolicy_NilJSONFields(t *testing.T) {
    row := &fakeRow{values: []any{..., nil, nil, nil, ...}}
    pol, err := scanCMIPolicy(row)

    testkit.NoError(t, err)
    testkit.Nil(t, pol.Motor, "Motor should be nil")
}

// 💥 Invalid JSON — error message ถูกไหม
func TestScanCMIPolicy_InvalidMotorJSON(t *testing.T) {
    row := &fakeRow{values: []any{..., []byte(`{invalid`), ...}}
    pol, err := scanCMIPolicy(row)

    testkit.Error(t, err)
    testkit.Contains(t, err.Error(), "unmarshal motor", "error message")
}
```

### 5.5 Integration Test — ต่อ DB จริง (optional)

```go
// integration_test.go — ทำงานเฉพาะถ้า set env

func TestIntegrationGetPolicyByJobID(t *testing.T) {
    dsn := os.Getenv("CMI_TEST_DSN")
    if dsn == "" {
        t.Skip("skip: CMI_TEST_DSN not set")    // ← ข้ามถ้าไม่มี DB
    }

    pool, _ := pgxpool.New(ctx, dsn)     // ← ต่อ DB จริง!
    repo := cmipg.NewCMIPolicyRepository(pool)
    svc := app.NewService(repo)          // ← wire เหมือน production

    policy, err := svc.GetPolicyByJobID(ctx, jobID)
    // ... save ผลลัพธ์เป็น JSON file ใน testdata/
}
```

**ใช้เมื่อ:** ต้องการ test SQL query จริงๆ กับ DB จริง  
**ไม่ run ใน CI ปกติ** — ต้อง set `CMI_TEST_DSN` เอง

### 5.6 testkit — เครื่องมือ Assert ที่โปรเจกต์นี้ใช้

```go
// ไม่ใช้ testify — ใช้ internal/testkit เท่านั้น

// Non-fatal (test ยัง run ต่อ)
testkit.Equal(t, got, want, "label")
testkit.NotNil(t, value, "label")
testkit.Contains(t, str, substr, "label")
testkit.NoError(t, err)
testkit.Error(t, err)

// Fatal (test หยุดทันที — ใช้ตอน setup)
testkit.MustNoError(t, err, "setup")
testkit.MustEqual(t, got, want, "precondition")
testkit.MustNotNil(t, value, "setup")
```

**ทำไมสร้างเอง?**

```
✅ ไม่มี external dependency (go.sum สะอาด)
✅ generic functions (Go 1.18+) — type-safe
✅ ข้อความ error ชัดเจน — "status code: want 200, got 404"
✅ ทุกคนในทีมเข้าใจ — เพราะเป็น Go ธรรมดา
```

---

## Appendix: Quick Reference Card

### ไฟล์ทั้งหมดของ CMI Module + หน้าที่

| ไฟล์ | Layer | หน้าที่ | Test ด้วย |
|---|---|---|---|
| `domain/cmi.go` | Domain | Pure structs (17 types) | — (ไม่ต้อง test) |
| `ports/repository.go` | Ports | Interface 2 methods | — (ไม่ test interface) |
| `app/service.go` | App | Business logic | `service_test.go` |
| `app/fakes_test.go` | App (test) | Fake สำหรับ service test | — |
| `app/service_test.go` | App (test) | 4 scenarios | — |
| `adapters/http/controller.go` | Adapter | CMIController interface | — |
| `adapters/http/handler.go` | Adapter | HTTP handler (1 route) | `handler_test.go` |
| `adapters/http/fakes_test.go` | Adapter (test) | Fake สำหรับ handler test | — |
| `adapters/http/handler_test.go` | Adapter (test) | 4 HTTP scenarios | — |
| `adapters/postgres/repository.go` | Adapter | SQL (13 fragments + scan) | `repository_test.go` |
| `adapters/postgres/repository_test.go` | Adapter (test) | Scan + SQL fragment tests | — |
| `module.go` | Wiring | Composition root | — |
| `integration_test.go` | Integration | ต่อ DB จริง (optional) | — |

### Interface → Implementor Map

```
ports.CMIPolicyRepository
  ├── postgres.CMIPolicyRepository   (production)
  ├── app.fakeCMIRepo                (service test)
  └── http.fakeRepo                  (handler test)

http.CMIController
  └── http.Handler                   (production + module.go)

pgx.Row
  ├── pgxpool.Row                    (production)
  └── postgres.fakeRow               (repository test)
```

### Test Coverage Map

```
         ┌──────────────────────────────────────────────┐
         │               Unit Tests                      │
         │                                               │
         │  service_test.go    handler_test.go            │
         │  ┌─────────────┐    ┌──────────────────┐      │
         │  │ fakeCMIRepo │    │ fakeRepo          │     │
         │  │      ↓      │    │      ↓             │     │
         │  │  Service    │    │ Service → Handler  │     │
         │  │  logic ✅   │    │ HTTP response ✅   │     │
         │  └─────────────┘    └──────────────────┘      │
         │                                               │
         │  repository_test.go                            │
         │  ┌──────────────────┐                          │
         │  │ fakeRow           │                         │
         │  │      ↓            │                         │
         │  │ scanCMIPolicy ✅  │                         │
         │  │ SQL fragments ✅  │                         │
         │  └──────────────────┘                          │
         └───────────────────┬──────────────────────────┘
                             │
                             ▼
         ┌──────────────────────────────────────────────┐
         │          Integration Test (optional)          │
         │                                               │
         │  integration_test.go                          │
         │  ┌──────────────────────────────┐             │
         │  │ pgxpool.Pool (real DB)       │             │
         │  │      ↓                       │             │
         │  │ Repo → Service → JSON file   │             │
         │  │ SQL + scan + logic ✅        │             │
         │  └──────────────────────────────┘             │
         │                                               │
         │  ⚠ ต้อง set CMI_TEST_DSN ถึงจะ run           │
         └──────────────────────────────────────────────┘
```

---

> **Takeaway:**  
> ทุก layer ของ CMI คุยกันผ่าน **interface เพียง 2 ตัว** — `CMIPolicyRepository` กับ `CMIController`  
> ระบบ test ทำงานได้โดย **สวม Fake เข้าไปแทน** implementation จริง  
> Fake ไม่ใช่ magic — มันแค่ **struct ที่ return ค่าคงที่** ตาม field ที่ set ไว้
