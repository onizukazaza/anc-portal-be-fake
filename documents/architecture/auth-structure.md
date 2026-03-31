# Authentication & Authorization Structure

> Version 1.0 — 30 Mar 2026  
> Status: **Implemented** (JWT + Auth Middleware)

---

## 1. ภาพรวมสถาปัตยกรรม

```
                     ┌────────────────────────────────────────────┐
                     │              HTTP Request                  │
                     └─────────────────┬──────────────────────────┘
                                       │
                            ┌──────────▼──────────┐
                            │   Fiber Middleware   │
                            │   Chain (server.go)  │
                            └──────────┬──────────┘
                                       │
         ┌─────────────────────────────┼─────────────────────────────┐
         │                             │                             │
    SkipPaths?                   Auth Middleware              (other middleware)
    ├── /v1/auth/login          middleware/auth.go
    ├── /v1/webhook/github       ┌────┴────┐
    ├── /v1/kafka/publish        │ Extract  │
    └── /healthz, /ready ...     │ Bearer   │
                                 └────┬────┘
                                      │
                              ┌───────▼────────┐
                              │ TokenSigner    │
                              │ .VerifyAccess  │
                              │  Token()       │
                              └───────┬────────┘
                                      │
                          ┌───────────┼───────────┐
                          │                       │
                  local (dev)             staging/production
                  SimpleTokenSigner       JWTTokenSigner
                  "dev-token:..."         HS256 + expiry
                                      │
                              ┌───────▼────────┐
                              │ Inject claims  │
                              │ c.Locals(...)  │
                              │ userID, roles  │
                              └───────┬────────┘
                                      │
                              ┌───────▼────────┐
                              │ Handler (next) │
                              └────────────────┘
```

## 2. ไฟล์ที่เกี่ยวข้อง

### Ports (Interface)

| ไฟล์ | หน้าที่ |
|------|---------|
| `internal/modules/auth/ports/token_signer.go` | Interface `TokenSigner` — Sign + Verify |
| `internal/modules/auth/ports/user_repository.go` | Interface `UserRepository` — FindByUsername |

### Adapters (Implementation)

| ไฟล์ | หน้าที่ | ใช้ใน |
|------|---------|-------|
| `adapters/external/jwt_token_signer.go` | JWT HS256 sign + verify | staging, production |
| `adapters/external/simple_token_signer.go` | Dev-only plaintext token | local |
| `adapters/postgres/user_repository.go` | Query users table | ทุก stage |
| `adapters/http/handler.go` | Login endpoint handler | ทุก stage |

### Middleware

| ไฟล์ | หน้าที่ |
|------|---------|
| `server/middleware/auth.go` | Auth middleware — extract Bearer, verify, inject claims |
| `server/middleware/auth_test.go` | Tests: skip path, missing header, invalid/valid token, wildcard |

### Config

| ไฟล์ | Field | Env Var | Default |
|------|-------|---------|---------|
| `config/config.go` | `Server.JWTSecretKey` | `SERVER_JWT_SECRET_KEY` | (required) |
| `config/config.go` | `Server.JWTExpiry` | `SERVER_JWT_EXPIRY` | `24h` |

### Module Registration

| ไฟล์ | หน้าที่ |
|------|---------|
| `internal/modules/auth/module.go` | Wire dependencies + expose `TokenSigner()` for middleware |
| `server/server.go` | Register auth middleware on `/v1` group |

---

## 3. Token Signer Selection (stage-based)

```go
// module.go — newTokenSigner()
if cfg.StageStatus == "local" {
    return external.NewSimpleTokenSigner()       // dev-token:userID:roles:ts
}
return external.NewJWTTokenSigner(secret, expiry) // HS256 JWT
```

| Stage | Token Signer | Token Format |
|-------|-------------|--------------|
| `local` | `SimpleTokenSigner` | `dev-token:{userID}:{roles}:{timestamp}` |
| `staging` | `JWTTokenSigner` | JWT HS256 (sub, roles, exp, iat, iss) |
| `production` | `JWTTokenSigner` | JWT HS256 (sub, roles, exp, iat, iss) |

