# Handler Constructor Pattern — `NewHandler()` ทำไมต้องมี?

> เอกสารนี้อธิบาย concept ของการวาง `NewHandler()` constructor  
> ตั้งแต่ **ทำไมต้องทำ**, **ทำงานยังไง**, **ไล่ data flow** ไปจนถึง **ประโยชน์จริงที่ได้**

---

## 1. ภาพรวม — Handler คือ "ประตูหน้าบ้าน"

ในสถาปัตยกรรม Hexagonal Architecture ที่โปรเจกต์นี้ใช้:

```
HTTP Request
     │
     ▼
 ┌────────────────────────────┐
 │  Handler (adapters/http/)  │ ← "ประตูหน้าบ้าน"
 │  - รับ HTTP request        │
 │  - แปลง request → Go type │
 │  - เรียก service           │
 │  - แปลง result → HTTP resp │
 └────────────┬───────────────┘
              │ เรียก method
              ▼
 ┌────────────────────────────┐
 │  Service (app/)            │ ← "สมองของระบบ"
 │  - business logic          │
 │  - ไม่รู้จัก HTTP เลย       │
 │  - inject interface        │
 └────────────┬───────────────┘
              │ ผ่าน interface
              ▼
 ┌────────────────────────────┐
 │  Repository (adapters/pg/) │ ← "มือที่หยิบข้อมูล"
 │  - SQL queries             │
 │  - return domain structs   │
 └────────────────────────────┘
```

**Handler ไม่ควรมี logic** — หน้าที่คือ "แปลภาษา" ระหว่าง HTTP กับ business logic

---

## 2. โค้ดจริง — `NewHandler()` หน้าตาเป็นยังไง

### Handler struct + constructor

```go
// internal/modules/cmi/adapters/http/handler.go

type Handler struct {
    service *app.Service    // ← ถือ reference ไปยัง Service
}

func NewHandler(service *app.Service) *Handler {
    return &Handler{service: service}    // ← inject dependency ผ่าน constructor
}
```

### ทำไมต้อง constructor? ทำไมไม่สร้างตรงๆ?

```go
// ❌ ห้ามทำแบบนี้ — ผูก dependency ตายตัว, test ไม่ได้
func (h *Handler) GetPolicy(c *fiber.Ctx) error {
    db := database.Connect()                     // ← สร้าง DB connection เอง
    repo := postgres.NewRepository(db)            // ← สร้าง repo เอง
    svc := app.NewService(repo)                   // ← สร้าง service เอง ทุก request!
    return svc.GetPolicyByJobID(c.UserContext(), id)
}

// ✅ ถูกต้อง — dependency ถูก inject มาตั้งแต่ตอนสร้าง
func NewHandler(service *app.Service) *Handler {
    return &Handler{service: service}
}

func (h *Handler) GetPolicy(c *fiber.Ctx) error {
    return h.service.GetPolicyByJobID(c.UserContext(), id)
    // ← ใช้ service ที่ inject มาเลย ไม่ต้องสร้างใหม่
}
```

---

## 3. Data Flow — จากกดปุ่มจนได้ข้อมูล

```
Client: GET /cmi/J001/request-policy-single-cmi
  │
  ▼ (1) Fiber router match → เรียก handler method
  │
  ├─ handler.go: GetPolicyByJobID(c *fiber.Ctx)
  │    ├─ c.Params("job_id")           → "J001"
  │    ├─ h.service.GetPolicyByJobID() → เรียก service
  │    │
  │    │  (2) Service ทำ business logic
  │    │
  │    ├─ service.go: GetPolicyByJobID(ctx, "J001")
  │    │    ├─ s.repo.JobExists()       → true
  │    │    └─ s.repo.FindPolicyByJobID() → *domain.CMIPolicy
  │    │         │
  │    │         │  (3) Repository query DB
  │    │         │
  │    │         └─ repository.go: SQL query → scan → return domain struct
  │    │
  │    │  (4) กลับมาที่ handler
  │    │
  │    └─ dto.Success(c, 200, policy)  → JSON response
  │
  ▼
Client: { "status": "success", "data": { ... } }
```

---

## 4. Composition Root — ใครเป็นคนสร้าง Handler?

**คำตอบ: `module.go`** — ทุก module มีไฟล์นี้เป็น "จุดประกอบ" (Composition Root)

```go
// internal/modules/cmi/module.go

func Register(router fiber.Router, deps module.Deps) {
    // === STEP 1: สร้าง Repository (adapter ล่างสุด) ===
    conn, err := deps.DB.External("meprakun")
    pool, err := database.PgxPool(conn)
    repo := cmipg.NewCMIPolicyRepository(pool)
    //                                     ↑
    //                               inject DB pool

    // === STEP 2: สร้าง Service (inject repo เข้าไป) ===
    service := app.NewService(repo)
    //                         ↑
    //                    inject interface

    // === STEP 3: สร้าง Handler (inject service เข้าไป) ===
    controller := cmihttp.NewCMIController(service)
    //                                      ↑
    //                               inject service

    // === STEP 4: ผูก route ===
    group := router.Group("/cmi")
    group.Get("/:job_id/request-policy-single-cmi",
        deps.Middleware.JWTAuth,
        controller.GetPolicyByJobID,
    )
}
```

