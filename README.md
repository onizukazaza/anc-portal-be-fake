# ANC Portal Backend — Blueprint Starter

> **Branch นี้คือ skeleton template** สำหรับเริ่มโปรเจกต์ใหม่ที่ใช้ architecture เดียวกัน
> Clone branch นี้แล้วเริ่ม implement module ของคุณเองได้เลย
>
> `Go 1.25` · `Fiber v2` · `PostgreSQL` · `Redis` · `Kafka` · `OpenTelemetry`

---

## สิ่งที่มีใน Blueprint

### Infrastructure (พร้อมใช้)

| Layer | รายละเอียด |
|-------|-----------|
| **Config** | Viper config loader (.env + yaml + env vars) |
| **Database** | Multi-driver (Postgres + MySQL), connection pool, health check |
| **Cache** | Redis client + Otter in-memory + Hybrid L1/L2 |
| **HTTP Client** | Retry + Circuit Breaker + OTel tracing |
| **Kafka** | Producer + Consumer + DLQ + Event Router |
| **Observability** | OpenTelemetry tracing + Prometheus metrics + Grafana |
| **Logging** | zerolog (colored console dev + JSON prod) |
| **Middleware** | JWT Auth + API Key + Access Log + Rate Limit + CORS |
| **Pagination** | Request/Response + SQL builders + Fiber query parser |
| **Validation** | go-playground/validator + BindAndValidate helper |
| **Test Kit** | assert/must helpers — ไม่มี external test dependency |

### Example Module (ตัวอย่างครบ pattern)

```
internal/modules/example/
├── domain/example.go              ← Pure struct
├── ports/repository.go            ← Go interface (contract)
├── app/
│   ├── service.go                 ← Business logic
│   ├── service_test.go            ← Unit test (ใช้ testkit)
│   └── fakes_test.go              ← In-memory fake repository
├── adapters/
│   ├── http/
│   │   ├── controller.go          ← Handler group
│   │   └── handler.go             ← Fiber handlers
│   └── postgres/
│       └── repository.go          ← PostgreSQL implementation
└── module.go                      ← Composition root (wiring)
```

### Auth Module (skeleton — Token Signer พร้อมใช้)

```
internal/modules/auth/
├── domain/auth.go                 ← User + Session structs
├── ports/
│   ├── token_signer.go            ← TokenSigner interface (JWT/dev)
│   └── user_repository.go         ← UserRepository interface
├── adapters/external/
│   ├── jwt_token_signer.go        ← JWT HS256 (production)
│   └── simple_token_signer.go     ← Plaintext dev token
└── module.go                      ← NewTokenSigner() factory
```

---

## Quick Start

```bash
# 1. Clone blueprint
git clone -b blueprint/starter <repo-url> my-project
cd my-project

# 2. Start dependencies (Postgres + Redis + Kafka)
make local-up

# 3. Run migration
make migrate

# 4. Start API (hot-reload)
make dev

# 5. Verify
curl http://localhost:3000/healthz
```

---

## สร้าง Module ใหม่

### 1. สร้าง folder structure

```bash
mkdir -p internal/modules/{name}/{domain,ports,app,adapters/http,adapters/postgres}
```

### 2. ตาม pattern นี้

```
domain/      → Pure structs (ห้าม import อะไรเลย)
ports/       → Go interface เท่านั้น (1-3 methods)
app/         → Business logic + inject ports ผ่าน constructor
adapters/    → HTTP handler, PostgreSQL repo, external API client
module.go    → Composition root: wire concrete → service → controller → routes
```

### 3. Register ใน server.go

```go
// server/server.go → registerRoutes()
import "github.com/onizukazaza/anc-portal-be-fake/internal/modules/{name}"

// ใน registerRoutes():
{name}.Register(api, deps)
```

---

## Architecture

```
cmd/api/main.go          → Bootstrap + Graceful shutdown
  ├── config/            → Viper config
  ├── internal/database/ → Multi-driver DB manager
  ├── pkg/               → Infrastructure packages
  └── server/
      ├── server.go      → Fiber app + middleware + route registration
      ├── middleware/     → Auth (JWT/API Key) + Access Log
      └── modules/       → Feature modules (hexagonal)

handler (http) → service (app) → port (interface) ← adapter (postgres)
                                       ↓
                                    domain (pure)
```

