# 🏢 ANC Portal Backend — FAKE

> **สำหรับแนวคิดและการศึกษา** 📚
>
> `Go 1.25` · `Fiber v2` · `PostgreSQL` · `Redis` · `Kafka` · `OpenTelemetry`

---

## 💡 Concept — แนวคิดของโปรเจกต์นี้

โปรเจกต์นี้เป็น **Backend API สำหรับระบบประกันภัย** ที่ออกแบบเพื่อศึกษาและทดลองแนวคิดต่าง ๆ
โดยใช้สถาปัตยกรรม **Modular Monolith + Hexagonal Architecture**

```
                        ┌─────────────────────────────────┐
                        │         ANC Portal Backend       │
                        │      (Modular Monolith + Hex)    │
                        └──────────────┬──────────────────┘
                                       │
               ┌───────────────────────┼───────────────────────┐
               │                       │                       │
        ┌──────▼──────┐        ┌───────▼──────┐       ┌───────▼──────┐
        │   Modules   │        │   Packages   │       │    Infra     │
        │             │        │              │       │              │
        │  auth       │        │  otel        │       │  Docker      │
        │  cmi        │        │  kafka       │       │  Kubernetes  │
        │  quotation  │        │  cache       │       │  GitHub CI   │
        │  document   │        │  httpclient  │       │  Grafana     │
        │  policy     │        │  retry       │       │  Dependabot  │
        │  payment    │        │  log         │       │              │
        │  job        │        │  buildinfo   │       │              │
        │  notification│       │              │       │              │
        └──────┬──────┘        └──────┬───────┘       └──────────────┘
               │                      │
               ▼                      ▼
        ┌─────────────────────────────────────┐
        │           Database Layer            │
        │   PostgreSQL (main) + External DBs  │
        │     Multi-Driver: postgres / mysql  │
        └─────────────────────────────────────┘
```

**แนวคิดหลัก:**

- **Modular Monolith** — แยก module ชัดเจน (auth, cmi, quotation ฯลฯ) แต่ deploy เป็น binary เดียว ลดความซับซ้อนของ infra ในขณะที่ code พร้อมแตกเป็น microservice ได้เมื่อถึงเวลา
- **Hexagonal (Ports & Adapters)** — business logic ไม่ผูกกับ framework ใด ๆ เปลี่ยน DB, HTTP framework, หรือ message broker ได้โดยไม่แก้ logic
- **Multi-Driver Database** — Main DB เป็น PostgreSQL แต่รองรับ External DB หลายตัว (Postgres/MySQL) ผ่าน interface เดียวกัน
- **Hybrid Cache (L1 → L2)** — In-memory cache (Otter) เป็น L1 ให้เร็ว, Redis เป็น L2 ให้แชร์ข้ามทุก instance
- **Event-Driven** — API ตอบ client ทันที งานหนักส่งผ่าน Kafka ไปทำใน Worker
- **Observability-First** — ทุก layer มี tracing (OTel) → ดูผ่าน Grafana ได้ตั้งแต่ HTTP ถึง DB query
- **Automated Quality** — CI 7 stages (Lint → Test → Vuln → Build → Docker → Scan → Notify) + Dependabot ดูแล dependency อัตโนมัติ

---

## 📋 สารบัญ