### Dependency Chain ที่เกิดขึ้น:

```
module.go สร้างทุกอย่างตามลำดับ:

  DB Pool ──inject──→ Repository ──inject──→ Service ──inject──→ Handler
  (concrete)          (concrete)            (concrete)          (concrete)
                      ↑                     ↑
                      implements            uses
                      ports.CMIPolicyRepo    ports.CMIPolicyRepo
                      (interface)            (interface)
```

**จุดสำคัญ:** `module.go` เป็น **ที่เดียว** ที่รู้จัก concrete type ทุกตัว  
ชั้นอื่นๆ เห็นแค่ interface

---

## 5. ทำไมต้อง `NewXxxController()` return interface?

ทุก module มี 2 constructor:

```go
// Return concrete — ใช้ใน test
func NewHandler(service *app.Service) *Handler {
    return &Handler{service: service}
}

// Return interface — ใช้ใน module.go
func NewCMIController(service *app.Service) CMIController {
    return &Handler{service: service}
}
```

### CMIController interface คืออะไร?

```go
// controller.go
type CMIController interface {
    GetPolicyByJobID(ctx *fiber.Ctx) error
}
```

### ทำไมต้องมี 2 ตัว?

| Constructor | Return Type | ใช้ที่ | เหตุผล |
|---|---|---|---|
| `NewHandler()` | `*Handler` (concrete) | test | test ต้อง access struct fields |
| `NewCMIController()` | `CMIController` (interface) | `module.go` | production code เห็นแค่ contract |

**ประโยชน์:** `module.go` ไม่รู้ว่า controller ข้างในเป็น `*Handler` — ในอนาคตจะเปลี่ยน implementation ได้โดยไม่แก้ `module.go`

---

## 6. ประโยชน์ของ Pattern นี้

### 6.1 Testability — Test ได้ง่ายมาก

```go
// handler_test.go — test handler โดยไม่ต้องต่อ DB

func TestGetPolicy_Success(t *testing.T) {
    // สร้าง fake repo (control ผลลัพธ์ได้)
    repo := &fakeRepo{
        exists: true,
        policy: &domain.CMIPolicy{JobID: "J001"},
    }

    // inject fake → service → handler
    svc := app.NewService(repo)
    h := NewHandler(svc)

    // สร้าง HTTP request จำลอง
    app := fiber.New()
    app.Get("/cmi/:job_id", h.GetPolicyByJobID)
    req := httptest.NewRequest("GET", "/cmi/J001", nil)

    resp, _ := app.Test(req, -1)
    // assert status, body...
}
```

**ไม่ต้อง:**
- ❌ ต่อ PostgreSQL จริง
- ❌ Seed data ลง DB
- ❌ รอ network
- ❌ ใช้ mock library

### 6.2 Single Responsibility — แต่ละชั้นทำแค่หน้าที่ตัวเอง

```
Handler:    HTTP ↔ Go type แปลง         (ไม่มี SQL, ไม่มี business rule)
Service:    Business logic              (ไม่รู้จัก HTTP, ไม่รู้จัก SQL)
Repository: SQL ↔ Domain struct         (ไม่รู้จัก HTTP, ไม่มี business rule)
```

### 6.3 Loose Coupling — เปลี่ยนชิ้นส่วนได้โดยไม่กระทบที่อื่น

| สถานการณ์ | แก้ที่ไหน | ไม่ต้องแก้ |
|---|---|---|
| เปลี่ยน DB จาก Postgres → MySQL | `adapters/postgres/` | handler, service |
| เปลี่ยน framework จาก Fiber → Chi | `adapters/http/` | service, repo |
| เพิ่ม cache layer | `module.go` (inject cache repo) | handler |
| เปลี่ยน auth จาก JWT → OAuth | `adapters/external/` | handler, service |

### 6.4 Dependency สร้างที่เดียว — ไม่กระจาย

```
module.go = "จุดประกอบ" เดียว
  ├─ สร้าง repo    ← ที่นี่ที่เดียว
  ├─ สร้าง service ← ที่นี่ที่เดียว
  ├─ สร้าง handler ← ที่นี่ที่เดียว
  └─ ผูก routes    ← ที่นี่ที่เดียว
```

**ถ้าอยากรู้ว่า module ใช้ dependency อะไรบ้าง → ดูแค่ `module.go` ไฟล์เดียว**

---

## 7. เปรียบเทียบ Pattern ที่ต่างกัน

### Pattern A: Global variable (❌ bad)

```go
// ❌ handler เรียก global database ตรง
var db *sql.DB

func GetPolicy(c *fiber.Ctx) error {
    rows, _ := db.Query("SELECT ...")    // ← ผูกตาย, test ไม่ได้
}
```