---

## กฎสำคัญ

| กฎ | ทำไม |
|----|------|
| **ห้าม import ข้าม layer** | domain ไม่ import ports/app/adapters |
| **ห้ามใช้ external test library** | ใช้ `internal/testkit` เท่านั้น |
| **ห้าม raw type assertion บน ExternalConn** | ใช้ `database.PgxPool()` / `database.SQLDB()` |
| **ใช้ sentinel errors** | `var ErrNotFound = errors.New(...)` + `errors.Is()` |
| **ใช้ early return** | ห้าม if-else สำหรับ error |
| **Enum เป็น string const** | ห้ามใช้ `iota` |

---

## Commands

| Command | Description |
|---------|-------------|
| `make dev` | API server with hot-reload (air) |
| `make build` | Build API binary |
| `make test` | Run all tests |
| `make test-cover` | Tests + coverage report |
| `make lint` | golangci-lint |
| `make ci` | Full CI pipeline (lint → test → vuln → build) |
| `make migrate` | Run database migrations |
| `make worker` | Start Kafka worker |
| `make local-up` | Start local dependencies (Postgres + Redis + Kafka) |
| `make otel-up` | Start observability stack (Grafana + Tempo + Prometheus) |
| `make swagger` | Generate Swagger docs |

---

## โครงสร้างโปรเจกต์

```
├── cmd/
│   ├── api/main.go              ← API server entry point
│   ├── migrate/main.go          ← Database migration CLI
│   └── worker/main.go           ← Kafka consumer worker
├── config/                      ← Viper config loader
├── internal/
│   ├── database/                ← Multi-driver DB manager
│   │   ├── provider.go          ← Provider interface
│   │   ├── conn.go              ← ExternalConn + type-safe helpers
│   │   ├── manager.go           ← Composition root (postgres/mysql)
│   │   ├── postgres/            ← Postgres driver + migration
│   │   └── mysql/               ← MySQL driver
│   ├── modules/
│   │   ├── auth/                ← Auth skeleton (TokenSigner ready)
│   │   └── example/             ← Complete example module
│   ├── shared/
│   │   ├── dto/                 ← API response + error codes
│   │   ├── enum/                ← String const enums
│   │   ├── module/              ← Deps + Module interface
│   │   ├── pagination/          ← Pagination helpers
│   │   ├── utils/               ← ID, JSON, pointer, string helpers
│   │   └── validator/           ← Request validation
│   └── testkit/                 ← Test assertions (NO external deps)
├── migrations/                  ← SQL migration files
├── pkg/
│   ├── banner/                  ← Startup banner
│   ├── buildinfo/               ← Git commit + build time
│   ├── cache/                   ← Redis client
│   ├── discord/                 ← Discord webhook client
│   ├── httpclient/              ← HTTP + retry + circuit breaker
│   ├── kafka/                   ← Producer + Consumer + DLQ
│   ├── localcache/              ← Otter + Hybrid cache
│   ├── log/                     ← zerolog wrapper
│   ├── otel/                    ← OpenTelemetry setup
│   └── retry/                   ← Retry strategies
├── server/
│   ├── server.go                ← Fiber app + routes
│   └── middleware/              ← Auth + Access Log
├── deployments/
│   ├── docker/                  ← Dockerfile (multi-target)
│   ├── k8s/                     ← Kubernetes manifests
│   ├── local/                   ← Docker Compose (dev deps)
│   └── observability/           ← OTel + Grafana stack
└── documents/                   ← Architecture docs
```

---

## รายละเอียดเพิ่มเติม

ดู `.github/instructions/` สำหรับกฎละเอียด:

- `architecture.instructions.md` — Hexagonal rules, module registration
- `go-conventions.instructions.md` — Import order, naming, error handling
- `testing.instructions.md` — testkit, fakes, test patterns
- `database.instructions.md` — pgx, ExternalConn, SQL conventions
- `project-overview.instructions.md` — Design philosophy overview
