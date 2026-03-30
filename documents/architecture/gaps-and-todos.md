# ANC Portal BE — Gap Analysis & TODO

> **วันที่ตรวจสอบ:** 2026-03-28 (อัพเดท: 2026-03-30)  
> **สถานะปัจจุบัน:** Score 9.0/10  
> **เป้าหมาย:** Production-Ready Backend

---

## สรุปภาพรวม

```text
ทำแล้ว                                ยังขาด
──────                                ──────
ทำแล้ว                                ยังขาด
──────                                ──────
✅ Hexagonal Architecture (4 modules)  ❌ Auth/JWT Middleware
✅ CI/CD Pipeline (4 workflows)        ❌ RBAC Guard Middleware
✅ K8s + Kustomize overlays            ✅ Request Logging Middleware ← DONE
✅ OTel Tracing ทุก layer              ❌ Real JWT Token Signer
✅ Kafka + DLQ                         ✅ Circuit Breaker ← DONE
✅ Hybrid Cache (L1+L2)                ✅ Request DTO Validation ← DONE
✅ Docker multi-stage                  ❌ Global Error Handler
✅ golangci-lint 17 linters            ❌ Domain Validation Methods
✅ testkit library                     ✅ Error Code Catalog (TraceId) ← DONE
✅ 174+ test cases pass                ❌ Handler/Adapter Tests
✅ 10 technical docs                   ❌ 14+ packages 0% coverage
✅ Worker Health Probe ← DONE
✅ Discord CI/CD Notification ← DONE
✅ Bug Fixes (4 items) ← DONE
✅ Swagger ErrorResponse + TraceId ← DONE
✅ CI/CD Pipeline Explained Doc ← DONE
```

---

## 🔴 Critical — ต้องมีก่อน Production

### C1: Auth/JWT Middleware

- [ ] สร้าง `internal/modules/auth/adapters/http/middleware.go`
- [ ] Implement JWT token verification (ใช้ `golang-jwt/jwt/v5`)
- [ ] ดึง claims (userID, roles) จาก token → ใส่ `fiber.Ctx.Locals()`
- [ ] Apply middleware กับ route group `/v1/*` (ยกเว้น `/v1/auth/login`)
- [ ] Return 401 เมื่อ token ไม่ถูกต้องหรือหมดอายุ
- [ ] เขียน unit test สำหรับ middleware

**ไฟล์ที่เกี่ยวข้อง:**

- `server/server.go` — mount middleware บน protected routes
- `config/config.go` — `JWTSecretKey` มีอยู่แล้วแต่ยังไม่ถูกใช้
- `internal/modules/auth/ports/token_signer.go` — เพิ่ม `VerifyToken()` method

### C2: Real JWT Token Signer

- [ ] เพิ่ม dependency `github.com/golang-jwt/jwt/v5`
- [ ] สร้าง `internal/modules/auth/adapters/external/jwt_token_signer.go`
- [ ] Implement `SignAccessToken()` — สร้าง JWT ด้วย HS256 + claims (sub, roles, exp, iat)
- [ ] Implement `VerifyAccessToken()` — verify signature + expiry
- [ ] Config: token expiry duration (เช่น 24h) จาก config
- [ ] เขียน unit test (sign → verify roundtrip, expired token, tampered token)
- [ ] เปลี่ยน `module.go` ให้ใช้ `JWTTokenSigner` แทน `SimpleTokenSigner` เมื่อ `StageStatus != "local"`

**ไฟล์ที่เกี่ยวข้อง:**

- `internal/modules/auth/adapters/external/simple_token_signer.go` — เก็บไว้สำหรับ local dev
- `internal/modules/auth/ports/token_signer.go` — เพิ่ม interface method

### C3: RBAC Guard Middleware

- [ ] สร้าง `internal/modules/auth/adapters/http/rbac.go`
- [ ] Implement `RequireRoles(roles ...string) fiber.Handler`
- [ ] ดึง roles จาก `c.Locals("roles")` (ที่ auth middleware set ไว้)
- [ ] Return 403 Forbidden เมื่อ role ไม่ตรง
- [ ] Apply: admin-only routes, ops routes, viewer routes
- [ ] เขียน unit test