**ปัญหา:** test ต้อง set global var, race condition, ผูกกับ DB เดียว

### Pattern B: Manual creation per request (❌ wasteful)

```go
// ❌ สร้างทุกอย่างใหม่ทุก request
func GetPolicy(c *fiber.Ctx) error {
    db := connectDB()                    // ← connection ใหม่ทุกครั้ง
    repo := NewRepo(db)
    svc := NewService(repo)
    return svc.GetPolicy(...)
}
```

**ปัญหา:** สิ้นเปลือง, connection pool ไม่ทำงาน

### Pattern C: Constructor Injection (✅ โปรเจกต์นี้ใช้)

```go
// ✅ สร้างครั้งเดียวตอน startup → ใช้ซ้ำทุก request
func NewHandler(service *app.Service) *Handler {
    return &Handler{service: service}
}

func (h *Handler) GetPolicy(c *fiber.Ctx) error {
    return h.service.GetPolicy(...)      // ← ใช้ของที่ inject มา
}
```

**ประโยชน์:** สร้างครั้งเดียว, test ง่าย, เปลี่ยน dependency ได้

---

## 8. ตัวอย่างจากทุก Module

| Module | Handler Constructor | Service Dependency | Controller Interface |
|---|---|---|---|
| **auth** | `NewHandler(svc)` | `ports.UserRepository` + `ports.TokenSigner` | `AuthController{Login}` |
| **cmi** | `NewHandler(svc)` | `ports.CMIPolicyRepository` | `CMIController{GetPolicyByJobID}` |
| **quotation** | `NewHandler(svc)` | `ports.QuotationRepository` | `QuotationController{GetByID, ListByCustomer}` |
| **externaldb** | `NewHandler(svc)` | `database.Provider` | `ExternalDBController{CheckAll, CheckByName}` |
| **webhook** | `NewHandler(svc)` | `ports.DiscordNotifier` | `WebhookController{HandleGitHubPush}` |

**Pattern เหมือนกันทุก module** → developer เข้าใจ module ใหม่ได้ทันทีเพราะโครงสร้างเดิม

---

## 9. Flow Diagram — จาก Startup ถึง Request

```
App Startup (main.go)
│
├─ config.Load()           → Config struct
├─ database.NewManager()   → DB connections
├─ server.New()            → Fiber app + middleware
│
└─ server.registerModules()
    │
    ├─ auth.Register(router, deps)
    │   └─ NewUserRepo(pool)
    │       → NewService(repo, tokenSigner)
    │           → NewAuthController(service)
    │               → router.Post("/auth/login", ctrl.Login)
    │
    ├─ cmi.Register(router, deps)
    │   └─ NewCMIPolicyRepo(pool)
    │       → NewService(repo)
    │           → NewCMIController(service)
    │               → router.Get("/cmi/:job_id/...", ctrl.GetPolicyByJobID)
    │
    └─ quotation.Register(router, deps)
        └─ NewQuotationRepo(pool)
            → NewService(repo)
                → NewQuotationController(service)
                    → router.Get("/quotations/:id", ctrl.GetByID)

───────────────────────────────────────────────
ทุกอย่างพร้อมแล้ว — รอรับ request

GET /cmi/J001/request-policy-single-cmi
│
├─ Middleware: JWTAuth → ตรวจ token
├─ Handler: GetPolicyByJobID(c)
│   └─ h.service.GetPolicyByJobID(ctx, "J001")
│       ├─ s.repo.JobExists(ctx, "J001") → true
│       └─ s.repo.FindPolicyByJobID(ctx, "J001") → *CMIPolicy
├─ dto.Success(c, 200, policy)
│
└─ Response: { "status": "success", "data": { ... } }
```

---

## 10. สรุป

| คำถาม | คำตอบ |
|---|---|
| `NewHandler()` คืออะไร | Constructor ที่ inject dependency เข้า handler |
| ทำไมต้อง inject | เพื่อให้ test ได้ง่าย + เปลี่ยน dependency ได้ |
| ใครเป็นคนเรียก `NewHandler()` | `module.go` (composition root) |
| ทำไมไม่สร้าง service ภายใน handler | เพราะจะสร้างใหม่ทุก request + test ไม่ได้ |
| `NewXxxController()` ต่างจาก `NewHandler()` ยังไง | return interface แทน concrete — ใช้ใน production code |
| Pattern นี้ชื่ออะไร | **Constructor Injection** (Dependency Injection รูปแบบหนึ่ง) |
| ใช้กับ module ไหน | **ทุก module** — pattern เดียวกัน 100% |

---

> **หลักการ:** Handler ไม่ต้องรู้ว่า Service ข้างในใช้ DB อะไร  
> Service ไม่ต้องรู้ว่า Handler ใช้ framework อะไร  
> `module.go` เป็นคนเดียวที่รู้ทุกอย่าง — แล้วสั่ง inject ให้ครบ