---

## 4. JWT Claims Structure

```json
{
  "sub": "user-123",
  "roles": ["admin", "viewer"],
  "iat": 1711800000,
  "exp": 1711886400,
  "iss": "anc-portal"
}
```

| Claim | Type | คำอธิบาย |
|-------|------|---------|
| `sub` | string | User ID |
| `roles` | []string | User roles |
| `iat` | number | Issued at (Unix) |
| `exp` | number | Expires at (Unix) |
| `iss` | string | Issuer = `anc-portal` |

---

## 5. Middleware Flow

```
1. Request เข้า /v1/xxx
2. ตรวจ SkipPaths → skip ถ้า match (exact หรือ wildcard *)
3. อ่าน Authorization header
   - ไม่มี → 401 + trace_id: auth-token-missing
   - ไม่ใช่ "Bearer " → 401 + trace_id: auth-token-missing
   - token ว่าง → 401 + trace_id: auth-token-missing
4. เรียก TokenSigner.VerifyAccessToken(token)
   - error → 401 + trace_id: auth-token-invalid
5. Inject claims → c.Locals("userID"), c.Locals("roles")
6. c.Next() → handler
```

### Skip Paths (ไม่ต้อง authenticate)

| Path | เหตุผล |
|------|--------|
| `/v1/auth/login` | Login endpoint — ยังไม่มี token |
| `/v1/webhook/github` | GitHub webhook — ใช้ HMAC signature แทน |
| `/v1/kafka/publish` | Dev-only endpoint (local stage เท่านั้น) |

---

## 6. Error Codes (Auth Module)

| Code | Trace ID | HTTP | คำอธิบาย |
|------|----------|------|---------|
| 10001 | `auth-bind-failed` | 400 | request body ไม่ถูกต้อง |
| 10002 | `auth-invalid-creds` | 401 | username/password ไม่ถูกต้อง |
| 10003 | `auth-internal-error` | 500 | เกิดข้อผิดพลาดภายใน auth service |
| 10004 | `auth-token-missing` | 401 | ไม่มี Authorization header หรือ token ว่าง |
| 10005 | `auth-token-invalid` | 401 | token ไม่ถูกต้องหรือหมดอายุ |

---

## 7. วิธีใช้ Claims ใน Handler

```go
import mw "github.com/onizukazaza/anc-portal-be-fake/server/middleware"

func (h *Handler) GetProfile(c *fiber.Ctx) error {
    userID := c.Locals(mw.CtxUserID).(string)
    roles  := c.Locals(mw.CtxRoles).([]string)
    // ...
}
```

---

## 8. Test Coverage

| ไฟล์ Test | ทดสอบ | จำนวน |
|-----------|------|------|
| `adapters/external/token_signer_test.go` | JWT sign/verify, expired, wrong secret, invalid, SimpleToken | 6 tests |
| `app/service_test.go` | Login flow (bcrypt, plain, invalid, errors) | 5 tests |
| `server/middleware/auth_test.go` | Skip path, missing header, invalid format, empty/bad/valid token, wildcard | 7 tests |

---

## 9. Configuration Example

```bash
# .env.local (dev — uses SimpleTokenSigner, no real JWT)
SERVER_JWT_SECRET_KEY=dev_super_secret_key_123
# SERVER_JWT_EXPIRY=24h   # ไม่จำเป็น (default 24h)

# .env.prod (production — uses JWTTokenSigner)
SERVER_JWT_SECRET_KEY=<strong-random-256bit-key>
SERVER_JWT_EXPIRY=8h
```

---

## 10. Roadmap ถัดไป

| Priority | งาน | สถานะ |
|----------|-----|-------|
| 1 | Auth middleware + JWT Token Signer | ✅ Done |
| 2 | Login Rate Limit แยก (5 req/min สำหรับ /auth/login) | ⬜ Planned |
| 3 | Account Lockout (ล็อคหลัง 5 failed attempts) | ⬜ Planned |
| 4 | Refresh Token (separate endpoint + rotation) | ⬜ Planned |
| 5 | Role-based Access Control middleware | ⬜ Planned |
