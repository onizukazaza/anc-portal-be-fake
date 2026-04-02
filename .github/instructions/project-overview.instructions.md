---
description: "Project overview, design philosophy, architecture patterns, and extensibility guide. Read this first to understand the codebase concept before working on any file."
applyTo: "**"
---

# AI Context — ANC Portal Backend

> ไฟล์นี้เป็น **จุดเริ่มต้น** สำหรับ AI ที่ทำงานกับโปรเจกต์นี้
> อ่านเพื่อเข้าใจ concept และ design philosophy — กฎละเอียดอยู่ใน `.github/instructions/`

---

## โปรเจกต์นี้คืออะไร

ANC Portal Backend — ระบบ API สำหรับ Insurance Portal
เขียนด้วย **Go 1.25** บน **Fiber v2** framework

---

## Design Philosophy

โปรเจกต์นี้ยึดหลัก **3 แนวคิดหลัก**:

### 1. Modular Monolith

แต่ละ feature เป็น **module อิสระ** อยู่ใน `internal/modules/{name}/`
module หนึ่งเข้าใจได้โดยไม่ต้องกระโดดไปอ่าน module อื่น

```
internal/modules/
├── auth/          ← Authentication & JWT
├── cmi/           ← Compulsory Motor Insurance
├── quotation/     ← Quotation from ERP
├── externaldb/    ← External DB health monitoring
├── webhook/       ← GitHub → Discord notification
├── document/      ← Document management
├── notification/  ← Notification service
├── job/           ← Job processing (placeholder)
├── payment/       ← Payment (future)
└── policy/        ← Policy (future)
```

### 2. Hexagonal Architecture (Ports & Adapters)

ทุก module แยก layer ตามหน้าที่ — business logic **ไม่ผูกกับ infrastructure**

```
domain ← ports ← app ← adapters
 (pure)  (interface) (logic) (HTTP/DB/External)
```

**หลักการ:** ถ้าเปลี่ยน database จาก Postgres เป็น MongoDB — แก้แค่ `adapters/` ไม่ต้องแตะ `app/` หรือ `domain/`

### 3. Interface-Driven Design

ทุก dependency ระหว่าง layer ผ่าน **Go interface** — ไม่ผูก concrete type
ทำให้ test ง่าย, เปลี่ยน implementation ได้, และเพิ่ม driver ใหม่ไม่กระทบ module เดิม

---

## Architecture Patterns ที่ใช้

| Pattern | ใช้ที่ | ทำไม |
|---------|--------|------|
| **Hexagonal / Ports & Adapters** | ทุก module | แยก business logic ออกจาก infrastructure |
| **Composition Root** | `module.go` ของทุก module | Wire dependency ที่จุดเดียว |
| **Repository Pattern** | `adapters/postgres/` | ซ่อน SQL ไว้ใน adapter, service ไม่เห็น query |
| **Dependency Injection** | constructor ของ Service | inject interface ไม่ใช่ concrete type |
| **Interface Segregation** | `ports/` | interface เล็ก 1-3 methods ต่อตัว |
| **Registry Pattern** | `internal/sync/` | เพิ่ม syncer ใหม่แค่ implement + register |
| **Functional Options** | `pkg/httpclient/`, `pkg/retry/` | config ยืดหยุ่นไม่ต้องแก้ constructor |
| **L1/L2 Cache (Hybrid)** | `pkg/localcache/` | otter (in-memory) + Redis (shared) |
| **Circuit Breaker** | `pkg/httpclient/` | ป้องกัน cascading failure |
| **Dead Letter Queue** | `pkg/kafka/` | message ที่ fail ซ้ำไม่หาย |

---

## โครงสร้าง Module (ต้นแบบ)

ทุก module ใน `internal/modules/` ต้องตาม pattern นี้:

```
internal/modules/{name}/
├── domain/                ← Pure structs เท่านั้น (ห้าม import)
├── ports/                 ← Go interface เท่านั้น (contract)
├── app/                   ← Business logic (inject ports)
│   ├── service.go
│   ├── service_test.go
│   └── fakes_test.go
├── adapters/
│   ├── http/              ← Fiber handler (transport)
│   ├── postgres/          ← Repository (persistence)
│   └── external/          ← Non-DB adapters (JWT, API client)
└── module.go              ← Composition root (wiring)
```

---

## Infrastructure Layer

infrastructure packages ออกแบบเป็น **interface-first** — module ใช้ผ่าน contract ไม่ผูกกับ implementation

```
internal/database/
├── provider.go       ← Interface (module เรียกตัวนี้)
├── conn.go           ← ExternalConn interface + type-safe helpers
├── manager.go        ← Composition root (switch-case driver)
├── postgres/         ← Postgres driver
├── mysql/            ← MySQL driver
└── seed/             ← Seed data

pkg/
├── cache/            ← Redis client (Cache interface)
├── localcache/       ← Otter in-memory + Hybrid L1/L2
├── httpclient/       ← HTTP client + retry + circuit breaker + tracing
├── kafka/            ← Producer + Consumer + DLQ
├── retry/            ← Retry strategies (exponential/constant/linear)
├── otel/             ← OpenTelemetry tracing + Prometheus metrics
├── log/              ← zerolog wrapper
├── buildinfo/        ← Git commit + build time
└── banner/           ← Startup banner

server/
├── server.go         ← Fiber app + middleware + route registration
└── middleware/        ← Auth (JWT + API Key), Access Log

config/               ← Viper config loader (.env + yaml + env vars)
```

---

## Observability Stack

ระบบ monitoring ครบวงจร — ใช้ **OpenTelemetry** เป็น standard (vendor-neutral):

```
Go App → OTel Collector → Tempo (traces) + Prometheus (metrics) → Grafana (UI)
```

ไม่ต้องเพิ่ม Jaeger — Tempo ทำหน้าที่เดียวกันและ integrate กับ Grafana ได้ดีกว่า

---

## เพิ่มของใหม่ทำยังไง

| เพิ่มอะไร | ทำยังไง |
|-----------|---------|
| **Module ใหม่** | สร้าง folder ตาม pattern → register ใน `server.go` |
| **Database driver** | สร้าง `internal/database/{driver}/` → เพิ่ม case ใน `manager.go` |
| **Sync table** | Implement `Syncer` interface → register ใน `Registry` |
| **Middleware** | สร้างใน `server/middleware/` → ใส่ `s.app.Use()` |
| **Cache driver** | Implement `cache.Cache` interface |
| **External API** | ใช้ `pkg/httpclient` + สร้าง adapter ใน `adapters/external/` |

---

## กฎละเอียด (Source of Truth)

ไฟล์ด้านล่างนี้คือ **กฎจริง** ที่มี code example — AI ต้องอ่านก่อน generate code:

| ไฟล์ | ใช้เมื่อ |
|------|----------|
| `copilot-instructions.md` | **อ่านทุกครั้ง** — สรุปสิ่งที่ห้ามทำ + ภาพรวม architecture |
| `architecture.instructions.md` | สร้าง/แก้ module, wiring, layer boundaries |
| `go-conventions.instructions.md` | เขียน Go code — import order, naming, error handling, enums |
| `testing.instructions.md` | เขียน test — testkit, fakes, handler/repo test patterns |
| `database.instructions.md` | เขียน query/repo — pgx, ExternalConn, SQL conventions |

> **ลำดับความสำคัญ:** กฎใน instructions > ไฟล์นี้ > เอกสารใน `documents/`
> ถ้าขัดกันให้ยึดตาม instructions เสมอ