- [� Concept — แนวคิดของโปรเจกต์นี้](#-concept--แนวคิดของโปรเจกต์นี้)
- [�🚀 Quick Start](#-quick-start)
- [⚡ คำสั่งที่ใช้บ่อย](#-คำสั่งที่ใช้บ่อย)
- [🔄 CI/CD Pipeline](#-cicd-pipeline)
- [🗄️ Database Architecture](#️-database-architecture)
- [📖 Swagger — API Documentation](#-swagger--api-documentation)
- [📊 Grafana — Observability Dashboard](#-grafana--observability-dashboard)
- [🤖 Dependabot — Auto Dependency Updates](#-dependabot--auto-dependency-updates)
- [🧪 Unit Test — Testing Strategy](#-unit-test--testing-strategy)
- [📈 Coverage & Test Logging](#-coverage--test-logging)
- [🏗️ โครงสร้างโปรเจกต์](#️-โครงสร้างโปรเจกต์)
- [🎯 Entry Points](#-entry-points)
- [📚 เอกสาร](#-เอกสาร)
- [🧰 Tech Stack](#-tech-stack)

---

## 🚀 Quick Start

**ขั้นตอนที่ 1** — เริ่ม Infrastructure

```powershell
.\run.ps1 local-up      # PostgreSQL + Redis + Kafka
.\run.ps1 otel-up       # Grafana + Tempo + Prometheus + OTel Collector (optional)
```

**ขั้นตอนที่ 2** — Setup Database

```powershell
.\run.ps1 migrate        # สร้าง tables
.\run.ps1 seed           # insert ข้อมูลเริ่มต้น (users, roles)
```

**ขั้นตอนที่ 3** — รัน API Server

```powershell
.\run.ps1 dev            # hot-reload ด้วย Air
```

เปิด Swagger UI → http://localhost:20000/swagger/index.html

---

## ⚡ คำสั่งที่ใช้บ่อย

ใช้ `.\run.ps1 <command>` (Windows) หรือ `make <command>` (Linux/macOS)

| | คำสั่ง | คำอธิบาย |
|---|---|---|
| 🔧 | `dev` | รัน API server ด้วย Air (hot-reload) |
| 🔧 | `build` | Build API binary พร้อม git commit info |
| 🧪 | `test` | รัน tests ทั้งหมด |
| 📖 | `swagger` | Generate Swagger docs จาก code annotations |
| 📦 | `tidy` | `go mod tidy` |
| 🧹 | `clean` | ลบ build artifacts |
| 🗄️ | `migrate` | รัน database migrations |
| 🌱 | `seed` | Seed ข้อมูลเริ่มต้น |
| 📥 | `import` | Import CSV data (ต้องระบุ ENV, PATH, TYPE) |
| 👷 | `worker` | รัน Kafka consumer |
| 🐳 | `local-up` / `local-down` | เปิด/ปิด PostgreSQL + Redis + Kafka |
| 📊 | `otel-up` / `otel-down` | เปิด/ปิด Observability stack |
| 🏗️ | `docker-build` | Build Docker image (API + Worker) |
| ✅ | `ci` | รัน Local CI Pipeline (Lint → Test → Vuln → Build) |

---

## 🔄 CI/CD Pipeline

ระบบ CI/CD ทำงานอัตโนมัติผ่าน **GitHub Actions** — ทุกครั้งที่ push หรือเปิด PR

```
  ┌─────────────────────────────────────────────────────────────────────────┐
  │                        CI Pipeline (GitHub Actions)                     │
  │                                                                         │
  │   push / PR                                                             │
  │       │                                                                 │
  │       ▼                                                                 │
  │   ┌────────┐   ┌────────┐   ┌────────┐                                 │
  │   │  Lint  │   │  Test  │   │  Vuln  │    ← รันพร้อมกัน (parallel)     │
  │   │ go-lint│   │go test │   │ govulncheck  │                            │
  │   └───┬────┘   └───┬────┘   └───┬────┘                                 │
  │       │            │            │                                       │
  │       └────────────┼────────────┘                                       │
  │                    ▼                                                    │
  │              ┌──────────┐                                               │
  │              │  Build   │  ← compile binary + verify                    │
  │              └────┬─────┘                                               │
  │                   ▼                                                     │
  │              ┌──────────┐                                               │
  │              │  Docker  │  ← multi-stage build + push GHCR              │
  │              └────┬─────┘                                               │
  │                   ▼                                                     │
  │              ┌──────────┐                                               │
  │              │  Scan    │  ← Trivy vulnerability scan                   │
  │              └────┬─────┘                                               │
  │                   ▼                                                     │
  │              ┌──────────┐                                               │
  │              │  Notify  │  ← Discord notification (success/failure)     │
  │              └──────────┘                                               │
  └─────────────────────────────────────────────────────────────────────────┘
```

### Workflow ทั้งชุด

```
  ┌──────────┐     ┌──────────┐     ┌───────────────┐     ┌──────────────────┐
  │ Developer│     │  GitHub  │     │  CI Pipeline  │     │  Environments    │
  └────┬─────┘     └────┬─────┘     └──────┬────────┘     └────────┬─────────┘
       │                │                   │                       │
       │── push ───────▶│                   │                       │
       │                │──── trigger ─────▶│                       │
       │                │                   │── Lint ──────────┐    │
       │                │                   │── Test ──────────┤    │
       │                │                   │── Vuln ──────────┘    │
       │                │                   │── Build ─────────┐    │
       │                │                   │── Docker ────────┤    │
       │                │                   │── Scan ──────────┘    │
       │                │                   │── Notify ────────────▶│ Discord
       │                │                   │                       │
       │── merge ──────▶│ (develop)         │                       │
       │                │──── auto ────────▶│──── deploy ──────────▶│ 🟡 Staging
       │                │                   │                       │
       │── tag v* ─────▶│                   │                       │
       │                │──── auto ────────▶│──── deploy ──────────▶│ 🟢 Production
       │                │                   │                       │
```

> 📝 รายละเอียดแต่ละ stage → [CI Pipeline Stages](documents/cicd/ci-pipeline-stages.md)
> | Workflow ทั้งหมด → [Workflow Concept](documents/cicd/workflow-concept.md)

---

## 🗄️ Database Architecture

โปรเจกต์ออกแบบ Database Layer ให้รองรับ **หลาย driver** (Multi-Driver) ด้วย interface-driven design

```
  ┌──────────────────────────────────────────────────────────────────────┐
  │                        Database Manager                              │
  │                      (Composition Root)                              │
  ├──────────────────────┬───────────────────────────────────────────────┤
  │                      │                                               │
  │   ┌──────────────────▼──────────────────┐                            │
  │   │        provider.go (Interface)       │  ← Module เรียกผ่านนี้    │
  │   │  Main() → *pgxpool.Pool             │                            │
  │   │  Read() / Write()                   │                            │
  │   │  External(name) → ExternalConn      │                            │
  │   └──────────────────┬──────────────────┘                            │
  │                      │                                               │
  │          ┌───────────┴───────────┐                                   │
  │          ▼                       ▼                                   │
  │   ┌─────────────┐       ┌─────────────────┐                         │
  │   │  postgres/  │       │  External DBs   │                         │
  │   │             │       │                 │                         │
  │   │ Main DB     │       │  ┌───────────┐  │                         │
  │   │ (pgxpool)   │       │  │ postgres  │  │  ← driver: postgres     │
  │   │             │       │  │ (pgx v5)  │  │                         │
  │   │ • connect   │       │  └───────────┘  │                         │
  │   │ • migrate   │       │  ┌───────────┐  │                         │
  │   │ • TLS       │       │  │  mysql     │  │  ← driver: mysql       │
  │   │ • pool tune │       │  │ (go-sql)  │  │                         │
  │   └─────────────┘       │  └───────────┘  │                         │
  │                         └─────────────────┘                         │
  └──────────────────────────────────────────────────────────────────────┘
```

### หลักการออกแบบ

| หลักการ | รายละเอียด |
|---------|-----------|
| **Interface แยกจาก Implementation** | `provider.go` ไม่ import driver ใดๆ — module ใช้ interface เท่านั้น |
| **Driver แต่ละตัวเป็นอิสระ** | `postgres/` ไม่รู้จัก `mysql/` — ป้องกัน circular dependency |
| **Type-Safe Helpers** | `database.PgxPool(conn)` / `database.SQLDB(conn)` ดึง connection อย่างปลอดภัย |
| **เพิ่ม Driver ง่าย** | สร้าง folder ใหม่ + เพิ่ม 1 case ใน `manager.go` |
| **Backward Compatible** | `Driver` ว่าง → default เป็น `"postgres"` — module เดิมไม่ต้องแก้ |

### External Database Config

```env
EXTERNAL_DBS=partner_a,legacy_erp

# partner_a — postgres (default driver)
EXTERNAL_DBS_PARTNER_A_HOST=db.partner.com
EXTERNAL_DBS_PARTNER_A_PORT=5432

# legacy_erp — mysql
EXTERNAL_DBS_LEGACY_ERP_DRIVER=mysql
EXTERNAL_DBS_LEGACY_ERP_HOST=10.0.0.5
EXTERNAL_DBS_LEGACY_ERP_PORT=3306
```

> 📝 รายละเอียด → [Database Concept](documents/architecture/database-concept.md)

---

## 📖 Swagger — API Documentation

API Documentation ใช้ **swag** generate จาก Go annotations → Swagger UI พร้อมดู/ทดสอบ API ได้ทันที

```
  ┌──────────────────────────────────────────────────────────┐
  │                     Swagger Flow                          │
  │                                                           │
  │   Go Source Code          swag init          Swagger UI   │
  │   ┌──────────────┐      ┌─────────┐      ┌────────────┐  │
  │   │ // @Summary  │ ───▶ │  docs/  │ ───▶ │    /swagger │  │
  │   │ // @Param    │      │ .json   │      │    /index   │  │
  │   │ // @Success  │      │ .yaml   │      │    .html    │  │
  │   └──────────────┘      └─────────┘      └────────────┘  │
  │                                                           │
  │   Annotations อยู่ใน    Auto-generated     เปิดผ่าน       │
  │   handler.go ของ        ไม่ต้องเขียนเอง    browser        │
  │   แต่ละ module                                            │
  └──────────────────────────────────────────────────────────┘
```

```powershell
.\run.ps1 swagger                     # generate docs
# เปิด → http://localhost:20000/swagger/index.html
```

> 📝 รายละเอียด → [Swagger Concept](documents/architecture/swagger-concept.md)

---

## 📊 Grafana — Observability Dashboard

ระบบ Observability ใช้ **OpenTelemetry** เก็บ traces + metrics แล้วดูผลผ่าน **Grafana**

```
  ┌────────────────────────────────────────────────────────────────────┐
  │                      Observability Stack                           │
  │                                                                    │
  │   Go App                OTel Collector           Backends          │
  │   ┌──────────┐         ┌──────────────┐         ┌──────────┐      │
  │   │  Fiber   │──OTLP──▶│              │────────▶│  Tempo   │      │
  │   │  Kafka   │         │   receive    │         │ (traces) │      │
  │   │  pgx     │         │   process    │         └──────────┘      │
  │   │  HTTP    │         │   export     │         ┌──────────┐      │
  │   └──────────┘         │              │────────▶│Prometheus│      │
  │                        └──────────────┘         │(metrics) │      │
  │                                                 └──────────┘      │
  │                                                       │           │
  │                                                 ┌─────▼──────┐    │
  │                                                 │  Grafana   │    │
  │                                                 │ Dashboard  │    │
  │                                                 │ :3000      │    │
  │                                                 └────────────┘    │
  └────────────────────────────────────────────────────────────────────┘
```

```powershell
.\run.ps1 otel-up         # เปิด stack ทั้งหมด
# Grafana  → http://localhost:3000
# Tempo    → http://localhost:3200
```

สิ่งที่ trace ได้ทันที: **HTTP requests**, **Kafka produce/consume**, **DB queries**, **External HTTP calls**

> 📝 รายละเอียด → [OTel Tracing Guide](documents/observability/otel-tracing-guide.md) | [Quick Start](documents/observability/otel-grafana-quickstart.md)

---

## 🤖 Dependabot — Auto Dependency Updates

โปรเจกต์ใช้ **Dependabot** ตรวจ dependency ที่ล้าสมัยหรือมีช่องโหว่ แล้วเปิด PR อัปเดตให้อัตโนมัติ

```
  ┌───────────────────────────────────────────────────────────────┐
  │                    Dependabot Workflow                         │
  │                                                               │
  │   ทุกวันจันทร์ 09:00 (Asia/Bangkok)                           │
  │       │                                                       │
  │       ▼                                                       │
  │   ┌──────────────────────────────────────────────┐            │
  │   │            Dependabot ตรวจ 3 ecosystems      │            │
  │   │                                              │            │
  │   │   📦 gomod          Go modules               │            │
  │   │   ⚙️ github-actions  Action versions          │            │
  │   │   🐳 docker          Base images              │            │
  │   └───────────────┬──────────────────────────────┘            │
  │                   ▼                                           │
  │           มี version ใหม่?                                    │
  │           ├── ✅ Yes → เปิด PR อัตโนมัติ (พร้อม labels)       │
  │           │           → CI รัน test ให้ทันที                   │
  │           └── ❌ No  → ไม่ทำอะไร                              │
  └───────────────────────────────────────────────────────────────┘
```

Config: [`.github/dependabot.yml`](.github/dependabot.yml)

| Ecosystem | Limit PRs | Labels |
|-----------|-----------|--------|
| Go modules | 5 / week | `dependencies`, `go` |
| GitHub Actions | 5 / week | `dependencies`, `ci` |
| Docker images | 3 / week | `dependencies`, `docker` |

> 📝 คู่มือ Dependabot → [Dependabot Guide](documents/cicd/dependabot-guide.md)

---

## 🧪 Unit Test — Testing Strategy

โปรเจกต์ใช้ **Go standard library `testing` + `internal/testkit`** ที่สร้างเอง — **ไม่มี external test dependency** (ไม่มี testify, gomock, mockery)

```
  ┌────────────────────────────────────────────────────────────────────┐
  │                     Testing Architecture                           │
  │                                                                    │
  │   Handler Test          Service Test           Repo Test           │
  │  ┌────────────┐       ┌────────────┐        ┌────────────┐        │
  │  │ fakeRepo   │       │   Fakes    │        │  fakeRow   │        │
  │  │ + Fiber    │──svc─▶│ (struct    │──port─▶│ (pgx.Row)  │        │
  │  │ + httptest │       │  fields)   │        │            │        │
  │  └─────┬──────┘       └─────┬──────┘        └─────┬──────┘        │
  │        │                    │                     │                │
  │   HTTP layer           Business logic        Scan / SQL logic     │
  │   status code          domain rules          unmarshal JSON       │
  │   response format      error handling        query fragments      │
  │   trace_id             dependency calls                           │
  └────────────────────────────────────────────────────────────────────┘
```

### สถิติ

| Metric | Value |
|--------|-------|
| Test packages | 18 |
| Test files | 25+ |
| Fakes files | 6 |
| External test deps | **0** |
| testkit functions | 17 (11 assert + 6 must) |
| Test layers | Service · Handler · Repository |

### Test Patterns ที่ใช้

| Pattern | ใช้ที่ | ตัวอย่าง |
|---------|--------|---------|
| **Table-Driven Tests** | ทุก layer | `[]struct{ name, want }` + `t.Run()` |
| **Hand-Written Fakes** | Service · Handler | struct fields กำหนดค่า return |
| **Closure-Based Mocks** | verify behavior | closure fields + call count |
| **fakeRow (pgx.Row)** | Repository | จำลอง DB row สำหรับ scan test |
| **setupApp + doRequest** | Handler | สร้าง Fiber app + httptest ทดสอบ HTTP |

### testkit — Assertion Helpers

`internal/testkit/` เป็น package ที่เขียนเองด้วย **Go Generics** ไม่มี external dependency

```go
testkit.Equal(t, got, want, "label")      // เทียบค่า
testkit.NoError(t, err)                    // ไม่มี error
testkit.ErrorIs(t, err, ErrNotFound)       // error ตรง target
testkit.Contains(t, body, "success")       // string contains
testkit.MustNoError(t, err, "setup")       // fatal ถ้า fail (ใช้ตอน setup)
```

### วิธีรัน

```powershell
.\run.ps1 test                                    # test ทั้งหมด
go test ./internal/modules/auth/app/ -v -count=1  # เฉพาะ package
go test ./... -race -count=1                       # พร้อม race detector
go test ./... -coverprofile=coverage.out           # พร้อม coverage
```

> 📝 รายละเอียด → [Unit Test Guide](documents/testing/unit-test-guide.md) | [Cheatsheet](documents/testing/unit-test-cheatsheet.md)

---

## 📈 Coverage & Test Logging

แนวคิดการเก็บ coverage และบันทึกผลการ test — ทุกครั้งที่รัน จะมี **report + Discord notification** อัตโนมัติ

```
  ┌─────────────────────────────────────────────────────────────────┐
  │                  Coverage & Logging Flow                        │
  │                                                                 │
  │   Developer                                                     │
  │       │                                                         │
  │       ▼                                                         │
  │   .\run.ps1 test-cover                                          │
  │       │                                                         │
  │       ├──▶ go test -coverprofile -covermode atomic ./...        │
  │       │         │                                               │
  │       │         ├── coverage.out  (raw data)                    │
  │       │         └── per-package % breakdown                     │
  │       │                                                         │
  │       ├──▶ Threshold Check (≥ 25%)                              │
  │       │         ├── PASS → ✅ continue                          │
  │       │         └── FAIL → ❌ exit 1                            │
  │       │                                                         │
  │       └──▶ Discord Notification                                 │
  │                 │                                               │
  │                 ▼                                               │
  │   ┌──────────────────────────────────┐                          │
  │   │ ✅ Coverage Report — 30.9%       │                          │
  │   │ ┌──────────────────────────────┐ │                          │
  │   │ │ ENV      = local             │ │                          │
  │   │ │ STATUS   = PASSED            │ │                          │
  │   │ │ COVERAGE = 30.9%             │ │                          │
  │   │ │ THRESHOLD= 25%              │ │                          │
  │   │ │ TESTED AT= 31 Mar 2026 22:15│ │                          │
  │   │ │ MACHINE  = user@DESKTOP      │ │                          │
  │   │ └──────────────────────────────┘ │                          │
  │   │ 🔀 Branch  📝 Commit  👤 Author │                          │
  │   │ 📦 Per-package breakdown        │                          │
  │   └──────────────────────────────────┘                          │
  └─────────────────────────────────────────────────────────────────┘
```

### คำสั่งที่ใช้

| คำสั่ง | ผลลัพธ์ |
|--------|---------|
| `.\run.ps1 test-cover` | รัน test + coverage + threshold check + Discord log |
| `.\run.ps1 ci` | full pipeline (lint → test → vuln → build) + Discord log |
| `go tool cover -html=coverage.out` | เปิด HTML report ในเบราว์เซอร์ |
| `go tool cover -func=coverage.out` | แสดง coverage แต่ละ function |

### สิ่งที่ถูกบันทึก (Discord Log)

ทุกครั้งที่รัน `test-cover` หรือ `ci` จะบันทึกลง Discord พร้อมข้อมูล:

| ข้อมูล | ตัวอย่าง | ทำไมต้องเก็บ |
|--------|---------|--------------|
| วัน-เวลาที่ test | `31 Mar 2026 22:15:30` | ดูว่าเทสล่าสุดเมื่อไหร่ |
| Coverage % | `30.9%` | ดู trend ว่าขึ้นหรือลง |
| Threshold | `25%` | ป้องกัน coverage ตกต่ำกว่าเกณฑ์ |
| Branch + Commit | `main` / `8574c0d` | ผูกกับ code version |
| Machine / Author | `guitar@DESKTOP` | รู้ว่าใครรัน จาก machine ไหน |
| Per-package breakdown | แต่ละ package กี่ % | หา package ที่ coverage ต่ำ |
| PASS / FAIL status | ✅ / ❌ | แจ้งเตือนทีมทันทีถ้า fail |

### Threshold Strategy

```
เริ่มต้น:  25%  ← ปัจจุบัน (โปรเจกต์ใหม่ยังมี test ไม่ครบ)
   ↓
ค่อย ๆ ขึ้น:  +5% ทุกครั้งที่เพิ่ม test ครบ module
   ↓
เป้าหมาย:  70%  ← เกณฑ์มาตรฐานสำหรับ production
```

> ปรับ threshold ที่ `run.ps1` → `$threshold = 25` และ `Makefile` → `COVERAGE_THRESHOLD ?= 70`

### ทำไมต้องส่ง Discord?

- **Visibility** — ทีมเห็นผล test ทันทีไม่ต้องเปิด terminal
- **History** — Discord เก็บ log ย้อนหลังได้ ดู trend coverage ตาม timeline
- **Accountability** — รู้ว่า commit ไหนทำ coverage ตก ใครเป็นคนรัน
- **Pattern เดียวกัน** — ทั้ง `ci` และ `test-cover` ส่ง Discord format เดียวกัน (infra-style)

---

## 🏗️ โครงสร้างโปรเจกต์

```
anc-portal-be/
├── cmd/                   ← Entry points (6 binaries)
│   ├── api/               ← HTTP server (Fiber)
│   ├── worker/            ← Kafka consumer
│   ├── migrate/           ← Database migration
│   ├── seed/              ← Data seeding
│   ├── import/            ← CSV import tool
│   └── sync/              ← External DB sync
│
├── internal/              ← Business logic
│   ├── modules/           ← Feature modules (hexagonal)
│   │   ├── auth/          ← Authentication
│   │   ├── cmi/           ← พรบ. เดี่ยว
│   │   ├── quotation/     ← ใบเสนอราคา
│   │   ├── document/      ← เอกสาร
│   │   ├── job/           ← งาน (placeholder)
│   │   ├── notification/  ← การแจ้งเตือน
│   │   ├── externaldb/    ← External DB health check
│   │   ├── webhook/       ← GitHub Webhook → Discord
│   │   ├── policy/        ← กรมธรรม์ (future)
│   │   └── payment/       ← การชำระเงิน (future)
│   ├── database/          ← Multi-driver DB layer
│   ├── shared/            ← DTO, enum, error codes, pagination, validator
│   ├── testkit/           ← Test assertion library (Go Generics)
│   ├── import/            ← CSV importers
│   └── sync/              ← Data sync framework
│
├── pkg/                   ← Reusable packages
│   ├── otel/              ← OpenTelemetry (tracing + metrics)
│   ├── kafka/             ← Kafka producer/consumer + DLQ
│   ├── cache/             ← Redis cache client
│   ├── localcache/        ← Otter in-memory cache (L1→L2 hybrid)
│   ├── httpclient/        ← HTTP client + retry + tracing + circuit breaker
│   ├── retry/             ← Retry strategies (exponential, linear, constant)
│   ├── log/               ← zerolog wrapper
│   ├── banner/            ← Startup banner (ANSI box-drawing)
│   └── buildinfo/         ← Git commit + build time (ldflags)
│
├── server/                ← Fiber server + routing + middleware
├── config/                ← Viper configuration loader
├── migrations/            ← SQL migration files
├── deployments/           ← Docker + K8s manifests
│   ├── docker/            ← Multi-stage Dockerfile
│   ├── k8s/               ← Kustomize base + overlays
│   ├── local/             ← docker-compose (dev dependencies)
│   └── observability/     ← OTel Collector + Grafana stack
├── documents/             ← Technical documentation
└── docs/                  ← Swagger generated files (auto)
```

---

## 🎯 Entry Points

| | Command | คำอธิบาย | วิธีรัน |
|---|---|---|---|
| 🌐 | **api** | HTTP server (Fiber) — REST API หลัก | `.\run.ps1 dev` |
| 👷 | **worker** | Kafka consumer — งาน background | `.\run.ps1 worker` |
| 🗄️ | **migrate** | Database migration (golang-migrate) | `.\run.ps1 migrate` |
| 🌱 | **seed** | Seed ข้อมูลเริ่มต้น | `.\run.ps1 seed` |
| 📥 | **import** | CSV import (insurer, user, province) | `go run ./cmd/import --help` |
| 🔄 | **sync** | Data sync จาก External DB → Main DB | `go run ./cmd/sync --help` |

---

## 📚 เอกสาร

> ดูสารบัญรวม → [documents/README.md](documents/README.md)

| | เอกสาร | หัวข้อ |
|---|---|---|
| 🏛️ | [Architecture](documents/architecture/README.md) | Modular Monolith, Hexagonal, module structure |
| 📖 | [Swagger Guide](documents/architecture/swagger-concept.md) | Swagger/OpenAPI — annotation, วิธีใช้ |
| 🗄️ | [Database Concept](documents/architecture/database-concept.md) | Multi-driver DB, External DB, connection pool |
| 🔄 | [CI/CD Guide](documents/cicd/ci-cd-guide.md) | Pipeline, GitHub Actions, Local CI |
| 📋 | [CI Pipeline Stages](documents/cicd/ci-pipeline-stages.md) | 7 stages — Lint, Test, Vuln, Build, Docker, Scan, Notify |
| 🔀 | [Workflow Concept](documents/cicd/workflow-concept.md) | push → CI → staging → tag → production |
| 🤖 | [Dependabot Guide](documents/cicd/dependabot-guide.md) | Dependabot คืออะไร + วิธีจัดการ PR |
| 📊 | [OTel Tracing](documents/observability/otel-tracing-guide.md) | Distributed tracing, Kafka propagation |
| ⚡ | [Grafana Quick Start](documents/observability/otel-grafana-quickstart.md) | Observability stack setup (5 นาที) |
| 🚀 | [Deployment Guide](documents/infrastructure/deployment-guide.md) | Local → Staging → Production |
| ☸️ | [Kubernetes Guide](documents/infrastructure/kubernetes-guide.md) | K8s manifests, Kustomize overlays |
| 🌐 | [INET Readiness](documents/infrastructure/inet-readiness-assessment.md) | ประเมินความพร้อม deploy บน INET Cloud |
| 💾 | [Redis Cache](documents/infrastructure/redis-cache-guide.md) | Cache patterns, Hybrid L1→L2 |
| 🧪 | [Unit Test Guide](documents/testing/unit-test-guide.md) | Test patterns, testkit, fakes |
| 📝 | [Unit Test Cheatsheet](documents/testing/unit-test-cheatsheet.md) | Quick reference — commands, patterns |
| 📥 | [Import Data](cmd/import/import_data_guide.md) | CSV import — insurer, province, user |

---

## 🧰 Tech Stack

| Layer | เทคโนโลยี |
|---|---|
| **Language** | Go 1.25 |
| **HTTP** | Fiber v2.52 |
| **Database** | PostgreSQL + pgx v5 (multi-driver, connection pool) |
| **External DB** | MySQL / PostgreSQL — เพิ่ม driver ได้ |
| **Cache** | Redis (go-redis v9) + Otter (in-memory, L1→L2 hybrid) |
| **Messaging** | Kafka (segmentio, KRaft mode, DLQ) |
| **Observability** | OpenTelemetry → Tempo + Prometheus + Grafana |
| **CI/CD** | GitHub Actions (7 stages) + Dependabot |
| **Config** | Viper (12-factor, env vars) |
| **Logging** | zerolog (structured JSON/console) |
| **Deploy** | Docker multi-stage + Kubernetes (Kustomize) |
| **API Docs** | Swagger/OpenAPI (swag + fiber-swagger) |
| **Security** | Trivy image scan, govulncheck, TLS 1.2+ |

---

> 💬 **GitHub Webhook** → Discord `#be-github-notification` — แจ้งเตือนทุก push, PR, CI result