**ไฟล์ที่เกี่ยวข้อง:**

- `internal/shared/enum/role.go` — role constants มีอยู่แล้ว (admin, user, viewer)
- `server/server.go` — apply RBAC per route group

### C4: Request Body Validation ✅ DONE

- [x] สร้าง `internal/shared/validator/validator.go` — wrapper รอบ `go-playground/validator`
- [x] สร้าง `internal/shared/validator/bind.go` — `BindAndValidate(c, &dto)` helper
- [x] เพิ่ม validate tags ให้ request DTOs:
  - [x] `auth` — LoginRequest (`username:"required"`, `password:"required"`)
  - [ ] `cmi` — FindByJobID (`job_id:"required"`) — ใช้ query param ไม่ใช่ body
  - [ ] `quotation` — FindByCustomerID (`customer_id:"required"`) — ใช้ query param ไม่ใช่ body
- [x] Return 422 Unprocessable Entity พร้อมรายละเอียด field ที่ผิด
- [x] เขียน unit test (5 tests)
- [ ] Custom error messages (ภาษาไทย/อังกฤษ) — optional

**ไฟล์ที่สร้าง/แก้ไข:**

- `internal/shared/validator/validator.go` — singleton + FormatErrors
- `internal/shared/validator/bind.go` — BindAndValidate helper
- `internal/shared/validator/validator_test.go` — unit tests
- `internal/modules/auth/adapters/http/handler.go` — ใช้ BindAndValidate แทน BodyParser

---

## 🟡 High — ส่งผลต่อ Production Reliability

### H1: Request/Response Logging Middleware ✅ DONE

- [x] สร้าง `server/middleware/access_log.go`
- [x] Log ทุก request: method, path, status, latency, request_id, ip, user_agent, bytes_in, bytes_out
- [x] ใช้ zerolog (structured logging)
- [x] Skip health/ready/metrics endpoints (configurable SkipPaths)
- [ ] Sensitive data masking (Authorization header, password fields) — optional improvement
- [x] เขียน unit test (3 tests)
- [x] Wire เข้า `server/server.go` middleware chain (หลัง requestid, ก่อน compress)

### H2: Global Error Handler

- [ ] สร้าง `server/middleware/error_handler.go`
- [ ] Set `fiber.Config.ErrorHandler` ให้จัดการ error ทั้งระบบ
- [ ] Map domain errors → HTTP status codes อัตโนมัติ:
  - `ErrNotFound` → 404
  - `ErrUnauthorized` → 401
  - `ErrForbidden` → 403
  - `ErrValidation` → 422
  - `ErrConflict` → 409
  - unknown → 500 (ไม่ expose internal error)
- [ ] สร้าง `internal/shared/apperror/errors.go` — domain error types
- [ ] เขียน unit test

### H3: Circuit Breaker ✅ DONE

- [x] เพิ่ม dependency `github.com/sony/gobreaker/v2` v2.4.0
- [x] เพิ่ม option `WithCircuitBreaker(name)` ใน `pkg/httpclient/options.go`
- [x] เพิ่ม option `WithCircuitBreakerSettings(settings)` สำหรับ custom config
- [x] Config: max failures (5), timeout (10s), interval (30s), half-open requests (5)
- [x] เพิ่ม `IsCircuitOpen(err)` helper ใน `pkg/httpclient/errors.go`
- [x] Wrap `Do()` execution chain: traced → circuit breaker → retry
- [ ] เขียน unit test เพิ่มเติม (circuit breaker specific)

**ไฟล์ที่แก้ไข:**

- `pkg/httpclient/client.go` — เพิ่ม CB field + wrap Do()
- `pkg/httpclient/options.go` — เพิ่ม WithCircuitBreaker options
- `pkg/httpclient/errors.go` — เพิ่ม IsCircuitOpen helper

### H4: Worker Health Probe ✅ DONE

