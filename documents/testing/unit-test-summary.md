# Unit Test — โครงสร้างและแนวทาง

> **Project:** ANC Portal Backend  
> **รูปแบบ:** Table-Driven Tests + Hand-Written Fakes + Custom Testkit  
> **External Test Dependencies:** 0

---

## 1. รูปแบบ (Pattern)

โปรเจกต์ใช้ **3 pattern** ร่วมกัน:

| Pattern | คำอธิบาย |
|---------|---------|
| **Table-Driven Tests** | ทุก test case อยู่ใน `[]struct{}` slice — loop รัน `t.Run()` ทีละ case |
| **Hand-Written Fakes** | เขียน struct จำลอง interface เอง (ไม่ใช้ mockgen/testify) |
| **Custom Testkit** | สร้าง assertion library เอง (`internal/testkit/`) ใช้ Go Generics — 0 external deps |

---

## 2. การวางไฟล์ (File Placement Guide)

```
internal/modules/{module}/
├── app/
│   ├── service.go              ← Production code (business logic)
│   ├── fakes_test.go           ← Fake structs สำหรับ mock interface
│   └── service_test.go         ← Unit test สำหรับ service
├── domain/
│   └── {entity}.go             ← Domain struct (ไม่มี test เพราะเป็น data struct)
├── ports/
│   └── repository.go           ← Interface definition (contract)
└── adapters/
    ├── http/handler.go         ← HTTP handler (ยังไม่มี test)
    └── postgres/repository.go  ← DB implementation (ยังไม่มี test)
```

### กฎการวางไฟล์

| กฎ | เหตุผล |
|----|--------|
| `_test.go` อยู่ข้างๆ ไฟล์ที่ test | Go convention — `go test` หาเจอเอง |
| `fakes_test.go` แยกออกจาก test | อ่านง่าย แก้ง่าย ใช้ร่วมข้าม test file ได้ |
| Fake อยู่ใน `package app` (ไม่ใช่ `app_test`) | เข้าถึง unexported fields ได้ |
| Testkit อยู่ที่ `internal/testkit/` | ใช้ร่วมทุก module — ไม่ผูกกับ module ใด |

---

## 3. ตัวอย่าง — Auth Login Test

### 3.1 Fake (จำลอง interface)

```go
// internal/modules/auth/app/fakes_test.go
package app

import (
    "context"
    "github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/domain"
)

// ─── Fake UserRepository ───
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

// ─── Fake TokenSigner ───
type fakeTokenSigner struct {
    token string
    err   error
}

func (f *fakeTokenSigner) SignAccessToken(_ context.Context, _ string, _ []string) (string, error) {
    if f.err != nil {
        return "", f.err
    }
    return f.token, nil
}
```

> **หลักคิด:** Fake แค่เก็บ return value ไว้ใน field — กำหนดตอนสร้าง struct ใน test case

### 3.2 Table-Driven Test

```go
// internal/modules/auth/app/service_test.go
package app

import (
    "context"
    "errors"
    "testing"

    "github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/domain"
    "github.com/onizukazaza/anc-portal-be-fake/internal/testkit"
    "golang.org/x/crypto/bcrypt"
)

func TestServiceLogin(t *testing.T) {
    // ── Setup ──
    bcryptHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
    testkit.MustNoError(t, err, "generate bcrypt hash")

    dbErr := errors.New("db unavailable")
    signErr := errors.New("sign failed")

    // ── Test Cases (ทุก case อยู่ใน slice) ──
    tests := []struct {
        name      string          // ชื่อ sub-test
        repo      *fakeUserRepo   // fake database
        signer    *fakeTokenSigner// fake token signer
        username  string
        password  string
        wantErr   error           // expected error (nil = success)
        wantToken string          // expected token
        wantUID   string          // expected user ID
    }{
        {
            name:      "success with plain password",
            repo:      &fakeUserRepo{user: &domain.User{ID: "u1", Username: "admin", PasswordHash: "admin123", Roles: []string{"admin"}}},
            signer:    &fakeTokenSigner{token: "token-123"},
            username:  "admin",
            password:  "admin123",
            wantToken: "token-123",
            wantUID:   "u1",
        },
        {
            name:      "success with bcrypt password",
            repo:      &fakeUserRepo{user: &domain.User{ID: "u1", Username: "admin", PasswordHash: string(bcryptHash), Roles: []string{"admin"}}},
            signer:    &fakeTokenSigner{token: "token-123"},
            username:  "admin",
            password:  "admin123",
            wantToken: "token-123",
            wantUID:   "u1",
        },
        {
            name:     "invalid credentials",
            repo:     &fakeUserRepo{user: &domain.User{ID: "u1", Username: "admin", PasswordHash: "admin123", Roles: []string{"admin"}}},
            signer:   &fakeTokenSigner{token: "token-123"},
            username: "admin",
            password: "wrong-pass",
            wantErr:  ErrInvalidCredentials,
        },
        {
            name:     "user repo error",
            repo:     &fakeUserRepo{err: dbErr},
            signer:   &fakeTokenSigner{token: "token-123"},
            username: "admin",
            password: "admin123",
            wantErr:  dbErr,
        },
        {
            name:     "token signer error",
            repo:     &fakeUserRepo{user: &domain.User{ID: "u1", Username: "admin", PasswordHash: "admin123", Roles: []string{"admin"}}},
            signer:   &fakeTokenSigner{err: signErr},
            username: "admin",
            password: "admin123",
            wantErr:  signErr,
        },
    }

    // ── Run ──
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
            testkit.Equal(t, session.UserID, tc.wantUID, "userID")
        })
    }
}
```

