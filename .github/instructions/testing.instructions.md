---
description: "Use when writing, reviewing, or generating unit tests. Covers testkit usage, fake patterns, handler tests, repository tests, and testing rules. CRITICAL: no external test dependencies allowed."
applyTo: "**/*_test.go"
---

# Testing — สิ่งที่ต้องระวัง

## 1. ห้ามใช้ External Test Dependencies

```go
// ❌ ห้ามเด็ดขาด
import "github.com/stretchr/testify/assert"
import "github.com/golang/mock/gomock"
import "go.uber.org/mock/mockgen"

// ✅ ใช้แทน
import "github.com/onizukazaza/anc-portal-be-fake/internal/testkit"
```

## 2. testkit Assertions

### Assert (non-fatal — test ยัง run ต่อ)

```go
testkit.Equal(t, got, want, "label")
testkit.NotEqual(t, got, notWant, "label")
testkit.True(t, condition, "label")
testkit.False(t, condition, "label")
testkit.Nil(t, value, "label")
testkit.NotNil(t, value, "label")
testkit.NoError(t, err)
testkit.Error(t, err)
testkit.ErrorIs(t, err, target)
testkit.Contains(t, str, substr)
testkit.Len(t, slice, wantLen, "label")
```

### Must (fatal — test หยุดทันที, ใช้ตอน setup)

```go
testkit.MustNoError(t, err, "setup")
testkit.MustEqual(t, got, want, "precondition")
testkit.MustNil(t, value, "setup")
testkit.MustNotNil(t, value, "setup")
testkit.MustTrue(t, ok, "setup")
testkit.MustErrorIs(t, err, target, "setup")
```

## 3. Fake Pattern — Hand-Written Fakes

### ไฟล์ fakes ไว้ที่ `fakes_test.go` ใน package เดียวกับ test

```go
// app/fakes_test.go
type fakeUserRepo struct {
    user *domain.User
    err  error
}

var _ ports.UserRepository = (*fakeUserRepo)(nil)

func (f *fakeUserRepo) FindByUsername(_ context.Context, _ string) (*domain.User, error) {
    if f.err != nil {
        return nil, f.err
    }
    return f.user, nil
}
```

### สิ่งที่ต้องระวัง

- **ต้อง** ใช้ `var _ Interface = (*Fake)(nil)` ตรวจ compile-time
- **ห้าม** สร้าง fake ที่ซับซ้อน — ใช้ struct fields return ค่าตรงๆ
- **ห้าม** ใช้ `interface{}` หรือ `any` แทน interface จริง
- context parameter → `_ context.Context` (ไม่ใช้ใน test)

## 4. Table-Driven Tests

```go
func TestServiceLogin(t *testing.T) {
    tests := []struct {
        name      string          // ต้องบอกเจตนา ไม่ใช่แค่ลำดับ
        repo      *fakeUserRepo
        username  string
        wantErr   error
        wantToken string
    }{
        {
            name:      "success with valid password",
            repo:      &fakeUserRepo{user: &domain.User{...}},
            wantToken: "token-123",
        },
        {
            name:    "user not found returns error",
            repo:    &fakeUserRepo{err: ErrNotFound},
            wantErr: ErrNotFound,
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            svc := NewService(tc.repo, tc.signer)
            session, err := svc.Login(context.Background(), tc.username, tc.password)

            if tc.wantErr != nil {
                testkit.ErrorIs(t, err, tc.wantErr)
                testkit.Nil(t, session, "session")
                return  // ← สำคัญ! ป้องกัน nil dereference
            }

            testkit.NoError(t, err)
            testkit.Equal(t, session.AccessToken, tc.wantToken, "token")
        })
    }
}
```

### กฎสำคัญ

- `name` ต้องบอกเจตนา ชัดเจน (แสดงใน `go test -v`)
- ใช้ `tc` เป็นชื่อ variable (test case)
- `want*` prefix สำหรับ expected values
- error case ต้อง `return` หลัง assert error

## 5. Handler Test — Fiber + httptest

```go
func setupApp(repo *fakeRepo) *fiber.App {
    fiberApp := fiber.New()
    svc := app.NewService(repo)
    h := NewHandler(svc)
    fiberApp.Get("/path/:id", h.GetByID)
    return fiberApp
}

func doRequest(t *testing.T, app *fiber.App, id string) (*http.Response, dto.ApiResponse) {
    t.Helper()
    req := httptest.NewRequest(http.MethodGet, "/path/"+id, nil)
    resp, err := app.Test(req, -1)
    testkit.MustNoError(t, err, "fiber.Test")

    body, err := io.ReadAll(resp.Body)
    testkit.MustNoError(t, err, "read body")
    defer resp.Body.Close()

    var apiResp dto.ApiResponse
    testkit.MustNoError(t, json.Unmarshal(body, &apiResp), "unmarshal")
    return resp, apiResp
}
```

### สิ่งที่ต้องระวัง

- ใช้ `fiberApp.Test(req, -1)` ไม่ใช่ `net/http/httptest.Server`
- ต้อง `defer resp.Body.Close()`
- Assert ทั้ง status code + `apiResp.Status` + `apiResp.Message`
- Error response ต้อง assert `trace_id` ด้วย

## 6. Repository Test — fakeRow (pgx.Row)

```go
type fakeRow struct {
    values []any
    err    error
}

func (f *fakeRow) Scan(dest ...any) error {
    if f.err != nil {
        return f.err
    }
    // type-switch assign ตาม dest type
}
```

### สิ่งที่ต้องระวัง

- จำนวน values ต้องตรงกับ SELECT columns เป๊ะ
- ต้อง test: success, scan error, nil JSON fields, invalid JSON
- SQL fragment test ใช้ `strings.Contains` ไม่ผูก exact SQL

## 7. Test File Placement

```
module/
├── app/
│   ├── service.go
│   ├── service_test.go     ← service (business) tests
│   └── fakes_test.go       ← fakes สำหรับ ports
├── adapters/http/
│   ├── handler.go
│   ├── handler_test.go     ← handler (HTTP) tests
│   └── fakes_test.go       ← fakes สำหรับ handler-level
└── adapters/postgres/
    ├── repository.go
    └── repository_test.go  ← repository (scan/SQL) tests + fakeRow
```

## 8. Test Helpers

- Helper function ต้องเรียก `t.Helper()` บรรทัดแรกเสมอ
- ชื่อ helper ขึ้นต้น lowercase (unexported, ใช้ใน package เดียว)
- Helper ที่ใช้ข้าม package → ย้ายเข้า `internal/testkit/`

## 9. คำสั่งรัน Test

```bash
go test ./... -count=1                    # ทั้งหมด
go test ./internal/modules/auth/app/ -v   # เฉพาะ package
go test ./... -race -count=1              # พร้อม race detector
```