- [x] เพิ่ม minimal HTTP server ใน `cmd/worker/main.go` (port 20001)
- [x] Endpoint `/healthz` — return 200 ถ้า consumer กำลังทำงาน (ใช้ atomic.Bool)
- [x] เพิ่ม `IsHealthy()` method ใน `pkg/kafka/consumer.go`
- [x] อัพเดต `deployments/k8s/base/worker-deployment.yaml` — เปลี่ยน probe จาก exec เป็น httpGet
- [x] เพิ่ม readinessProbe ด้วย (httpGet /healthz port 20001)
- [x] Graceful shutdown สำหรับ health probe server

### H5: Error Code Catalog (TraceId System) ✅ DONE

- [x] สร้าง `internal/shared/dto/error_codes.go` — TraceId constants ทั้งโปรเจกต์
- [x] กำหนด trace codes ตาม module (15 codes):
  - Auth (10xxx): `auth-bind-failed`, `auth-invalid-creds`, `auth-internal-error`
  - Quotation (11xxx): `qt-id-required`, `qt-not-found`, `qt-internal-error`, `qt-customer-id-required`, `qt-list-internal-error`
  - CMI (12xxx): `cmi-job-id-required`, `cmi-job-not-found`, `cmi-internal-error`
  - ExternalDB (13xxx): `extdb-name-required`, `extdb-not-found`, `extdb-unhealthy`
  - Webhook (14xxx): `wh-invalid-signature`, `wh-process-failed`
- [x] สร้าง `dto.ErrorWithTrace()` helper + `ErrorResponse` / `ErrorResult` structs
- [x] เปลี่ยน `dto.Error()` → `dto.ErrorWithTrace()` ทุก handler (auth, quotation, cmi, externaldb, webhook)
- [x] อัพเดต Swagger annotations ทุก endpoint ให้ใช้ `dto.ErrorResponse` พร้อม trace_id
- [x] Swagger @description ใน `cmd/api/main.go` มี Error Code Catalog table
- [x] Regenerate docs ด้วย `swag init` สำเร็จ

**ไฟล์ที่สร้าง/แก้ไข:**

- `internal/shared/dto/error_codes.go` — TraceId constants
- `internal/shared/dto/response.go` — ErrorWithTrace(), ErrorResponse, ErrorResult
- `internal/modules/*/adapters/http/handler.go` — ทุก handler (5 ไฟล์)
- `cmd/api/main.go` — Swagger @description + version 1.1.0
- `docs/` — regenerated (docs.go, swagger.json, swagger.yaml)

### H6: Handler-Level Tests

- [ ] `internal/modules/auth/adapters/http/handler_test.go`
- [ ] `internal/modules/cmi/adapters/http/handler_test.go`
- [ ] `internal/modules/quotation/adapters/http/handler_test.go`
- [ ] `internal/modules/externaldb/adapters/http/handler_test.go`
- [ ] ใช้ `app.Test()` ของ Fiber สำหรับ HTTP-level testing
- [ ] Test: request parsing, validation, response format, error mapping

### H7: Domain Validation Methods

- [ ] `internal/modules/auth/domain/auth.go` — `User.Validate()`, `Session.IsExpired()`
- [ ] `internal/modules/cmi/domain/cmi.go` — `CMIPolicy.Validate()`
- [ ] `internal/modules/quotation/domain/quotation.go` — `Quotation.Validate()`
- [ ] ย้าย validation logic จาก handler/service → domain layer
- [ ] เขียน unit test สำหรับ domain validation

---

## 🟢 Medium — ความครบถ้วนของระบบ

### M1: Missing Test Coverage (14+ packages)

- [ ] `config/` — loader tests (valid config, missing fields, defaults)
- [ ] `pkg/cache/` — Client tests (get, set, delete, JSON, miss, prefix)
- [ ] `pkg/localcache/` — Client + Hybrid tests (L1→L2 fallback, write-through)
- [ ] `pkg/log/` — logger tests (level parsing, env detection)
- [ ] `internal/shared/utils/` — id, json, pointer, slice, string tests
- [ ] `internal/shared/dto/` — response builder tests
- [ ] `internal/import/` — CSV reader + importer tests
- [ ] `internal/database/postgres/manager.go` — manager lifecycle tests
- [ ] Adapter tests (ต้องใช้ test DB หรือ sqlmock):
  - [ ] `auth/adapters/postgres/user_repository_test.go`
  - [ ] `cmi/adapters/postgres/repository_test.go`
  - [ ] `quotation/adapters/postgres/repository_test.go`

