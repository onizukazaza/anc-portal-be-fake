# Unit Test — Cheatsheet

> **Pattern:** Table-Driven Tests · Hand-Written Fakes · Custom Testkit (0 external deps)
>
> ดูรายละเอียดเต็ม: [Unit Test Guide (ฉบับเต็ม)](unit-test-guide.md)

---

## โครงสร้างไฟล์

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
    ├── http/handler.go         ← HTTP handler
    └── postgres/repository.go  ← DB implementation

internal/testkit/               ← Assertion library (Go Generics, ใช้ร่วมทุก module)
```

### กฎการวางไฟล์

| กฎ | เหตุผล |
|----|--------|
| `_test.go` อยู่ข้างๆ ไฟล์ที่ test | Go convention — `go test` หาเจอเอง |
| `fakes_test.go` แยกออกจาก test | อ่านง่าย แก้ง่าย ใช้ร่วมข้าม test file ได้ |
| Fake อยู่ใน `package app` (ไม่ใช่ `app_test`) | เข้าถึง unexported fields ได้ |
| Testkit อยู่ที่ `internal/testkit/` | ใช้ร่วมทุก module — ไม่ผูกกับ module ใด |

---

## 3 Patterns

| Pattern | คำอธิบาย |
|---------|---------|
| **Table-Driven Tests** | ทุก test case อยู่ใน `[]struct{}` — loop `t.Run()` ทีละ case |
| **Hand-Written Fakes** | เขียน struct จำลอง interface เอง (ไม่ใช้ mockgen/testify) |
| **Custom Testkit** | Assertion library เอง ใช้ Go Generics — 0 external deps |

---

## ตัวอย่าง — Auth Login

### Fake (จำลอง interface)

```go
// fakes_test.go
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

### Table-Driven Test

```go
// service_test.go
func TestServiceLogin(t *testing.T) {
    bcryptHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
    testkit.MustNoError(t, err, "generate bcrypt hash")

    tests := []struct {
        name      string
        repo      *fakeUserRepo
        signer    *fakeTokenSigner
        username  string
        password  string
        wantErr   error
        wantToken string
        wantUID   string
    }{
        {
            name:      "success with bcrypt password",
            repo:      &fakeUserRepo{user: &domain.User{ID: "u1", PasswordHash: string(bcryptHash), Roles: []string{"admin"}}},
            signer:    &fakeTokenSigner{token: "token-123"},
            username:  "admin",
            password:  "admin123",
            wantToken: "token-123",
            wantUID:   "u1",
        },
        {
            name:    "invalid credentials",
            repo:    &fakeUserRepo{user: &domain.User{PasswordHash: "admin123"}},
            signer:  &fakeTokenSigner{token: "token-123"},
            username: "admin",
            password: "wrong-pass",
            wantErr: ErrInvalidCredentials,
        },
        {
            name:    "db error",
            repo:    &fakeUserRepo{err: errors.New("db unavailable")},
            wantErr: errors.New("db unavailable"),
        },
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
            testkit.Equal(t, session.UserID, tc.wantUID, "userID")
        })
    }
}
```

### ผลลัพธ์

```
=== RUN   TestServiceLogin
=== RUN   TestServiceLogin/success_with_bcrypt_password
=== RUN   TestServiceLogin/invalid_credentials
=== RUN   TestServiceLogin/db_error
--- PASS: TestServiceLogin (0.12s)
```

---

## Test Flow Diagram

```
    ┌──────────────────────────┐
    │      Test Case Data      │   ← กำหนด input + expected output
    │  []struct{ name, repo,   │
    │    signer, want... }     │
    └────────────┬─────────────┘
                 │
    ┌────────────▼─────────────┐
    │     for _, tc := range   │   ← วน loop ทุก case
    │       t.Run(tc.name)     │
    └────────────┬─────────────┘
                 │
       ┌─────────┼─────────┐
       ▼         ▼         ▼
  ┌─────────┐┌─────────┐┌─────────┐
  │  Fake   ││  Fake   ││ Service │
  │  Repo   ││ Signer  ││ .Login()│
  └─────────┘└─────────┘└────┬────┘
                              │
                      ┌───────▼───────┐
                      │   Assertions  │
                      │ testkit.Equal │
                      │ testkit.Error │
                      └───────────────┘
```

---

## ข้อดี / ข้อเสีย

| ข้อดี | ข้อเสีย |
|-------|---------|
| 0 external dependencies | Fake ต้องเขียนเอง (interface ใหญ่ = ลำบาก) |
| เร็วมาก (< 1 วินาที) | ไม่มี call count, argument capture แบบ mockgen |
| Compile-time safe (interface เปลี่ยน = compile fail) | Coverage ยังต่ำ (~28%) — เฉพาะ Service layer |
| อ่านง่าย — เห็น case ทั้งหมดที่เดียว | ไม่มี integration test กับ DB จริง |
| เพิ่ม case ง่าย — แค่เพิ่ม `{}` ใน slice | Testkit ไม่ครบเท่า testify |
| Testkit ย้ายไป project อื่นใช้ได้ | |

---

## Quick Commands

```bash
# รัน test ทั้งโปรเจกต์
go test ./...

# รัน test ด้วย race detection
go test -race ./...

# รัน test + coverage
go test -coverprofile coverage.out ./...
go tool cover -func coverage.out     # ดูสรุป
go tool cover -html coverage.out     # ดู HTML report

# รัน test เฉพาะ module
go test ./internal/modules/auth/app/...

# รัน test เฉพาะ function
go test -run TestServiceLogin ./internal/modules/auth/app/
```
