# Unit Test Summary

> **Pattern:** Table-Driven Tests · Hand-Written Fakes · Custom Testkit (0 external deps)

---

## โครงสร้างไฟล์

```
internal/modules/{module}/app/
├── service.go          ← Production code
├── fakes_test.go       ← Fake structs (mock interface)
└── service_test.go     ← Unit test

internal/testkit/       ← Assertion library (Go Generics, ใช้ร่วมทุก module)
```

---

## ตัวอย่าง — Auth Login

**Fake:** จำลอง interface ด้วย struct ธรรมดา

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
```

**Test:** ทุก case อยู่ใน `[]struct{}` — วน loop `t.Run()`

```go
// service_test.go
func TestServiceLogin(t *testing.T) {
    tests := []struct {
        name      string
        repo      *fakeUserRepo
        signer    *fakeTokenSigner
        username  string
        password  string
        wantErr   error
        wantToken string
    }{
        {
            name:      "success",
            repo:      &fakeUserRepo{user: &domain.User{ID: "u1", PasswordHash: "admin123"}},
            signer:    &fakeTokenSigner{token: "token-123"},
            username:  "admin",
            password:  "admin123",
            wantToken: "token-123",
        },
        {
            name:     "invalid credentials",
            repo:     &fakeUserRepo{user: &domain.User{PasswordHash: "admin123"}},
            username: "admin",
            password: "wrong-pass",
            wantErr:  ErrInvalidCredentials,
        },
        {
            name:     "db error",
            repo:     &fakeUserRepo{err: errors.New("db unavailable")},
            wantErr:  errors.New("db unavailable"),
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            svc := NewService(tc.repo, tc.signer)
            session, err := svc.Login(ctx, tc.username, tc.password)

            if tc.wantErr != nil {
                testkit.ErrorIs(t, err, tc.wantErr)
                return
            }
            testkit.NoError(t, err)
            testkit.Equal(t, session.AccessToken, tc.wantToken)
        })
    }
}
```

**ผลลัพธ์:**

```
=== RUN   TestServiceLogin
=== RUN   TestServiceLogin/success
=== RUN   TestServiceLogin/invalid_credentials
=== RUN   TestServiceLogin/db_error
--- PASS: TestServiceLogin (0.12s)
```

---

## ข้อดี / ข้อเสีย

| ข้อดี | ข้อเสีย |
|-------|--------|
| 0 external test deps — build เร็ว ไม่มี version conflict | Fake ต้องเขียนเอง (เหมาะกับ interface เล็ก 1-3 methods) |
| Compile-time safe — interface เปลี่ยน test พังทันที | ไม่มี call count / argument capture แบบ mockgen |
| อ่านง่าย — case ทั้งหมดเป็น data อยู่ที่เดียว | Coverage ยังต่ำ (28.4%) ครอบคลุมเฉพาะ Service layer |
| เพิ่ม case ง่าย — แค่เพิ่ม `{}` ใน slice | ยังไม่มี Handler/Adapter/Integration test |
| Testkit ใช้ Go Generics — type-safe, portable | Testkit ไม่ครบเท่า testify (ไม่มี JSONEq, Eventually ฯลฯ) |

---

## สถิติ

| Metric | Value |
|--------|-------|
| Test Files | 25 |
| Test Functions | 174+ |
| Status | **ALL PASS** |
| Coverage | 28.4% |
| Execution Time | ~6s |
| External Test Deps | 0 |