### M2: Root `.env.example`

- [ ] สร้าง `.env.example` ที่ root ที่รวมทุก env var:
  - Database (host, port, user, password, dbname, sslmode)
  - Redis (host, port, password)
  - Kafka (brokers, topic, group_id)
  - OTel (enabled, exporter_url)
  - JWT (secret_key, expiry)
  - Server (port, body_limit, timeout)
  - Stage (local/staging/production)

### M3: Getting-Started Guide

- [ ] สร้าง `documents/getting-started.md`
- [ ] เนื้อหา:
  - Prerequisites (Go, Docker, Docker Compose)
  - Clone + setup
  - `run.ps1 local-up` / `make local-up`
  - `run.ps1 dev` / `make dev`
  - Run tests, lint
  - Swagger access
  - Seed data
  - Troubleshooting

### M4: ADR (Architecture Decision Records)

- [ ] สร้าง `documents/adr/` folder
- [ ] ADR template (`documents/adr/000-template.md`)
- [ ] ADR-001: ทำไมเลือก Modular Monolith (ไม่ใช่ microservices)
- [ ] ADR-002: ทำไมเลือก Hexagonal Architecture
- [ ] ADR-003: ทำไมเลือก Fiber (ไม่ใช่ Echo/Chi/Gin)
- [ ] ADR-004: ทำไมเลือก pgx (ไม่ใช่ GORM/sqlx)
- [ ] ADR-005: ทำไมสร้าง testkit เอง (ไม่ใช่ testify)

### M5: 5 Empty Placeholder Modules

- [ ] `internal/modules/document/` — implement ตาม hexagonal pattern
- [ ] `internal/modules/job/` — implement ตาม hexagonal pattern
- [ ] `internal/modules/notification/` — implement ตาม hexagonal pattern
- [ ] `internal/modules/payment/` — implement ตาม hexagonal pattern
- [ ] `internal/modules/policy/` — implement ตาม hexagonal pattern

### M6: Dockerfiles for sync/import

- [ ] อัพเดต `deployments/docker/Dockerfile` — เพิ่ม sync + import build targets
- [ ] หรือใช้ binary ที่มีอยู่ใน API image (มี sync/import binary อยู่แล้ว)

### M7: Database Manager Cleanup

- [ ] แก้ `internal/database/postgres/manager.go` — close connections ที่เปิดแล้วเมื่อ external DB connect fail
- [ ] เพิ่ม log warning เมื่อ CMI/quotation module skip registration (silent return)

---

## 🔵 Nice-to-Have — ปรับปรุงเพิ่มเติม

### N1: Retry Jitter

- [ ] เพิ่ม random jitter ±20% ให้ `pkg/retry/ExponentialBackoff`
- [ ] เพิ่ม max delay cap (เช่น 30s)

### N2: Type-Safe Enums

- [ ] เปลี่ยน `enum/role.go` จาก `string` เป็น `type Role string`
- [ ] เพิ่ม `IsValid()` method
- [ ] เปลี่ยน `enum/stage.go` เป็น `type Stage string`

### N3: Truncate Unicode Safety

- [ ] แก้ `internal/shared/utils/string.go` `Truncate()` ให้ทำงานบน runes (ไม่ใช่ bytes)
- [ ] รองรับข้อความภาษาไทยไม่ถูกตัดกลางตัวอักษร

### N4: Test Coverage Gate

- [x] CI pipeline มี coverage summary แสดงใน GitHub Step Summary
- [x] Coverage artifact upload (retention 14 days)
- [ ] เพิ่ม threshold gate ที่ fail CI ถ้า < 60% (ปัจจุบันเป็น summary เฉยๆ ยังไม่ fail)

### N5: CSRF Protection

- [ ] พิจารณาเพิ่ม CSRF middleware (ถ้ามี cookie-based auth ในอนาคต)
- [ ] ปัจจุบันใช้ Bearer token → ไม่จำเป็นเร่งด่วน

