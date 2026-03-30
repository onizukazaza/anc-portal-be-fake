# CI/CD Pipeline — อธิบายทุกส่วน

> **v1.0** — Last updated: 2026-03-30
>
> อธิบายว่าแต่ละขั้นตอนของ CI/CD ทำอะไร ตั้งแต่ Local จนถึง Production

---

## สารบัญ

- [1. ภาพรวม](#1-ภาพรวม)
- [2. Local CI Pipeline (run.ps1 ci)](#2-local-ci-pipeline-runps1-ci)
  - [Step 1 — Lint](#step-1--lint)
  - [Step 2 — Test](#step-2--test)
  - [Step 3 — Vuln](#step-3--vuln)
  - [Step 4 — Build](#step-4--build)
  - [Discord Notification](#discord-notification)
- [3. GitHub Actions CI (ci.yml)](#3-github-actions-ci-ciyml)
- [4. CD — Deploy Staging](#4-cd--deploy-staging)
- [5. CD — Deploy Production](#5-cd--deploy-production)
- [6. Release Notes (release.yml)](#6-release-notes-releaseyml)
- [7. Flow Diagram](#7-flow-diagram)
- [8. สรุป](#8-สรุป)

---

## 1. ภาพรวม

CI/CD ย่อมาจาก **Continuous Integration / Continuous Deployment**

| คำ | ความหมาย | ทำเมื่อไหร่ |
|---|---|---|
| **CI** (Continuous Integration) | ตรวจสอบโค้ดอัตโนมัติ — lint, test, vuln, build | ทุกครั้งที่ push/PR |
| **CD** (Continuous Deployment) | Deploy ไป server อัตโนมัติ | CI ผ่าน → staging อัตโนมัติ, production ต้อง approve |

```
   Developer                CI (ตรวจสอบโค้ด)                  CD (Deploy)
   ─────────               ──────────────────               ─────────────
   push code ──────────▶  Lint → Test → Vuln → Build ──▶  Staging (auto)
                                                   └──▶  Production (manual approve)
```

---

## 2. Local CI Pipeline (run.ps1 ci)

รันด้วยคำสั่ง:

```powershell
.\run.ps1 ci
```

ผลลัพธ์ที่เห็น:

```
  CI Pipeline - ANC Portal Backend
  =================================

  [1/4] Lint  PASS (7s)
  [2/4] Test  PASS (9.6s)
  [3/4] Vuln  PASS (9.9s)
  [4/4] Build PASS (7.7s)

  +---------+--------+---------+
  | Step    | Status | Time    |
  +---------+--------+---------+
  | Lint    | PASS   | 7s      |
  | Test    | PASS   | 9.6s    |
  | Vuln    | PASS   | 9.9s    |
  | Build   | PASS   | 7.7s    |
  +---------+--------+---------+

  PIPELINE PASSED (total: 34.3s)
```

### Step 1 — Lint

```
[1/4] Lint PASS (7s)
```

| รายการ | รายละเอียด |
|---|---|
| **คำสั่ง** | `golangci-lint run ./...` |
| **ทำอะไร** | ตรวจสอบคุณภาพโค้ด — หา bug, ปัญหา performance, ปัญหา security |
| **Linter ที่ใช้** | 17 ตัว (ตาม `.golangci.yml`) |
| **ถ้า FAIL** | Pipeline หยุดทันที — ไม่ต่อ step ถัดไป |

**Linter แบ่งตามประเภท:**

| ประเภท | Linter | ตรวจอะไร |
|---|---|---|
| **Bugs** | `errcheck` | ไม่ตรวจ error ที่ return กลับมา |
| | `govet` | โครงสร้างโค้ดผิดปกติ (เช่น printf format ไม่ตรง) |
| | `staticcheck` | วิเคราะห์โค้ดระดับสูง (deprecated API, dead code)  |
| | `bodyclose` | HTTP response body ไม่ปิด (memory leak) |
| | `noctx` | HTTP request ไม่ส่ง context (ทำให้ cancel ไม่ได้) |
| | `rowserrcheck` | ไม่ตรวจ `sql.Rows.Err()` |
| | `sqlclosecheck` | SQL Rows/Stmt ไม่ปิด (connection leak) |
| **Performance** | `prealloc` | slice ไม่ pre-allocate (GC ทำงานหนัก) |
| | `ineffassign` | assign ค่าให้ตัวแปรแต่ไม่ได้ใช้ |
| **Style** | `unused` | โค้ดที่ไม่ได้ใช้ |
| | `gosimple` | โค้ดที่เขียนง่ายกว่าได้ |
| | `gocritic` | คำแนะนำเพิ่มเติม (performance + style) |
| **Security** | `gosec` | ช่องโหว่ด้าน security (SQL injection, hardcoded secrets) |

**ตัวอย่าง error ที่ Lint จับ:**

```
handler.go:42:3: G101: Potential hardcoded credentials (gosec)
service.go:15:2: ineffassign: ineffective assignment to ctx (ineffassign)
```

---

### Step 2 — Test

```
[2/4] Test PASS (9.6s)
```

| รายการ | รายละเอียด |
|---|---|
| **คำสั่ง** | `go test -count 1 ./...` |
| **ทำอะไร** | รัน unit test ทั้งโปรเจกต์ |
| **-count 1** | บังคับรันใหม่ ไม่ใช้ cache |
| **./...** | ทุก package ใน project |
| **ถ้า FAIL** | Pipeline หยุดทันที |

**สิ่งที่ Test ตรวจ:**

| Module | จำนวน Test | ตรวจอะไร |
|---|---|---|
| auth/app | ~20 tests | Login logic (ถูก/ผิด/ว่าง/hash fail) |
| cmi/app | ~15 tests | ค้นหา CMI policy (found/not found/error) |
| quotation/app | ~25 tests | CRUD quotation (get/list/error) |
| externaldb/app | ~10 tests | DB health check (healthy/unhealthy/not found) |
| pagination | ~15 tests | คำนวณ page/offset (edge cases) |
| validator | ~5 tests | Validate request body |
| testkit | ~10 tests | Assert functions |
| sync | ~5 tests | Data sync framework |
| httpclient | ~15 tests | HTTP client + retry + circuit breaker |
| kafka | ~10 tests | Producer/consumer |
| otel | ~10 tests | Tracer registry |
| banner | ~5 tests | Startup banner |
| server | ~15 tests | Route registration, middleware |
| **รวม** | **174+** | |

**ตัวอย่าง test output เมื่อ fail:**

```
--- FAIL: TestLogin/invalid_password (0.00s)
    service_test.go:45: expected error "invalid credentials", got nil
FAIL    github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/app
```

---

### Step 3 — Vuln

```
[3/4] Vuln PASS (9.9s)
```

| รายการ | รายละเอียด |
|---|---|
| **คำสั่ง** | `govulncheck ./...` |
| **ทำอะไร** | ตรวจสอบ dependency ที่มีช่องโหว่ด้าน security |
| **ฐานข้อมูล** | Go Vulnerability Database (vuln.go.dev) |
| **ถ้า FAIL** | แจ้งเตือน (ไม่หยุด pipeline) |

**ตรวจอะไรบ้าง:**

```
govulncheck ./...

Scanning your code and 185 packages across 42 dependent modules
for known vulnerabilities...

No vulnerabilities found.     ← ✅ ปลอดภัย
```

**ถ้าเจอ vulnerability จะแสดง:**

```
Vulnerability #1: GO-2024-XXXX
    Description: SQL injection in database/sql
    Found in: github.com/lib/pq@v1.10.0
    Fixed in: github.com/lib/pq@v1.10.9
    More info: https://pkg.go.dev/vuln/GO-2024-XXXX
```

**วิธีแก้:** อัพเดต dependency → `go get -u <package>` → `go mod tidy`

---

### Step 4 — Build

```
[4/4] Build PASS (7.7s)
```

| รายการ | รายละเอียด |
|---|---|
| **คำสั่ง** | `go build -o ./tmp/<name>.exe ./cmd/<name>` |
| **ทำอะไร** | Build binary ทั้ง 5 ตัว — ทดสอบว่า compile ผ่าน |
| **ถ้า FAIL** | แสดง compile error |

**5 binaries ที่ build:**

| Binary | Source | หน้าที่ |
|---|---|---|
| `main.exe` | `cmd/api` | HTTP API server (Fiber) |
| `worker.exe` | `cmd/worker` | Kafka consumer (งาน background) |
| `migrate.exe` | `cmd/migrate` | Database migration |
| `seed.exe` | `cmd/seed` | Seed ข้อมูลเริ่มต้น |
| `import.exe` | `cmd/import` | CSV import tool |

**ตัวอย่าง compile error:**

```
./handler.go:25:10: undefined: dto.ErrorWithTrace
```

---

### Discord Notification

เมื่อ pipeline จบ (ไม่ว่าจะ PASS หรือ FAIL) จะส่ง notification ไป Discord:

```
┌─────────────────────────────────────┐
│ ✅ CI Pipeline Passed (Local)       │
│                                     │
│ Branch:  develop                    │
│ Commit:  a1b2c3d                    │
│ Machine: guitar@DESKTOP-XXX        │
│ Message: feat: add TraceId system   │
│                                     │
│ ✅ Lint  ✅ Test  ✅ Vuln  ✅ Build │
│ Total Time: 34.3s                   │
└─────────────────────────────────────┘
```

**ตั้งค่า:** สร้างไฟล์ `.env.local` ที่ root:

```
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/xxx/yyy
```

---

## 3. GitHub Actions CI (ci.yml)

เมื่อ push หรือ PR ไปที่ `develop` / `main` จะรัน CI อัตโนมัติบน GitHub:

```
Trigger: push/PR → develop, main

   ┌──────────────────────────────────────────────┐
   │           Parallel (พร้อมกัน)                 │
   │  ┌──────┐   ┌──────┐   ┌──────┐             │
   │  │ Lint │   │ Test │   │ Vuln │             │
   │  └──┬───┘   └──┬───┘   └──┬───┘             │
   │     └──────────┼──────────┘                   │
   │                ▼                              │
   │  ┌───────────────────────┐                    │
   │  │  Build (5 binaries)   │                    │
   │  └───────────┬───────────┘                    │
   │              ▼                                │
   │  ┌───────────────────────┐                    │
   │  │   Docker (2 images)   │                    │
   │  └───────────┬───────────┘                    │
   │              ▼                                │
   │  ┌───────────────────────┐                    │
   │  │  Scan (Trivy security)│                    │
   │  └───────────┬───────────┘                    │
   │              ▼                                │
   │  ┌───────────────────────┐                    │
   │  │   Notify (Discord)    │                    │
   │  └───────────────────────┘                    │
   └──────────────────────────────────────────────┘
```

**7 Jobs อธิบาย:**

| # | Job | ทำอะไร | ต้องรอ |
|---|---|---|---|
| 1 | **lint** | `golangci-lint` — ตรวจคุณภาพโค้ด | - |
| 2 | **test** | `go test -race -coverprofile` — รัน test + วัด coverage | - |
| 3 | **vuln** | `govulncheck` — ตรวจ vulnerability | - |
| 4 | **build** | Build 5 binaries ด้วย ldflags (commit SHA, build time) | lint + test + vuln |
| 5 | **docker** | Build Docker images + push ไป GHCR (Container Registry)  | build |
| 6 | **scan** | Trivy scan images หาช่องโหว่ CRITICAL/HIGH | docker |
| 7 | **notify** | ส่งผลลัพธ์ทั้งหมดเข้า Discord | ทุก job (always) |

**เปรียบเทียบ Local vs GitHub Actions:**

| รายการ | Local (`run.ps1 ci`) | GitHub Actions (`ci.yml`) |
|---|---|---|
| **Lint** | ✅ เหมือนกัน | ✅ เหมือนกัน |
| **Test** | `go test -count 1` | `go test -race -coverprofile` (มี race detection + coverage) |
| **Vuln** | ✅ เหมือนกัน | ✅ เหมือนกัน |
| **Build** | 5 binaries (ไม่มี ldflags) | 5 binaries + ldflags (Git SHA, Build Time) |
| **Docker** | ❌ ไม่มี | ✅ Build + Push 2 images |
| **Scan** | ❌ ไม่มี | ✅ Trivy security scan |
| **Notify** | Discord (ถ้ามี webhook URL) | Discord (always) |
| **Parallel** | Sequential (ทีละ step) | Parallel (lint/test/vuln พร้อมกัน) |

---

## 4. CD — Deploy Staging

```
Trigger: CI ผ่าน (auto) หรือ Manual dispatch

   ┌─────────────────────────────────────────┐
   │              Deploy Staging              │
   │                                         │
   │  1. ลบ migration job เก่า               │
   │  2. รัน migration job ใหม่              │
   │  3. รอ migration เสร็จ (120s timeout)   │
   │  4. Deploy API + Worker (Kustomize)     │
   │  5. รอ rollout เสร็จ (180s timeout)     │
   │  6. Smoke test: curl /healthz           │
   │  7. แจ้ง Discord                        │
   └─────────────────────────────────────────┘
```

| ขั้นตอน | ทำอะไร | Timeout |
|---|---|---|
| **Migration** | รัน SQL migration (สร้าง/แก้ table) | 120s |
| **Deploy** | อัพเดต image tag → K8s rollout (zero-downtime) | 180s |
| **Smoke Test** | port-forward → curl `/healthz` ต้องได้ 200 | - |
| **Notify** | แจ้ง Discord ว่า deploy สำเร็จหรือไม่ | - |

---

## 5. CD — Deploy Production

```
Trigger: push tag v* (เช่น v1.0.0) หรือ Manual dispatch

   ┌─────────────────────────────────────────┐
   │           Deploy Production              │
   │                                         │
   │  ⚠️  ต้อง Approve ใน GitHub ก่อน!       │
   │                                         │
   │  1. ตรวจสอบว่า image มีอยู่จริง         │
   │  2. ลบ migration job เก่า               │
   │  3. รัน migration job ใหม่              │
   │  4. รอ migration เสร็จ (180s timeout)   │
   │  5. Deploy API + Worker (Kustomize)     │
   │  6. รอ rollout เสร็จ (300s timeout)     │
   │  7. ตรวจสอบ pods running                │
   │  8. แจ้ง Discord                        │
   └─────────────────────────────────────────┘
```

**ต่างจาก Staging อย่างไร:**

| รายการ | Staging | Production |
|---|---|---|
| **Trigger** | CI ผ่าน (auto) | Push tag `v*` |
| **Approval** | ❌ ไม่ต้อง | ✅ ต้อง approve |
| **Image check** | ❌ ไม่ตรวจ | ✅ ตรวจว่ามีใน registry |
| **Migration timeout** | 120s | 180s |
| **Rollout timeout** | 180s | 300s |
| **Smoke test** | `/healthz` | ตรวจ pods running |

---

## 6. Release Notes (release.yml)

```
Trigger: push tag v* (เช่น v1.0.0)

   ┌─────────────────────────────────────────┐
   │           Create Release                 │
   │                                         │
   │  1. ดึง version จาก tag                  │
   │  2. ตรวจว่าเป็น pre-release ไหม         │
   │  3. หา tag ก่อนหน้า                      │
   │  4. นับ commits ระหว่าง 2 tags           │
   │  5. สร้าง changelog                      │
   │  6. สร้าง GitHub Release                 │
   │  7. แจ้ง Discord                         │
   └─────────────────────────────────────────┘
```

---

## 7. Flow Diagram

**ภาพรวมทั้งหมด ตั้งแต่เขียนโค้ดจนถึง Production:**

```
Developer เขียนโค้ด
       │
       ▼
  .\run.ps1 ci  ◄── Local CI (ก่อน push)
  ┌─[Lint]─[Test]─[Vuln]─[Build]─┐
  │ PASS? ──▶ git push            │
  │ FAIL? ──▶ แก้โค้ด → รันใหม่    │
  └───────────────────────────────┘
       │
       ▼
  git push develop  ──▶  GitHub Actions CI
  ┌─[Lint]─[Test]─[Vuln]─┐
  │ ──▶ [Build] ──▶ [Docker] ──▶ [Scan] ──▶ [Notify]
  └───────────────────────────────────────────────────┘
       │
       ▼ (CI ผ่าน)
  Deploy Staging (อัตโนมัติ)
  ┌─[Migration]─[Deploy]─[Smoke Test]─[Notify]─┐
  └─────────────────────────────────────────────┘
       │
       ▼ (ทดสอบใน Staging ผ่าน)
  git tag v1.0.0 && git push --tags
       │
       ▼
  Deploy Production (ต้อง Approve)
  ┌─[Image Check]─[Migration]─[Deploy]─[Verify]─[Notify]─┐
  └───────────────────────────────────────────────────────┘
       │
       ▼
  Release Notes สร้างอัตโนมัติบน GitHub
```

---

## 8. สรุป

### Local CI — เช็คก่อน push

```powershell
.\run.ps1 ci
```

| Step | คำสั่ง | ตรวจอะไร |
|---|---|---|
| **Lint** | `golangci-lint run ./...` | คุณภาพโค้ด + security |
| **Test** | `go test -count 1 ./...` | Unit tests ทั้งหมด |
| **Vuln** | `govulncheck ./...` | Dependency ที่มีช่องโหว่ |
| **Build** | `go build -o ./tmp/...` | Compile ผ่านหรือไม่ |

### GitHub Actions — เช็คอัตโนมัติหลัง push

| Job | เพิ่มจาก Local |
|---|---|
| **lint** | เหมือน Local |
| **test** | + race detection + coverage report |
| **vuln** | เหมือน Local |
| **build** | + ldflags (Git SHA, Build Time) |
| **docker** | Build + push Docker image ไป GHCR |
| **scan** | Trivy scan หาช่องโหว่ใน Docker image |
| **notify** | Discord notification (always) |

### CD Pipeline

| Stage | Trigger | Approval |
|---|---|---|
| **Staging** | CI ผ่าน (auto) | ❌ ไม่ต้อง |
| **Production** | Push tag `v*` | ✅ ต้อง approve |

---

> **v1.0** — March 2026 | ANC Portal Backend Team