### 3.3 ผลลัพธ์

```
=== RUN   TestServiceLogin
=== RUN   TestServiceLogin/success_with_plain_password
=== RUN   TestServiceLogin/success_with_bcrypt_password
=== RUN   TestServiceLogin/invalid_credentials
=== RUN   TestServiceLogin/user_repo_error
=== RUN   TestServiceLogin/token_signer_error
--- PASS: TestServiceLogin (0.12s)
    --- PASS: TestServiceLogin/success_with_plain_password (0.00s)
    --- PASS: TestServiceLogin/success_with_bcrypt_password (0.06s)
    --- PASS: TestServiceLogin/invalid_credentials (0.00s)
    --- PASS: TestServiceLogin/user_repo_error (0.00s)
    --- PASS: TestServiceLogin/token_signer_error (0.00s)
PASS
```

---

## 4. อธิบายโครงสร้าง

```
                    ┌──────────────────────────┐
                    │      Test Case Data      │   ← กำหนด input + expected output
                    │  []struct{ name, repo,   │
                    │    signer, want... }      │
                    └────────────┬─────────────┘
                                 │
                    ┌────────────▼─────────────┐
                    │     for _, tc := range    │   ← วน loop ทุก case
                    │       t.Run(tc.name, ...) │
                    └────────────┬─────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              ▼                  ▼                  ▼
      ┌──────────────┐  ┌──────────────┐   ┌──────────────┐
      │  Fake Repo   │  │ Fake Signer  │   │   Service    │
      │ (return data │  │ (return token│   │  .Login()    │
      │  or error)   │  │  or error)   │   │  ← ตัวที่ test│
      └──────────────┘  └──────────────┘   └──────┬───────┘
                                                   │
                                           ┌───────▼───────┐
                                           │   Assertions  │
                                           │ testkit.Equal │
                                           │ testkit.Error │
                                           └───────────────┘
```

**Flow:**
1. กำหนด test cases เป็น data ใน `[]struct{}`
2. แต่ละ case สร้าง Fake ด้วยค่าที่ต้องการ (success/error)
3. สร้าง Service จริง แต่ inject Fake เข้าไปแทน interface
4. เรียก method ที่ต้องการ test
5. ตรวจผลด้วย `testkit.Equal()`, `testkit.ErrorIs()`, `testkit.Nil()`

---

## 5. ข้อดี

| # | ข้อดี | รายละเอียด |
|---|-------|-----------|
| 1 | **0 External Dependencies** | ไม่ใช้ testify, mockgen — ไม่มีปัญหา version conflict, build เร็ว |
| 2 | **เร็วมาก** | ทุก test < 1 วินาที — ไม่พึ่ง DB/Network/Docker |
| 3 | **Compile-time Safe** | Fake implement interface จริง — ถ้า interface เปลี่ยน, test compile ไม่ผ่านทันที |
| 4 | **อ่านง่าย** | Table-driven = เห็น case ทั้งหมดเป็น data ที่เดียว — ไม่ต้องกระโดดข้ามไฟล์ |
| 5 | **เพิ่ม case ง่าย** | แค่เพิ่ม `{}` อีก 1 ตัวใน slice — ไม่ต้องเขียน function ใหม่ |
| 6 | **Hexagonal Clean** | Test ที่ Service layer — swap Fake เข้าแทน interface → test business logic ล้วนๆ |
| 7 | **Go Generics Testkit** | `testkit.Equal[T]()` type-safe — ไม่ต้อง cast, IDE autocomplete ได้ |
| 8 | **Portable** | `testkit/` ย้ายไป project อื่นใช้ได้เลย |

---

## 6. ข้อเสีย

| # | ข้อเสีย | ผลกระทบ |
|---|--------|---------|
| 1 | **Fake ต้องเขียนเอง** | Interface ใหญ่ (10+ methods) เขียน Fake ลำบาก — แต่ปัจจุบัน interface เล็ก (1-3 methods) |
| 2 | **ไม่มี Mock ขั้นสูง** | ไม่มี call count, argument capture, call order verification แบบ mockgen |
| 3 | **Coverage ยังต่ำ (28.4%)** | Test ครอบคลุมเฉพาะ Service layer — ยังไม่มี Handler/Adapter test |
| 4 | **ไม่มี Integration Test** | ยังไม่ test กับ DB จริง — พึ่ง Fake อย่างเดียว |
| 5 | **Testkit ไม่ครบเท่า testify** | ไม่มี `assert.JSONEq`, `assert.Eventually`, `assert.InDelta` ฯลฯ |

---

## 7. สรุปผลลัพธ์ Test ทั้งโปรเจกต์

```
  CI Pipeline - ANC Portal Backend
  =================================
  [1/4] Lint  PASS (3.6s)
  [2/4] Test  PASS (6.0s)
  [3/4] Vuln  PASS (7.5s)
  [4/4] Build PASS (3.0s)
  PIPELINE PASSED (total: 20.1s)
```

| Metric | Value |
|--------|-------|
| Test Files | 25 |
| Test Functions | 174+ |
| Status | **ALL PASS** |
| Coverage | 28.4% |
| External Test Deps | 0 |
| Execution Time | ~6s |