### N6: Cache Stampede Prevention

- [ ] เพิ่ม `singleflight` ใน `pkg/localcache/hybrid.go` `Fetch()`
- [ ] ป้องกัน concurrent requests ที่ cache miss พร้อมกัน

### N7: Kafka Commit Error Handling

- [ ] แก้ `pkg/kafka/consumer.go` — log error เมื่อ `CommitMessages` fail (ไม่ใช่ `_ =`)

### N8: Log Init Race Fix

- [ ] แก้ `pkg/log/logger.go` — ใช้ `sync.RWMutex` แทน `sync.Once` + package-level `Set()`
- [ ] ป้องกัน race ระหว่าง `L()` กับ `Set()`

---

## ลำดับการทำที่แนะนำ

```text
Phase 1 — Security Foundation (Critical)
├── C1: Auth/JWT Middleware
├── C2: Real JWT Token Signer
├── C3: RBAC Guard Middleware
└── C4: Request Body Validation ✅ DONE

Phase 2 — Production Reliability (High)
├── H1: Request/Response Logging ✅ DONE
├── H2: Global Error Handler
├── H5: Error Code Catalog (TraceId) ✅ DONE
└── H7: Domain Validation

Phase 3 — Operational Stability (High)
├── H3: Circuit Breaker ✅ DONE
├── H4: Worker Health Probe ✅ DONE
└── M7: Database Manager Cleanup

Phase 4 — Test Coverage (Medium)
├── H6: Handler-Level Tests
└── M1: Missing Test Coverage

Phase 5 — Documentation (Medium)
├── M2: Root .env.example
├── M3: Getting-Started Guide
└── M4: ADR Records

Phase 6 — Module Implementation (Medium)
├── M5: 5 Empty Modules
└── M6: Dockerfiles

Phase 7 — Polish (Nice-to-Have)
├── N1–N8: Various improvements
```

---

## 🛠 Bug Fixes Completed (2026-03-30)

บักที่พบและแก้ไขจากการ code review รอบที่ 1:

### BF1: Webhook Goroutine Panic Recovery ✅

- **ไฟล์:** `internal/modules/webhook/app/service.go`
- **ปัญหา:** goroutine ที่ส่ง Discord notification ไม่มี panic recovery — ถ้า panic จะทำให้ process ตาย
- **แก้ไข:** เพิ่ม `defer func() { if r := recover()... }()` ใน goroutine
- **บัคเพิ่ม:** แก้ `ctx, span :=` เป็น `_, span :=` (แก้ ineffassign lint error)

### BF2: ExternalDB Health Check HTTP Status ✅

- **ไฟล์:** `internal/modules/externaldb/adapters/http/handler.go`
- **ปัญหา:** เมื่อ DB unhealthy (connect ได้แต่ query ช้า) คืน 404 แทนที่ควรเป็น 503
- **แก้ไข:** เพิ่ม `if result.Status == enum.DBUnhealthy` → return 503 Service Unavailable

### BF3: Pagination Division-by-Zero Guard ✅

- **ไฟล์:** `internal/shared/pagination/pagination.go`
- **ปัญหา:** ถ้า `req.Limit = 0` → `math.Ceil(total / 0)` = panic
- **แก้ไข:** เพิ่ม guard `if req.Limit < 1 { req.Limit = 1 }`

### BF4: Quotation OTel Error Recording ✅

- **ไฟล์:** `internal/modules/quotation/adapters/http/handler.go`
- **ปัญหา:** `GetByID` และ `ListByCustomer` ไม่ record error ใน tracing span เมื่อเกิด error
- **แก้ไข:** เพิ่ม `span.RecordError(err)` ก่อน return 500

---

> **หมายเหตุ:** เอกสารนี้เป็น living document — อัพเดตเมื่อทำ item เสร็จ  
> เมื่อทั้งหมดเสร็จ คาดว่า score จะเพิ่มจาก **9.0/10 → 9.5/10**  
> **Progress:** 9/26 items done (C4, H1, H3, H4, H5, BF1-BF4) + Discord notification + Swagger TraceId — Score 8.5 → 9.0
