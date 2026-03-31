# Unit Test Patterns — ANC Portal Backend

> **v3.0** — Last updated: March 2026
>
> คู่มือ Unit Test Patterns สำหรับทีม ครอบคลุม แนวคิด, กฎ, ตัวอย่างจริง, ข้อดี/ข้อเสีย
> และสิ่งที่ต้องระวัง เพื่อให้ทุกคนเขียนเทสในทิศทางเดียวกัน

---

## สารบัญ

1. [ภาพรวม](#1-ภาพรวม)
2. [Hexagonal Testing — ทำไมต้องแบบนี้](#2-hexagonal-testing--ทำไมต้องแบบนี้)
3. [Test Pattern ที่ใช้](#3-test-pattern-ที่ใช้)
   - 3.1 [Table-Driven Tests](#31-table-driven-tests)
   - 3.2 [Hand-Written Fakes](#32-hand-written-fakes)
   - 3.3 [Closure-Based Mocks](#33-closure-based-mocks)
   - 3.4 [Test Helpers (doRequest, testConfig)](#34-test-helpers)
   - 3.5 [Handler Testing Pattern (HTTP Layer)](#35-handler-testing-pattern-http-layer)
   - 3.6 [Repository Testing Pattern (Database Layer)](#36-repository-testing-pattern-database-layer)
4. [testkit Package — Assertion Helpers](#4-testkit-package--assertion-helpers)
   - 4.1 [Assert Functions (Non-Fatal)](#41-assert-functions-non-fatal)
   - 4.2 [Must Functions (Fatal)](#42-must-functions-fatal)
   - 4.3 [Fixture & Golden Helpers](#43-fixture--golden-helpers)
5. [กฎการเขียน Unit Test](#5-กฎการเขียน-unit-test)
6. [ข้อดี](#6-ข้อดี)
7. [ข้อเสีย & Trade-offs](#7-ข้อเสีย--trade-offs)
8. [สิ่งที่ต้องระวัง](#8-สิ่งที่ต้องระวัง)
9. [Decision Record — ทำไมไม่ใช้ testify / mockgen](#9-decision-record--ทำไมไม่ใช้-testify--mockgen)
10. [Before / After — ตัวอย่างการ Refactor](#10-before--after--ตัวอย่างการ-refactor)
11. [Flow Chart — ขั้นตอนการเขียน Test](#11-flow-chart--ขั้นตอนการเขียน-test)
12. [Quick Reference Card](#12-quick-reference-card)

---

## 1. ภาพรวม

โปรเจกต์นี้ใช้ **Go standard library `testing` เท่านั้น** ร่วมกับ package `internal/testkit`
ที่สร้างขึ้นเอง ไม่มี external test dependency ใดๆ (ไม่มี testify, gomock, mockery)

```
go.mod dependencies สำหรับ test = 0
```

### สถิติปัจจุบัน

| Metric | Value |
|--------|-------|
| Test packages | 18 |
| Test files | 25+ |
| Fakes files | 6 (`*_fakes_test.go` + adapter fakes) |
| External test deps | **0** |
| testkit functions | 17 (11 assert + 6 must) |
| Test layers | Service · Handler · Repository |

---

## 2. Hexagonal Testing — ทำไมต้องแบบนี้

โปรเจกต์ใช้ **Hexagonal Architecture (Ports & Adapters)** ดังนั้น test จะแบ่งเป็น:

```
┌──────────────────────────────────────────────────────────────┐
│                        Test Layer                            │
│                                                              │
│   Handler Test          Service Test         Repo Test       │
│  ┌──────────┐         ┌──────────┐        ┌──────────┐      │
│  │ fakeRepo │───▶ svc │  Fakes   │──▶port │ fakeRow  │      │
│  │ +Fiber   │         │(test-only)│        │ (pgx.Row)│      │
│  │ httptest │         └─────┬────┘        └─────┬────┘      │
│  └──────────┘               │                   │            │
│       │              ┌──────▼───────┐    ┌──────▼───────┐   │
│       └─────────────▶│ App Service  │    │  scan/SQL    │   │
│                      │ (Business)   │    │  (Repo logic)│   │
│                      └──────────────┘    └──────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

### หลักการ

- **Port** = Go interface ประกาศไว้ใน `ports/` directory
- **Fake** = struct ที่ implement interface นั้น สำหรับ test เท่านั้น
- **Service** = business logic ที่รับ interface ผ่าน constructor injection
- **Test** = สร้าง fake → inject เข้า service → เรียก method → assert ผลลัพธ์

ตัวอย่างจริง:

```
internal/modules/auth/
├── ports/
│   ├── user_repository.go    ← Interface: UserRepository
│   └── token_signer.go       ← Interface: TokenSigner
├── app/
│   ├── service.go             ← Business logic (รับ ports เป็น dependency)
│   ├── service_test.go        ← Unit test (ใช้ fakes)
│   └── fakes_test.go          ← Fakes ของ ports ทั้งหมด
└── domain/
    └── user.go                ← Domain model
```

---

## 3. Test Pattern ที่ใช้

### 3.1 Table-Driven Tests

**รูปแบบหลัก** ของทุก test ในโปรเจกต์ — ประกาศ test cases เป็น slice of struct
แล้ว loop ด้วย `t.Run()`:

```go
func TestServiceLogin(t *testing.T) {
    tests := []struct {
        name      string         // ชื่อ test case (ต้องบอกเจตนาชัดเจน)
        repo      *fakeUserRepo  // fake dependency
        signer    *fakeTokenSigner
        username  string         // input
        password  string
        wantErr   error          // expected output
        wantToken string
    }{
        {
            name:      "success with plain password",
            repo:      &fakeUserRepo{user: &domain.User{...}},
            signer:    &fakeTokenSigner{token: "token-123"},
            username:  "admin",
            password:  "admin123",
            wantToken: "token-123",
        },
        {
            name:    "invalid credentials",
            repo:    &fakeUserRepo{user: &domain.User{...}},
            signer:  &fakeTokenSigner{token: "token-123"},
            username: "admin",
            password: "wrong-pass",
            wantErr:  ErrInvalidCredentials,
        },
        // ... more cases
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            svc := NewService(tc.repo, tc.signer)
            session, err := svc.Login(context.Background(), tc.username, tc.password)

            if tc.wantErr != nil {
                testkit.ErrorIs(t, err, tc.wantErr)
                testkit.Nil(t, session, "session")
                return
            }

            testkit.NoError(t, err)
            testkit.Equal(t, session.AccessToken, tc.wantToken, "token")
        })
    }
}
```

**กฎ Table-Driven:**

| กฎ | เหตุผล |
|----|--------|
| `name` ต้องบอกเจตนา ไม่ใช่แค่ลำดับ | `go test -v` จะแสดงชื่อนี้ ต้องอ่านแล้วเข้าใจทันที |
| ใช้ `tc` เป็นชื่อ variable (test case) | Convention ของทีม |
| `want*` prefix สำหรับ expected values | แยก input กับ expected output ชัดเจน |
| error case ต้อง `return` หลัง assert error | ป้องกัน nil pointer dereference ใน success path |

---

### 3.2 Hand-Written Fakes

Fake คือ **struct ที่ implement port interface** สำหรับ test โดยเฉพาะ
เก็บไว้ในไฟล์ `*_fakes_test.go` (ใช้ `_test.go` suffix เพื่อไม่ให้ compile เข้า production)

**ตัวอย่าง — Port Interface:**

```go
// ports/user_repository.go
type UserRepository interface {
    FindByUsername(ctx context.Context, username string) (*domain.User, error)
}
```

**ตัวอย่าง — Fake:**

```go
// app/fakes_test.go
type fakeUserRepo struct {
    user *domain.User  // ค่าที่จะ return
    err  error         // error ที่จะ return
}

func (f *fakeUserRepo) FindByUsername(_ context.Context, _ string) (*domain.User, error) {
    if f.err != nil {
        return nil, f.err
    }
    return f.user, nil
}
```

**ข้อสังเกต:**

- Fake ใช้ **struct fields** กำหนดค่า return ไม่มี logic ซับซ้อน
- ขนาดเล็กมาก: 10-20 บรรทัดต่อ interface
- ไม่ต้อง generate code → ไม่มี generated files บวม
- `_ context.Context` — ไม่ใช้ context ใน test, ใส่ underscore

**Pattern ของ fakes ในโปรเจกต์:**

| File | Fake Struct | Interface / Target | Lines |
|------|------------|-------------------|-------|
| `auth/app/fakes_test.go` | `fakeUserRepo` | `UserRepository` | 12 |
| | `fakeTokenSigner` | `TokenSigner` | 10 |
| `quotation/app/fakes_test.go` | `fakeQuotationRepo` | `QuotationRepository` | 18 |
| `cmi/app/fakes_test.go` | `fakeCMIRepo` | `CMIPolicyRepository` | 12 |
| `cmi/adapters/http/fakes_test.go` | `fakeRepo` | `CMIPolicyRepository` | 15 |
| `cmi/adapters/postgres/repository_test.go` | `fakeRow` | `pgx.Row` (external) | 45 |
| `externaldb/app/fakes_test.go` | `fakeDBProvider` | `ExternalDBProvider` | 15 |
| `sync/sync_test.go` | `fakeSyncer` | `Syncer` | 15 |

---

### 3.3 Closure-Based Mocks

สำหรับ mock ที่ต้องการ **verify behavior** (เช่น ตรวจว่า method ถูกเรียกกี่ครั้ง
หรือ argument ที่ส่งมาถูกต้อง) ใช้ **closure fields** แทน fixed return values:

```go
// server/server_test.go
type mockKafkaProducer struct {
    publishFn func(ctx context.Context, msg kafka.Message) error
    calls     int  // นับจำนวนครั้งที่ถูกเรียก
}

func (m *mockKafkaProducer) PublishMessage(ctx context.Context, msg kafka.Message) error {
    m.calls++
    if m.publishFn != nil {
        return m.publishFn(ctx, msg)
    }
    return nil
}
```

**ใช้เมื่อ:**

- ต้อง verify ว่า argument ที่ส่งถูกต้อง
- ต้องนับจำนวนครั้งที่เรียก
- ต้อง return ค่าต่างกันตาม input
- Interface มีหลาย methods แต่บาง test ใช้ไม่กี่ methods

**ตัวอย่างการใช้ใน test:**

```go
producer := &mockKafkaProducer{
    publishFn: func(_ context.Context, msg kafka.Message) error {
        if msg.Key != "u1" {
            return errors.New("unexpected key")
        }
        return nil
    },
}
// ... ทำ test ...
testkit.Equal(t, producer.calls, 1, "publish calls")
```

---

### 3.4 Test Helpers

Helper functions ที่ซ้ำหลาย test ดึงออกมาเป็น function:

```go
// testConfig สร้าง config สำหรับ test
func testConfig(stage string) *config.Config {
    return &config.Config{
        StageStatus: stage,
        Server: config.Server{
            Port:         8080,
            AllowOrigins: []string{"*"},
            BodyLimit:    1024 * 1024,
            Timeout:      2 * time.Second,
            JWTSecretKey: "test-secret",
        },
        Swagger: config.Swagger{Enabled: false},
    }
}

// doRequest — HTTP helper สำหรับ test server endpoints
func doRequest(t *testing.T, s *Server, method, path string, body []byte) (int, map[string]any) {
    t.Helper()  // ← สำคัญมาก! ทำให้ error report ชี้ไปที่ caller ไม่ใช่ helper
    // ...
}
```

**กฎ:**

- Helper ต้องเรียก `t.Helper()` **บรรทัดแรกเสมอ**
- ชื่อ helper ขึ้นต้นด้วย lowercase (unexported) — ใช้ใน package เดียว
- Helper ที่ใช้ข้าม package → ย้ายเข้า `internal/testkit/`

---

### 3.5 Handler Testing Pattern (HTTP Layer)

สำหรับ test **HTTP handler** โดยไม่ต้อง start server จริง — ใช้ `fiber.New()` + `httptest.NewRequest`

**หลักการ:**

```
Fake Repo → Real Service → Real Handler → Fiber App → httptest
```

Handler test ครอบคลุม: route registration, param parsing, response format, status codes, trace_id

**ตัวอย่าง — Setup & Request Helpers:**

```go
// setupApp creates a Fiber app with the handler route.
func setupApp(repo *fakeRepo) *fiber.App {
    fiberApp := fiber.New()
    svc := app.NewService(repo)
    h := NewHandler(svc)
    fiberApp.Get("/cmi/:job_id/request-policy-single-cmi", h.GetPolicyByJobID)
    return fiberApp
}

// doRequest sends a GET request and parses response.
func doRequest(t *testing.T, fiberApp *fiber.App, jobID string) (*http.Response, dto.ApiResponse) {
    t.Helper()
    req := httptest.NewRequest(http.MethodGet, "/cmi/"+jobID+"/request-policy-single-cmi", nil)
    resp, err := fiberApp.Test(req, -1)
    testkit.MustNoError(t, err, "fiber.Test")

    body, err := io.ReadAll(resp.Body)
    testkit.MustNoError(t, err, "read body")
    defer resp.Body.Close()

    var apiResp dto.ApiResponse
    testkit.MustNoError(t, json.Unmarshal(body, &apiResp), "unmarshal response")
    return resp, apiResp
}
```

**ตัวอย่าง — test case:**

```go
func TestGetPolicyByJobID_JobNotFound(t *testing.T) {
    repo := &fakeRepo{exists: false}

    fiberApp := setupApp(repo)
    resp, apiResp := doRequest(t, fiberApp, "missing-job")

    testkit.Equal(t, resp.StatusCode, http.StatusNotFound, "status code")
    testkit.Equal(t, apiResp.Status, "ERROR", "response status")
    testkit.Equal(t, apiResp.Message, "job not found", "message")
    testkit.Contains(t, extractTraceID(t, apiResp), dto.TraceCMIJobNotFound, "trace_id")
}
```

**กฎ Handler Test:**

| กฎ | เหตุผล |
|----|--------|
| ใช้ `setupApp()` สร้าง Fiber app ใหม่ทุก test | แยก state ไม่ให้ test กระทบกัน |
| `doRequest()` ต้องเรียก `t.Helper()` | Error report ชี้ไปที่ caller ไม่ใช่ helper |
| Assert ทั้ง status code + body + trace_id | ครอบคลุมทั้ง transport layer และ error format |
| Fake inject ที่ระดับ repo ไม่ใช่ service | ให้ Service logic ถูก test ไปด้วย (integration ย่อย) |

**ไฟล์ที่เกี่ยวข้อง:**

```
cmi/adapters/http/
├── handler.go          ← Production code
├── handler_test.go     ← Handler tests (4 test cases)
└── fakes_test.go       ← fakeRepo → implements ports.CMIPolicyRepository
```

---

### 3.6 Repository Testing Pattern (Database Layer)

สำหรับ test **repository logic** (scan, unmarshal, SQL builder) โดยไม่ต้องเชื่อมต่อ database จริง
ใช้ **fakeRow** ที่ implement `pgx.Row` interface

**หลักการ:**

```
fakeRow (mock DB result) → scanCMIPolicy() → assert domain struct
```

**ตัวอย่าง — fakeRow:**

```go
type fakeRow struct {
    values []any
    err    error
}

func (f *fakeRow) Scan(dest ...any) error {
    if f.err != nil {
        return f.err
    }
    for i, val := range f.values {
        switch d := dest[i].(type) {
        case *string:
            *d = val.(string)
        case *bool:
            *d = val.(bool)
        case **int:
            if val == nil { *d = nil } else { v := val.(int); *d = &v }
        case *[]byte:
            switch v := val.(type) {
            case []byte: *d = v
            case nil:    *d = nil
            }
        case *time.Time:
            *d = val.(time.Time)
        // ... more types as needed
        }
    }
    return nil
}
```

**ตัวอย่าง — test scan:**

```go
func TestScanCMIPolicy_Success(t *testing.T) {
    row := buildSuccessRow(t)  // helper สร้าง fakeRow ที่มีครบทุก column

    pol, err := scanCMIPolicy(row)

    testkit.NoError(t, err)
    testkit.NotNil(t, pol, "policy")
    testkit.Equal(t, pol.JobID, "job-001", "JobID")
    testkit.NotNil(t, pol.Motor, "Motor")
    testkit.Equal(t, pol.Motor.Brand, "Toyota", "Motor.Brand")
}
```

**ตัวอย่าง — test SQL fragments:**

```go
func TestSQLFragments_NotEmpty(t *testing.T) {
    tests := []struct {
        name string
        fn   func() string
    }{
        {"sqlSelectJobFields", sqlSelectJobFields},
        {"sqlMotorInfo", sqlMotorInfo},
        {"sqlAssetInfo", sqlAssetInfo},
        // ... more fragment functions
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            sql := tc.fn()
            testkit.True(t, len(strings.TrimSpace(sql)) > 0,
                fmt.Sprintf("%s should not be empty", tc.name))
        })
    }
}
```

**กฎ Repository Test:**

| กฎ | เหตุผล |
|----|--------|
| ใช้ `fakeRow` แทน real DB | ทดสอบ scan logic โดยไม่ต้องมี connection |
| `buildSuccessRow()` helper สำหรับ happy path | ลด boilerplate + ใช้ซ้ำได้ |
| Test nil JSON fields แยก | ป้องกัน nil pointer dereference ใน unmarshal |
| Test invalid JSON แยก | ตรวจ error handling ของ unmarshal |
| SQL fragment test ใช้ `strings.Contains` | ไม่ผูก exact SQL → ทนต่อ format changes |

**ไฟล์ที่เกี่ยวข้อง:**

```
cmi/adapters/postgres/
├── repository.go       ← Production code (scan, unmarshal, SQL builders)
└── repository_test.go  ← Repo tests (fakeRow, scan tests, SQL fragment tests)
```

---

## 4. testkit Package — Assertion Helpers

> `internal/testkit/` — Generic assertion helpers สำหรับลด boilerplate

### 4.1 Assert Functions (Non-Fatal)

เมื่อ assertion fail → **test ล้มเหลวแต่ยัง run ต่อ** (ได้เห็น error ทุกตัว):

| Function | Signature | ใช้เมื่อ |
|----------|-----------|---------|
| `Equal` | `Equal[T comparable](t, got, want T, msg...)` | เทียบค่า |
| `NotEqual` | `NotEqual[T comparable](t, got, notWant T, msg...)` | ค่าต้องต่างกัน |
| `True` | `True(t, value bool, msg...)` | ต้อง true |
| `False` | `False(t, value bool, msg...)` | ต้อง false |
| `Nil` | `Nil(t, value any, msg...)` | ต้อง nil * |
| `NotNil` | `NotNil(t, value any, msg...)` | ต้องไม่ nil * |
| `NoError` | `NoError(t, err error, msg...)` | err ต้อง nil |
| `Error` | `Error(t, err error, msg...)` | err ต้องไม่ nil |
| `ErrorIs` | `ErrorIs(t, err, target error, msg...)` | `errors.Is` check |
| `Contains` | `Contains(t, s, substr string, msg...)` | string contains |
| `Len` | `Len[T any](t, slice []T, want int, msg...)` | length ของ slice |

> \* `Nil` / `NotNil` จัดการ **interface-wrapped nil pointers** ได้ถูกต้อง
> ผ่าน `reflect.ValueOf(v).IsNil()` — ไม่ต้องกังวลเรื่อง `(*T)(nil) != nil`

**ตัวอย่าง:**

```go
testkit.Equal(t, status, http.StatusOK, "status code")
testkit.NoError(t, err)
testkit.Contains(t, body, "success")
testkit.Len(t, items, 3, "items count")
```

### 4.2 Must Functions (Fatal)

เมื่อ assertion fail → **test หยุดทันที** (ใช้สำหรับ setup ที่ล้มเหลวแล้วไม่มีประโยชน์ run ต่อ):

| Function | ใช้เมื่อ |
|----------|---------|
| `MustEqual` | setup ต้องได้ค่าตามที่คาด |
| `MustNoError` | setup operation ต้องสำเร็จ |
| `MustNil` | setup result ต้อง nil |
| `MustNotNil` | setup result ต้องไม่ nil |
| `MustTrue` | setup condition ต้อง true |
| `MustErrorIs` | setup ต้อง error ตาม sentinel |

**ตัวอย่าง:**

```go
func TestServiceLogin(t *testing.T) {
    // ← Must: ถ้า bcrypt ล้มเหลว ไม่มีประโยชน์ run test ต่อ
    bcryptHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
    testkit.MustNoError(t, err, "generate bcrypt hash")

    // ... table-driven tests ...
}
```

### เมื่อไหร่ใช้ Assert vs Must?

```
┌─────────────────────────────────────────────────┐
│  ขั้นตอน Setup (เตรียมข้อมูล, สร้าง connection) │
│  → ใช้ Must (Fatal)                             │
│  เหตุผล: ล้มเหลวแล้วไม่มีประโยชน์ run ต่อ       │
├─────────────────────────────────────────────────┤
│  ขั้นตอน Assert (ตรวจผลลัพธ์)                    │
│  → ใช้ Assert (Error)                           │
│  เหตุผล: อยากเห็น error ทุกตัวในรอบเดียว         │
└─────────────────────────────────────────────────┘
```

### 4.3 Fixture & Golden Helpers

| Function | Purpose |
|----------|---------|
| `Fixture(t, parts...)` | สร้าง absolute path ไปยัง `testdata/` ของ test file |
| `LoadJSON(t, path, &dest)` | อ่าน JSON file แล้ว unmarshal |
| `Golden(t, path, got)` | Snapshot testing — เทียบ output กับ golden file |

**Golden File Testing:**

```go
func TestBannerOutput(t *testing.T) {
    output := renderBanner()
    testkit.Golden(t, "testdata/banner.golden", output)
}
```

อัปเดต golden files:

```bash
TESTKIT_UPDATE=1 go test ./pkg/banner/ -run TestBannerOutput
```

---

## 5. กฎการเขียน Unit Test

### กฎข้อ 1: ไฟล์ต้องอยู่ถูกที่

```
module/
├── app/
│   ├── service.go           ← production code
│   ├── service_test.go      ← unit test
│   └── fakes_test.go        ← fakes ทุกตัวของ module นี้
```

- `*_test.go` — Go จะไม่ compile เข้า binary
- Fakes ทุกตัวของ module รวมไว้ไฟล์เดียว `fakes_test.go`
- ห้ามเอา fake ไปไว้นอก `_test.go`

### กฎข้อ 2: One Fake Per Port Interface

```go
// ❌ ห้าม — fake ที่ implement หลาย interface
type megaFake struct { ... }

// ✅ ถูก — 1 fake : 1 interface
type fakeUserRepo struct { ... }      // implements UserRepository
type fakeTokenSigner struct { ... }   // implements TokenSigner
```

### กฎข้อ 3: Fake ใช้ Struct Fields ไม่ใช่ Hard-coded Values

```go
// ❌ ห้าม — hard-coded
func (f *fakeUserRepo) FindByUsername(_ context.Context, _ string) (*domain.User, error) {
    return &domain.User{ID: "u1", Username: "admin"}, nil
}

// ✅ ถูก — configurable ผ่าน struct fields
type fakeUserRepo struct {
    user *domain.User
    err  error
}

func (f *fakeUserRepo) FindByUsername(_ context.Context, _ string) (*domain.User, error) {
    if f.err != nil {
        return nil, f.err
    }
    return f.user, nil
}
```

### กฎข้อ 4: Test Name ต้องบอกเจตนา

```go
// ❌ ห้าม
"test case 1"
"error"
"happy path"

// ✅ ถูก
"success with plain password"
"returns ErrInvalidCredentials when password mismatch"
"user repo error propagates to caller"
```

### กฎข้อ 5: ใช้ testkit แทน if/Fatalf

```go
// ❌ ห้าม — verbose 3 บรรทัด
if status != http.StatusOK {
    t.Fatalf("status: want %d, got %d", http.StatusOK, status)
}

// ✅ ถูก — 1 บรรทัด + มี context label
testkit.Equal(t, status, http.StatusOK, "status")
```

### กฎข้อ 6: Error Case ต้อง Return

```go
if tc.wantErr != nil {
    testkit.ErrorIs(t, err, tc.wantErr)
    testkit.Nil(t, session, "session")
    return  // ← สำคัญ! ป้องกัน nil dereference ใน success assertions ด้านล่าง
}

testkit.NoError(t, err)
testkit.Equal(t, session.AccessToken, tc.wantToken, "token")
```

### กฎข้อ 7: Helper ต้องเรียก t.Helper()

```go
func doRequest(t *testing.T, s *Server, method, path string, body []byte) (int, map[string]any) {
    t.Helper()  // ← บรรทัดแรกเสมอ
    // ...
}
```

### กฎข้อ 8: ใช้ `context.Background()` ใน Test

```go
// ❌ ห้าม — nil context
session, err := svc.Login(nil, "admin", "pass")

// ✅ ถูก
session, err := svc.Login(context.Background(), "admin", "pass")
```

### กฎข้อ 9: Argument Order ของ testkit

```go
// Signature: Equal[T](t, got, want, msg...)
//                      ^^^  ^^^^
//                      ผลลัพธ์จริง  ค่าที่คาดหวัง

testkit.Equal(t, got, want, "label")
//               ↑     ↑
//               1st   2nd
```

> **ระวัง**: `got` มาก่อน `want` — ตรงข้ามกับ testify ที่ใช้ `assert.Equal(t, want, got)`

### กฎข้อ 10: Optional Message สำหรับ Assert ใน Loop

```go
// ✅ ใน table-driven test — ต้องมี context label เสมอ
testkit.Equal(t, got.Status, tc.wantStatus, "status")
testkit.Equal(t, got.Token, tc.wantToken, "token")

// ✅ format string ก็ได้
testkit.Equal(t, got.ID, tc.wantID, "user[%d] ID", i)
```

---

## 6. ข้อดี

### 6.1 Zero External Dependencies

```
go.mod (test-related) = ไม่มี
```

- ไม่มี version conflict ของ test library
- `go mod tidy` ไม่ดึง dependency เพิ่ม
- CI build เร็วขึ้น

### 6.2 Compile-Time Type Safety

```go
// Go generics ตรวจ type ตอน compile
testkit.Equal(t, 42, "hello")  // ← compile error: int vs string
```

ต่างจาก testify ที่ใช้ `interface{}` — bug จะเจอตอน runtime ไม่ใช่ compile time

### 6.3 Fakes เล็ก & อ่านง่าย

```
fakeUserRepo      = 12 บรรทัด
fakeTokenSigner   = 10 บรรทัด
fakeQuotationRepo = 18 บรรทัด
```

เทียบกับ mockgen/mockery ที่ generate 100-200+ บรรทัดต่อ interface

### 6.4 Test อ่านแล้วเข้าใจทันที

ไม่มี magic, ไม่มี DSL, ไม่มี implicit behavior
ทุก test case อ่านจากบนลงล่างได้ครบ: **setup → act → assert**

### 6.5 Refactor-Friendly

เพิ่ม method ใน interface → compiler บอกว่า fake ไหนต้อง update
ไม่ต้อง re-generate code

### 6.6 IDE & Debugger ทำงานได้เต็ม

- Go to Definition ไปที่ fake struct ได้ทันที
- Step-through debugger ทำงานปกติ (ไม่มี generated code กั้น)
- การค้นหา usages แม่นยำ

---

## 7. ข้อเสีย & Trade-offs

### 7.1 Fakes ต้องเขียนเอง

ทุกครั้งที่เพิ่ม interface ใหม่ → ต้องเขียน fake เอง
เทียบกับ mockgen ที่ `go generate` แล้วได้ทันที

> **Mitigation**: Fakes ไม่กี่บรรทัด (10-20) และเขียนครั้งเดียวต่อ interface
> Trade-off คุ้มค่าเพราะไม่ต้อง maintain generated files

### 7.2 ไม่มี Argument Capture Built-in

testify/mock มี `.CalledWith()` สำเร็จรูป — ของ fake ต้องเพิ่ม field เอง:

```go
type fakeEmailSender struct {
    lastTo   string   // ← เพิ่มเอง เพื่อ capture argument
    lastBody string
    err      error
}
```

### 7.3 Deep Comparison ไม่ได้

`testkit.Equal` ใช้ `comparable` constraint → **ใช้กับ struct ที่มี slice/map ไม่ได้**

```go
// ❌ compile error — []string ไม่ใช่ comparable
testkit.Equal(t, got.Roles, want.Roles)

// ✅ เทียบทีละตัว หรือใช้ reflect.DeepEqual
if !reflect.DeepEqual(got.Roles, want.Roles) {
    t.Fatalf("roles mismatch: %v vs %v", got.Roles, want.Roles)
}
```

### 7.4 testkit — Argument Order ต่างจาก testify

```go
// testify:  assert.Equal(t, expected, actual)      // want ก่อน
// testkit:  testkit.Equal(t, actual, expected)      // got ก่อน
```

ทีมต้องตกลงกัน → **ในโปรเจกต์นี้ใช้ `got, want` (actual ก่อน)**

---

## 8. สิ่งที่ต้องระวัง

### 8.1 Interface-Wrapped Nil

```go
var p *User = nil
var i interface{} = p
fmt.Println(i == nil) // false !!
```

`testkit.Nil` / `testkit.MustNil` จัดการเคสนี้ได้แล้ว (ใช้ `reflect`)
**แต่ถ้าใช้ `== nil` ตรงๆ จะผิด** → ใช้ testkit เท่านั้น

### 8.2 Race Condition ใน Parallel Tests

ถ้าใช้ `t.Parallel()` → fake struct ที่ share ข้าม goroutine ต้องระวัง:

```go
// ❌ อันตราย — tc ถูก capture by reference
for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        t.Parallel()
        svc := NewService(tc.repo, tc.signer)  // tc อาจเปลี่ยนระหว่าง run
    })
}

// ✅ ปลอดภัย — Go 1.22+ loop variable scoping แก้ปัญหานี้แล้ว
// แต่ถ้า fake มี mutable state (เช่น calls counter) ต้องใช้ atomic
```

### 8.3 อย่าลืม `return` หลัง Error Assert

```go
if tc.wantErr != nil {
    testkit.ErrorIs(t, err, tc.wantErr)
    // ← ถ้าลืม return → success assertions ด้านล่างจะ panic (nil pointer)
    return
}
testkit.NoError(t, err)
testkit.Equal(t, session.AccessToken, tc.wantToken, "token")
```

### 8.4 Closure-Based Mock กับ Nil Check

```go
type mockKafkaProducer struct {
    publishFn func(ctx context.Context, msg kafka.Message) error
}

func (m *mockKafkaProducer) PublishMessage(ctx context.Context, msg kafka.Message) error {
    if m.publishFn != nil {  // ← ต้องเช็ค nil เสมอ
        return m.publishFn(ctx, msg)
    }
    return nil
}
```

ถ้าลืม nil check → test ที่ไม่ได้ set `publishFn` จะ panic

### 8.5 Golden File Drift

Golden files ไม่อัปเดตอัตโนมัติ ถ้า code เปลี่ยน output → test fail
ต้อง run `TESTKIT_UPDATE=1 go test ...` เพื่ออัปเดต

> **ระวัง**: Review golden file diff ทุกครั้ง ไม่ใช่แค่ update แล้ว commit

---

## 9. Decision Record — ทำไมไม่ใช้ testify / mockgen

### ทำไมไม่ใช้ testify?

| Criteria | testify | testkit (ของเรา) |
|----------|---------|-------------------|
| Dependencies | +67 transitive deps | 0 |
| Type safety | `interface{}` (runtime) | Go generics (compile-time) |
| Argument order | `Equal(t, expected, actual)` | `Equal(t, got, want)` |
| Learning curve | ต้องจำ API 50+ functions | 17 functions |
| File size | 0 (import only) | ~300 บรรทัดรวม |
| Debug experience | ดี | ดี (เหมือนกัน) |

### ทำไมไม่ใช้ mockgen / mockery?

| Criteria | mockgen | Hand-written Fakes |
|----------|---------|-------------------|
| Generated file size | 100-200+ บรรทัด/interface | 10-20 บรรทัด/interface |
| Maintenance | ต้อง `go generate` ทุกครั้ง | เขียนครั้งเดียว |
| Readability | Generated code อ่านยาก | อ่านง่าย |
| IDE navigation | ต้อง jump ผ่าน generated code | Go to Definition ตรงๆ |
| Argument matching | Built-in matchers | เขียน logic ใน closure |

**สรุป**: โปรเจกต์มี ~6 port interfaces ขนาด 1-3 methods → hand-written fakes คุ้มกว่า
ถ้าอนาคตมี interface 10+ methods → อาจพิจารณา mockgen เฉพาะ interface นั้น

---

## 10. Before / After — ตัวอย่างการ Refactor

### Auth Service Test — Assertion Block

**Before (14 บรรทัด):**

```go
if tc.wantErr != nil {
    if !errors.Is(err, tc.wantErr) {
        t.Fatalf("error: want %v, got %v", tc.wantErr, err)
    }
    if session != nil {
        t.Fatalf("session: want nil, got %+v", session)
    }
    return
}

if err != nil {
    t.Fatalf("unexpected error: %v", err)
}
if session.AccessToken != tc.wantToken {
    t.Fatalf("token: want %s, got %s", tc.wantToken, session.AccessToken)
}
if session.UserID != tc.wantUID {
    t.Fatalf("userID: want %s, got %s", tc.wantUID, session.UserID)
}
```

**After (8 บรรทัด = ลด 43%):**

```go
if tc.wantErr != nil {
    testkit.ErrorIs(t, err, tc.wantErr)
    testkit.Nil(t, session, "session")
    return
}

testkit.NoError(t, err)
testkit.Equal(t, session.AccessToken, tc.wantToken, "token")
testkit.Equal(t, session.UserID, tc.wantUID, "userID")
```

### Server Test — doRequest Helper

**Before:**

```go
res, err := s.app.Test(req, -1)
if err != nil {
    t.Fatalf("request failed: %v", err)
}
defer res.Body.Close()

data, err := io.ReadAll(res.Body)
if err != nil {
    t.Fatalf("read response failed: %v", err)
}
```

**After:**

```go
res, err := s.app.Test(req, -1)
testkit.MustNoError(t, err, "request")
defer res.Body.Close()

data, err := io.ReadAll(res.Body)
testkit.MustNoError(t, err, "read response")
```

---

## 11. Flow Chart — ขั้นตอนการเขียน Test

```
                    ┌──────────────────┐
                    │  เขียน/แก้ไข      │
                    │  Business Logic   │
                    └────────┬─────────┘
                             │
                    ┌────────▼─────────┐
                    │ เลือก Layer ที่    │
                    │ จะ test           │
                    └────────┬─────────┘
                    ╱        │         ╲
                   ╱         │          ╲
         ┌────────▼──┐ ┌────▼─────┐ ┌───▼──────────┐
         │  Handler   │ │ Service  │ │  Repository  │
         │  (HTTP)    │ │ (App)    │ │  (Postgres)  │
         └────┬───────┘ └────┬─────┘ └───┬──────────┘
              │              │            │
     ┌────────▼───────┐     │    ┌───────▼────────┐
     │ สร้าง fakeRepo │     │    │ สร้าง fakeRow  │
     │ + setupApp()   │     │    │ + buildRow()   │
     │ + doRequest()  │     │    │ helper         │
     └────────┬───────┘     │    └───────┬────────┘
              │              │            │
              │    ┌─────────▼──────┐     │
              │    │ มี Port ใหม่?  │     │
              │    └────┬──────┬───┘     │
              │    Yes ╱       ╲ No      │
              │       ╱         ╲        │
              │ ┌────▼──────┐    ╲       │
              │ │ สร้าง Fake │    ╲      │
              │ │ fakes_test │     ╲     │
              │ └────┬──────┘      │     │
              │      │             │     │
              └──────┼─────────────┼─────┘
                     │             │
              ┌──────▼─────────────▼───┐
              │  เขียน Test Cases       │
              │  - Table-Driven (svc)  │
              │  - Per-function (http) │
              │  - Scan/SQL (repo)     │
              └──────┬─────────────────┘
                     │
              ┌──────▼──────────────┐
              │ ต้องการ Verify       │
              │ Arguments/Calls?    │
              └──────┬──────────────┘
                 Yes ╱ ╲ No
                    ╱   ╲
       ┌───────────▼┐  ┌▼──────────┐
       │ ใช้ Closure │  │ ใช้ Struct │
       │ Mock        │  │ Field     │
       │ Pattern     │  │ Fake      │
       └──────┬──────┘  └──┬───────┘
              │            │
              └─────┬──────┘
                    │
              ┌─────▼────────────────┐
              │  Assert ด้วย testkit  │
              │  - Setup → Must*     │
              │  - Assert → Equal/   │
              │    NoError/ErrorIs   │
              └─────┬────────────────┘
                    │
              ┌─────▼────────────────┐
              │  go test -v -count=1 │
              │  ./path/to/package/  │
              └──────────────────────┘
```

---

## 12. Quick Reference Card

### testkit Assertion Cheat Sheet

```go
// ── ค่าเท่ากัน ──
testkit.Equal(t, got, want, "label")
testkit.NotEqual(t, got, notWant, "label")

// ── Boolean ──
testkit.True(t, condition, "label")
testkit.False(t, condition, "label")

// ── Nil ──
testkit.Nil(t, value, "label")
testkit.NotNil(t, value, "label")

// ── Error ──
testkit.NoError(t, err)
testkit.Error(t, err)
testkit.ErrorIs(t, err, ErrNotFound)

// ── String / Slice ──
testkit.Contains(t, "hello world", "world")
testkit.Len(t, items, 3, "items")

// ── Fatal (setup) ──
testkit.MustNoError(t, err, "setup")
testkit.MustEqual(t, got, want, "setup")
testkit.MustNil(t, value, "setup")
testkit.MustNotNil(t, value, "setup")
testkit.MustTrue(t, ok, "setup")
testkit.MustErrorIs(t, err, ErrNotFound, "setup")

// ── Fixture ──
path := testkit.Fixture(t, "users.json")
testkit.LoadJSON(t, path, &dest)
testkit.Golden(t, "testdata/output.golden", actualOutput)
```

### Fake Pattern Template

```go
// fakes_test.go — Copy & modify สำหรับ interface ใหม่

type fake<Name> struct {
    <field1> <ReturnType>
    err      error
}

func (f *fake<Name>) <Method>(_ context.Context, _ <ArgType>) (<ReturnType>, error) {
    if f.err != nil {
        return <zero>, f.err
    }
    return f.<field1>, nil
}
```

### คำสั่ง Test ที่ใช้บ่อย

```bash
# Run ทุก test
go test ./... -count=1

# Run test เฉพาะ package
go test ./internal/modules/auth/app/ -v -count=1

# Run test เฉพาะ function
go test ./internal/modules/auth/app/ -run TestServiceLogin -v

# Run with race detector
go test ./... -race -count=1

# Update golden files
TESTKIT_UPDATE=1 go test ./pkg/banner/ -run TestBannerOutput

# Coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

> v3.0 — March 2026 | ANC Portal Backend Team
