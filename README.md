# ANC Portal Backend

> **Go 1.25 · Fiber v2 · PostgreSQL · Redis · Kafka · OpenTelemetry**
>
> Backend API สำหรับระบบ ANC Insurance Portal
> ออกแบบแบบ Modular Monolith + Hexagonal Architecture

---

## สารบัญ

- [ANC Portal Backend](#anc-portal-backend)
  - [สารบัญ](#สารบัญ)
  - [Quick Start](#quick-start)
    - [ขั้นตอนที่ 1 — เริ่ม Infrastructure](#ขั้นตอนที่-1--เริ่ม-infrastructure)
    - [ขั้นตอนที่ 2 — Setup Database](#ขั้นตอนที่-2--setup-database)
    - [ขั้นตอนที่ 3 — รัน API Server](#ขั้นตอนที่-3--รัน-api-server)
  - [คำสั่งที่ใช้บ่อย](#คำสั่งที่ใช้บ่อย)
    - [Development](#development)
    - [Database \& Data](#database--data)
    - [Infrastructure](#infrastructure)
  - [โครงสร้างโปรเจกต์](#โครงสร้างโปรเจกต์)
  - [Entry Points](#entry-points)
  - [เอกสาร](#เอกสาร)
  - [Tech Stack](#tech-stack)

---

## Quick Start

### ขั้นตอนที่ 1 — เริ่ม Infrastructure

```powershell
.\run.ps1 local-up      # PostgreSQL + Redis + Kafka
.\run.ps1 otel-up       # Grafana + Tempo + Prometheus + OTel Collector (optional)
```

### ขั้นตอนที่ 2 — Setup Database

```powershell
.\run.ps1 migrate        # สร้าง tables
.\run.ps1 seed           # insert ข้อมูลเริ่มต้น (users, roles)
```

### ขั้นตอนที่ 3 — รัน API Server

```powershell
.\run.ps1 dev            # hot-reload ด้วย Air
```

เปิด Swagger UI: http://localhost:20000/swagger/index.html

---

## คำสั่งที่ใช้บ่อย

ใช้ `.\run.ps1 <command>` (Windows) หรือ `make <command>` (Linux/macOS)

### Development

| คำสั่ง | คำอธิบาย |
|---|---|
| `dev` | รัน API server ด้วย Air (hot-reload) |
| `build` | Build API binary พร้อม git commit info |
| `test` | รัน tests ทั้งหมด |
| `swagger` | Generate Swagger docs จาก code annotations |
| `tidy` | `go mod tidy` |
| `clean` | ลบ build artifacts |

### Database & Data

| คำสั่ง | คำอธิบาย |
|---|---|
| `migrate` | รัน database migrations |
| `seed` | Seed ข้อมูลเริ่มต้น |
| `import` | Import CSV data (ต้องระบุ ENV, PATH, TYPE) |
| `worker` | รัน Kafka consumer |

### Infrastructure

| คำสั่ง | คำอธิบาย |
|---|---|
| `local-up` / `local-down` | เปิด/ปิด PostgreSQL + Redis + Kafka |
| `otel-up` / `otel-down` | เปิด/ปิด Observability stack |
| `docker-build` | Build Docker image (API) |
| `docker-build-worker` | Build Docker image (Worker) |

---

## โครงสร้างโปรเจกต์

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
│   │   ├── externaldb/    ← External DB health check
│   │   ├── webhook/       ← GitHub Webhook → Discord notification
│   │   ├── policy/        ← กรมธรรม์ (future)
│   │   └── payment/       ← การชำระเงิน (future)
│   ├── database/          ← DB connection + migration
│   ├── shared/            ← Shared code (dto, enum, error codes, pagination, validator)
│   ├── testkit/           ← Test assertion library
│   ├── import/            ← CSV importers
│   └── sync/              ← Data sync framework
│
├── pkg/                   ← Reusable packages
│   ├── otel/              ← OpenTelemetry (tracing + metrics)
│   ├── kafka/             ← Kafka producer/consumer + DLQ
│   ├── cache/             ← Redis cache client
│   ├── localcache/        ← Otter in-memory cache
│   ├── httpclient/        ← HTTP client + retry + tracing + circuit breaker
│   ├── retry/             ← Retry strategies (exponential, linear, constant)
│   ├── log/               ← zerolog wrapper
│   ├── banner/            ← Startup banner (ANSI box-drawing)
│   └── buildinfo/         ← Git commit + build time (ldflags)
│
├── server/                ← Fiber server setup + routing + middleware
│   └── middleware/        ← Custom middlewares (access log)
├── config/                ← Viper configuration loader
├── migrations/            ← SQL migration files
├── deployments/           ← Docker + K8s manifests
│   ├── docker/            ← Multi-stage Dockerfile
│   ├── k8s/               ← Kustomize base + overlays
│   ├── local/             ← docker-compose (dev dependencies)
│   └── observability/     ← OTel Collector + Grafana stack
├── documents/             ← Architecture & integration docs
└── docs/                  ← Swagger generated files (auto)
```

---

## Entry Points

โปรเจกต์มี binary เดียว แยก 6 commands:

| Command | คำอธิบาย | วิธีรัน |
|---|---|---|
| **api** | HTTP server (Fiber) — REST API หลัก | `.\run.ps1 dev` |
| **worker** | Kafka consumer — งาน background | `.\run.ps1 worker` |
| **migrate** | Database migration (golang-migrate) | `.\run.ps1 migrate` |
| **seed** | Seed ข้อมูลเริ่มต้น | `.\run.ps1 seed` |
| **import** | CSV import (insurer, user, province) | `go run ./cmd/import --help` |
| **sync** | Data sync จาก External DB → Main DB | `go run ./cmd/sync --help` |

---

## เอกสาร

| เอกสาร | หัวข้อ |
|---|---|
| [Architecture](documents/architecture/README.md) | Clean Architecture, module structure, design patterns |
| [Swagger Guide](documents/architecture/swagger-concept.md) | Swagger/OpenAPI — annotation, วิธีใช้, เปรียบเทียบเครื่องมือ |
| [Deployment Guide](documents/infrastructure/deployment-guide.md) | Local → Staging → Production deployment |
| [K8s Manifests](deployments/k8s/README.md) | Kubernetes resources, overlays, troubleshooting |
| [OTel Tracing](documents/observability/otel-tracing-guide.md) | Distributed tracing, Kafka propagation, tracer registry |
| [OTel Quick Start](documents/observability/otel-grafana-quickstart.md) | Observability stack setup (5 นาที) |
| [Redis Cache](documents/infrastructure/redis-cache-guide.md) | Cache patterns, interface design, TTL strategy |
| [GitHub → Discord](documents/infrastructure/discord-notification.md) | Webhook notification design |
| [Unit Test Guide](documents/testing/unit-test-guide.md) | Test patterns, testkit, fakes, กฎการเขียน test |
| [Microservice Readiness](documents/architecture/microservice-readiness.md) | Microservice คืออะไร, readiness score, แผน extract |
| [CI/CD Guide](documents/cicd/ci-cd-guide.md) | CI/CD pipeline, GitHub Actions, Docker build, K8s deploy |
| [CI Pipeline Stages](documents/cicd/ci-pipeline-stages.md) | อธิบายทุกขั้นตอน CI/CD — Lint, Test, Vuln, Build |
| [Project Structure](documents/architecture/project-structure.md) | Directory layout, module map, layer descriptions |
| [Unit Test Cheatsheet](documents/testing/unit-test-cheatsheet.md) | Test architecture summary — patterns, counts, coverage |
| [Import Data](cmd/import/import_data_guide.md) | CSV import — insurer, province, user |

---

## Tech Stack

| Layer | เทคโนโลยี |
|---|---|
| **Language** | Go 1.25 |
| **HTTP** | Fiber v2.52 |
| **Database** | PostgreSQL + pgx v5 (multi-DB, connection pool) |
| **Cache** | Redis (go-redis v9) + Otter (in-memory) |
| **Messaging** | Kafka (segmentio, KRaft mode) |
| **Observability** | OpenTelemetry → Tempo + Prometheus + Grafana |
| **Config** | Viper (12-factor, env vars) |
| **Logging** | zerolog (structured JSON/console) |
| **Deploy** | Docker multi-stage + Kubernetes (Kustomize) |
| **API Docs** | Swagger/OpenAPI (swag + fiber-swagger) |

---

> **GitHub Webhook** → ส่ง notification เข้า `#be-github-notification` channel บน Discord
