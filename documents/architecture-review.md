# 📊 ANC Portal Backend — Architecture Review

> สรุปจุดแข็ง จุดอ่อน และแนวทางปรับปรุง
> Last updated: April 2026

---

## ภาพรวม

```
  ┌─────────────────────────────────────────────────────────────────┐
  │                    ANC Portal Backend                            │
  │              Go 1.25 · Fiber v2 · PostgreSQL                     │
  │                                                                  │
  │   ┌────────────┐  ┌────────────┐  ┌────────────┐                │
  │   │  Modules   │  │  Packages  │  │   Infra    │                │
  │   │ (Hexagonal)│  │ (Reusable) │  │ (Deploy)   │                │
  │   │            │  │            │  │            │                │
  │   │ auth       │  │ otel       │  │ Docker     │                │
  │   │ cmi        │  │ kafka      │  │ K8s        │                │
  │   │ quotation  │  │ cache      │  │ CI/CD      │                │
  │   │ document   │  │ httpclient │  │ Grafana    │                │
  │   │ ...        │  │ retry      │  │            │                │
  │   └──────┬─────┘  └──────┬─────┘  └────────────┘                │
  │          │               │                                       │
  │          ▼               ▼                                       │
  │   ┌──────────────────────────────┐                               │
  │   │      Database Layer          │                               │
  │   │  Postgres (main) + External  │                               │
  │   │  Multi-Driver (PG / MySQL)   │                               │
  │   └──────────────────────────────┘                               │
  └─────────────────────────────────────────────────────────────────┘
```

---

## 🧠 Architecture นี้คืออะไร? คล้ายอะไร? ยอดนิยมไหม?

### อธิบายแบบเข้าใจง่าย

โปรเจกต์นี้ใช้ **3 แนวคิดรวมกัน** — เปรียบเทียบกับร้านอาหาร:

```
  ┌─────────────────────────────────────────────────────────────────┐
  │                    เปรียบเทียบกับร้านอาหาร                       │
  │                                                                  │
  │  🍽️ Modular Monolith = ร้านอาหารที่มีหลายเคาน์เตอร์ในอาคารเดียว    │
  │     (อาหารไทย, ญี่ปุ่น, อิตาเลียน — แยกกันทำ แต่อยู่ร้านเดียว)     │
  │                                                                  │
  │  🔌 Hexagonal = เชฟทำอาหารโดยไม่สนว่าจานจะเสิร์ฟแบบไหน            │
  │     (จะเสิร์ฟที่โต๊ะ, ส่ง delivery, หรือ take away — สูตรเดียวกัน)  │
  │                                                                  │
  │  📋 Interface-Driven = เมนูเป็น "สัญญา" ระหว่างลูกค้ากับครัว       │
  │     (ลูกค้าสั่งตามเมนู ไม่ต้องรู้ว่าครัวใช้เตาอะไร)               │
  └─────────────────────────────────────────────────────────────────┘
```

### ศัพท์สำคัญ (พร้อมคำอธิบาย)

| ศัพท์ | ภาษาง่ายๆ | ตัวอย่างในโปรเจกต์ |
|-------|----------|-------------------|
| **Hexagonal Architecture** | "ของข้างในไม่ผูกกับข้างนอก" — เปลี่ยน DB ได้โดยไม่แก้ logic | `ports/` = สัญญา, `adapters/` = ตัวเชื่อม |
| **Ports & Adapters** | ชื่อเดียวกับ Hexagonal — "Port" คือช่องเสียบ, "Adapter" คือปลั๊ก | `ports/repo.go` = ช่องเสียบ, `adapters/postgres/` = ปลั๊ก Postgres |
| **Modular Monolith** | Deploy เป็นก้อนเดียว แต่ข้างในแยก module ชัดเจน | `internal/modules/auth/`, `cmi/`, `document/` |
| **Composition Root** | จุดเดียวที่ต่อสายทุกอย่างเข้าด้วยกัน (เหมือนปลั๊กพ่วง) | `module.go` ของแต่ละ module |
| **Dependency Injection** | ส่งของที่ต้องใช้เข้ามาจากข้างนอก แทนที่จะสร้างเอง | `NewService(repo, cache)` ไม่ใช่ `NewService()` ที่สร้าง repo เอง |
| **Interface Segregation** | interface เล็กๆ ทำแค่ 1-3 อย่าง ไม่ยัดทุกอย่างรวม | `ports/` แต่ละไฟล์มี 1-3 methods |
| **Repository Pattern** | ซ่อน SQL ไว้ในกล่อง — service ไม่เห็น query | `adapters/postgres/` เก็บ SQL ทั้งหมด |
| **Circuit Breaker** | "เบรกเกอร์" ตัด connection ถ้าเรียก service อื่นแล้ว fail ซ้ำ | `pkg/httpclient/` ป้องกัน cascading failure |
| **Dead Letter Queue** | "ถังขยะพิเศษ" เก็บ message ที่ fail — ไม่หาย กลับมา retry ได้ | `pkg/kafka/` DLQ |

---

### โปรเจกต์ดังๆ ที่ใช้ pattern เดียวกัน

```
  ┌──────────────────────────────────────────────────────────────────┐
  │          ใครใช้ Hexagonal / Modular Monolith บ้าง?                │
  │                                                                   │
  │  🏢 Shopify         Modular Monolith (Ruby on Rails)              │
  │     → เริ่มจาก monolith แล้วแยก module ภายใน ไม่ไป microservice   │
  │     → ประกาศว่า "Modular Monolith ดีกว่า microservice สำหรับเรา"  │
  │                                                                   │
  │  📦 Basecamp/DHH    Monolith-first (Ruby on Rails)                │
  │     → ผู้สร้าง Rails สนับสนุน "The Majestic Monolith"            │
  │                                                                   │
  │  🎵 Spotify         Hexagonal / Ports & Adapters (Java + more)    │
  │     → Backend services ใช้ hexagonal pattern ภายใน                │
  │                                                                   │
  │  🎮 Netflix         Hexagonal Architecture (Java)                 │
  │     → แต่ละ microservice ข้างในเป็น hexagonal                     │
  │                                                                   │
  │  🇹🇭 ธนาคารไทยหลายแห่ง  Hexagonal + DDD (Java Spring Boot)       │
  │     → ระบบ core banking ใหม่ๆ ใช้ hexagonal เป็น standard         │
  └──────────────────────────────────────────────────────────────────┘
```

**คำตอบ: ใช่ เป็น pattern ยอดนิยมมาก** — โดยเฉพาะในระบบ enterprise, fintech, insurance

---

### 🥊 เทียบ Architecture Patterns ยอดนิยม (Candidate Comparison)

```
  ┌──────────────── Architecture Patterns ──────────────────┐
  │                                                          │
  │  1. Hexagonal          "ข้างในไม่ผูกข้างนอก"             │
  │  2. Clean Architecture "วงกลมซ้อนกัน 4 ชั้น"            │
  │  3. Onion Architecture "หัวหอมลอก layer ได้"             │
  │  4. Vertical Slice     "หั่นตามคามสามารถ ไม่ใช่ตาม layer" │
  │  5. Traditional MVC    "แบบดั้งเดิม 3 ชั้น"              │
  │                                                          │
  └──────────────────────────────────────────────────────────┘
```

#### 1. Hexagonal (Ports & Adapters) ← **โปรเจกต์นี้ใช้**

```
  ภายนอก ──▶ [Port] ──▶ Business Logic ──▶ [Port] ──▶ ภายนอก
  (HTTP)    (interface)    (ไม่รู้จัก       (interface)   (DB)
                           framework)
```

- **ผู้คิด:** Alistair Cockburn (2005)
- **แนวคิด:** Application อยู่ตรงกลาง — ภายนอก (DB, HTTP, Queue) เชื่อมผ่าน "port" (interface) + "adapter" (implementation)
- **ภาษาง่าย:** _"ต่อปลั๊กอะไรก็ได้ ตราบใดที่เสียบเข้ารูเดียวกัน"_
- **เหมาะกับ:** ระบบที่ต้องเปลี่ยน infrastructure บ่อย, มีหลาย adapter

#### 2. Clean Architecture (Uncle Bob)

```
  ┌─────────────────────────────────────────┐
  │           Frameworks & Drivers           │  ← ชั้นนอกสุด (DB, Web)
  │   ┌─────────────────────────────────┐   │
  │   │      Interface Adapters          │   │  ← Controllers, Gateways
  │   │   ┌─────────────────────────┐   │   │
  │   │   │     Use Cases            │   │   │  ← Application logic
  │   │   │   ┌─────────────────┐   │   │   │
  │   │   │   │    Entities      │   │   │   │  ← ชั้นในสุด (Domain)
  │   │   │   └─────────────────┘   │   │   │
  │   │   └─────────────────────────┘   │   │
  │   └─────────────────────────────────┘   │
  └─────────────────────────────────────────┘

  กฎ: dependency ชี้เข้าข้างในเสมอ (outer → inner, ห้ามย้อน)
```

- **ผู้คิด:** Robert C. Martin / Uncle Bob (2012)
- **แนวคิด:** วงกลม 4 ชั้น — Entities → Use Cases → Interface Adapters → Frameworks
- **ภาษาง่าย:** _"ชั้นในไม่รู้จักชั้นนอก — เหมือนหัวหน้าไม่รู้ว่าลูกน้องใช้ tool อะไร"_
- **เหมาะกับ:** ระบบใหญ่ที่ต้องการ layer ชัดเจน, หลายทีมทำงานพร้อมกัน

#### 3. Onion Architecture

```
  ┌─────────────────────────────────────────┐
  │            Infrastructure                │  ← DB, UI, External
  │   ┌─────────────────────────────────┐   │
  │   │       Application Services       │   │  ← Orchestration
  │   │   ┌─────────────────────────┐   │   │
  │   │   │    Domain Services       │   │   │  ← Business rules
  │   │   │   ┌─────────────────┐   │   │   │
  │   │   │   │   Domain Model   │   │   │   │  ← Entities + Value Objects
  │   │   │   └─────────────────┘   │   │   │
  │   │   └─────────────────────────┘   │   │
  │   └─────────────────────────────────┘   │
  └─────────────────────────────────────────┘
```

- **ผู้คิด:** Jeffrey Palermo (2008)
- **แนวคิด:** เหมือน Clean แต่เน้น "Domain Model" ที่แกนกลางมากกว่า
- **ภาษาง่าย:** _"ลอกหัวหอมทีละชั้น — แต่ละชั้นรู้จักแค่ชั้นข้างใน"_
- **เหมาะกับ:** Domain-Driven Design (DDD), ระบบที่ domain ซับซ้อน

#### 4. Vertical Slice Architecture

```
  ┌──────────┐  ┌──────────┐  ┌──────────┐
  │ Feature A│  │ Feature B│  │ Feature C│
  │          │  │          │  │          │
  │ Handler  │  │ Handler  │  │ Handler  │
  │ Logic    │  │ Logic    │  │ Logic    │
  │ DB Query │  │ DB Query │  │ DB Query │
  │          │  │          │  │          │
  └──────────┘  └──────────┘  └──────────┘

  ⬆ แต่ละ feature เป็นอิสระ — ไม่แบ่งตาม layer แต่แบ่งตาม feature
```

- **ผู้คิด:** Jimmy Bogard (2018+)
- **แนวคิด:** ไม่แบ่ง layer (handler/service/repo) แต่แบ่งตาม feature — แต่ละ feature มีครบในตัว
- **ภาษาง่าย:** _"หั่นเค้กแนวตั้ง ไม่ใช่แนวนอน — แต่ละชิ้นมีครบทุก layer"_
- **เหมาะกับ:** ทีม CQRS, ระบบที่ feature เปลี่ยนบ่อยและไม่ค่อยแชร์ logic

#### 5. Traditional MVC / 3-Layer

```
  Controller ──▶ Service ──▶ Repository ──▶ DB

  (handler)      (logic)     (SQL query)
```

- **แนวคิด:** แบ่งแค่ 3 ชั้น — Controller / Service / Repository
- **ภาษาง่าย:** _"แบบดั้งเดิม ง่ายสุด แต่ service ผูกกับ DB ตรงๆ"_
- **เหมาะกับ:** โปรเจกต์เล็ก, prototype, CRUD ง่ายๆ

---

### 🔍 ตาราง Comparison ละเอียด

| เกณฑ์ | Hexagonal 🏆 | Clean | Onion | Vertical Slice | MVC |
|-------|:----------:|:-----:|:-----:|:--------------:|:---:|
| **ความยอดนิยม** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **ความซับซ้อน** | ปานกลาง | สูง | สูง | ต่ำ-ปานกลาง | ต่ำ |
| **เหมาะกับ Go** | ✅ ดีมาก | ✅ ดี | ⚠️ เยอะไป | ✅ ดี | ✅ ดี |
| **เปลี่ยน DB ง่าย** | ✅ | ✅ | ✅ | ⚠️ ขึ้นกับ code | ❌ |
| **Test ง่าย** | ✅ | ✅ | ✅ | ✅ | ⚠️ ต้อง mock DB |
| **เรียนรู้ง่าย** | ปานกลาง | ยาก | ยาก | ง่าย | ง่ายมาก |
| **เหมาะกับทีมเล็ก** | ✅ | ⚠️ overhead | ⚠️ overhead | ✅ | ✅ |
| **พร้อม microservice** | ✅ | ✅ | ✅ | ✅ | ❌ |
| **Folder structure** | ตาม module | ตาม layer | ตาม layer | ตาม feature | ตาม layer |
| **ใช้ใน industry** | Fintech, Insurance | Enterprise Java | DDD projects | CQRS/.NET | ทุกที่ |

### 🏆 สรุป: ทำไม Hexagonal ถึงเหมาะกับโปรเจกต์นี้

```
  ┌──────────────────────────────────────────────────────────────────┐
  │                                                                   │
  │  ❌ MVC ธรรมดา                                                    │
  │     → service ผูกกับ DB ตรง เปลี่ยน DB = แก้ทุกที่                │
  │     → test ยาก ต้อง mock DB จริง                                  │
  │                                                                   │
  │  ❌ Clean Architecture                                             │
  │     → 4 layer + กฎซ้อนกันเยอะ — Go ชอบ simplicity                │
  │     → overhead สำหรับทีมเล็ก (ต้องสร้าง Use Case layer แยก)      │
  │                                                                   │
  │  ❌ Onion Architecture                                             │
  │     → คล้าย Clean แต่เน้น DDD — ยังไม่ต้องการ DDD ตอนนี้         │
  │                                                                   │
  │  ⚠️ Vertical Slice                                                │
  │     → ดีสำหรับ CQRS แต่ share logic ระหว่าง feature ยาก          │
  │     → Go community ไม่ค่อยใช้                                     │
  │                                                                   │
  │  ✅ Hexagonal (Ports & Adapters) ← เลือกตัวนี้                    │
  │     → พอดีกับ Go — interface เล็ก, ไม่ over-engineer               │
  │     → แยก layer ชัด แต่ไม่ซ้อนเยอะเท่า Clean                     │
  │     → เปลี่ยน DB / framework / queue ได้จริง                      │
  │     → ยอดนิยมใน fintech + insurance (domain เดียวกับเรา)          │
  │     → Modular Monolith ทำให้ deploy ง่าย ไม่ต้อง manage 20 services│
  │                                                                   │
  └──────────────────────────────────────────────────────────────────┘
```

### 💡 ความจริงที่คนมักสับสน

```
  Hexagonal ≈ Clean ≈ Onion

  ทั้ง 3 ตัวมีแนวคิดเหมือนกัน 80%:
  "Business logic อยู่ตรงกลาง, ไม่ผูกกับ infrastructure"

  ต่างกันแค่:
  ┌────────────┬─────────────────────────────────────────┐
  │ Hexagonal  │ เน้น "Port + Adapter" (in/out)          │
  │ Clean      │ เน้น "4 วงกลม + dependency rule"        │
  │ Onion      │ เน้น "Domain Model ที่แกนกลาง"          │
  └────────────┴─────────────────────────────────────────┘

  โปรเจกต์นี้เรียกว่า Hexagonal แต่จริงๆ ก็ได้ประโยชน์จากทั้ง 3
  เพราะหลักการพื้นฐานเหมือนกัน: "Dependency Inversion"
  (ชั้นในไม่ import ชั้นนอก)
```

---

## ✅ จุดแข็ง (Strengths)

### 1. Architecture — ออกแบบดี พร้อม scale

```
  ┌─────────────────────────────────────────────────┐
  │              Hexagonal Architecture               │
  │                                                   │
  │   handler ──▶ service ──▶ port ──▶ adapter        │
  │   (HTTP)      (logic)    (interface) (DB/API)     │
  │                                                   │
  │   เปลี่ยน DB?       → แก้แค่ adapter              │
  │   เปลี่ยน framework? → แก้แค่ handler              │
  │   เพิ่ม module?      → สร้าง folder ตาม pattern    │
  └─────────────────────────────────────────────────┘
```

| จุดแข็ง | รายละเอียด |
|---------|-----------|
| **Modular Monolith** | แยก module ชัดเจน แต่ deploy binary เดียว — ง่ายต่อ ops, พร้อมแตก microservice |
| **Hexagonal Pattern** | business logic ไม่ผูกกับ framework — เปลี่ยน Fiber เป็น Echo ได้โดยไม่แก้ service |
| **Interface-Driven** | ทุก dependency ผ่าน interface — mock ง่าย, test ง่าย, swap implementation ได้ |
| **Composition Root** | `module.go` wire dependency ที่จุดเดียว — ชัดเจน ไม่กระจาย |

### 2. Infrastructure — เตรียมพร้อมทุก layer

```
  Local (docker-compose)
    │
    ├── PostgreSQL 17 + Redis 7 + Kafka 3.9
    │
    ▼
  Staging / UAT / Production (Kubernetes)
    │
    ├── Kustomize base + overlays (3 environments)
    ├── HPA (auto-scale CPU/Memory)
    ├── PDB (ป้องกัน downtime ตอน node drain)
    ├── Health probes (startup + liveness + readiness)
    └── Multi-stage Dockerfile (non-root, < 30MB)
```

| จุดแข็ง | รายละเอียด |
|---------|-----------|
| **Docker Multi-stage** | binary เล็ก (~30MB), non-root user, health check ในตัว |
| **K8s Kustomize** | base + overlays 3 env (staging/uat/prod) — ไม่ duplicate manifest |
| **Observability ครบ** | OTel → Tempo (traces) + Prometheus (metrics) → Grafana (UI) |
| **CI 7 stages** | Lint → Test → Vuln → Build → Docker → Scan → Notify |

### 3. Security — ออกแบบมาตั้งแต่แรก

| จุดแข็ง | รายละเอียด |
|---------|-----------|
| **JWT + API Key** | dual auth strategy, constant-time comparison ป้องกัน timing attack |
| **Non-root container** | `appuser:10001` ไม่ใช่ root |
| **Trivy scan** | scan Docker image ทุก build |
| **govulncheck** | ตรวจ Go dependency vulnerabilities ใน CI |
| **TLS 1.2+** | MySQL external DB บังคับ minimum TLS 1.2 |
| **SQL injection safe** | parameterized queries ทั้งหมด, MySQL `MultiStatements=false` |

### 4. Developer Experience — ใช้งานง่าย

| จุดแข็ง | รายละเอียด |
|---------|-----------|
| **`run.ps1` / `Makefile`** | คำสั่งเดียวทำทุกอย่าง (`dev`, `test`, `ci`, `migrate`) |
| **Hot-reload (Air)** | แก้ code → auto-restart ไม่ต้อง rebuild |
| **Swagger auto-gen** | เขียน annotation → สร้าง API docs อัตโนมัติ |
| **Discord notification** | ผล CI/coverage ส่ง Discord ทันที |
| **Custom testkit** | Go Generics assertions ไม่พึ่ง testify |
| **เอกสารครบ** | 16+ docs ครอบคลุม architecture, testing, deployment, observability |

### 5. Packages — Reusable & Production-Ready

```
  pkg/
  ├── httpclient   → retry + circuit breaker + tracing
  ├── kafka        → producer + consumer + DLQ (dead letter queue)
  ├── cache        → Redis with interface
  ├── localcache   → Otter L1 + Redis L2 (hybrid)
  ├── retry        → exponential / linear / constant backoff
  ├── otel         → tracing + metrics (vendor-neutral)
  └── log          → zerolog structured logging
```

| จุดแข็ง | รายละเอียด |
|---------|-----------|
| **Circuit Breaker** | ป้องกัน cascading failure เมื่อ downstream ล่ม |
| **L1/L2 Hybrid Cache** | Otter (in-memory) → Redis (shared) — เร็วและ consistent |
| **Dead Letter Queue** | Kafka message ที่ fail ซ้ำไม่หาย — retry หรือ debug ได้ |
| **Retry Strategies** | เลือก backoff ได้ (exponential, linear, constant) |

---

## ⚠️ จุดอ่อน (Weaknesses)

### 1. Coverage ยังต่ำ

```
  ปัจจุบัน                         เป้าหมาย
  ████████░░░░░░░░░░░░  ~29%      ██████████████░░░░░░  70%
```

| ปัญหา | ผลกระทบ | แนวทาง |
|-------|---------|--------|
| Coverage ~29% | ไม่มั่นใจเมื่อ refactor | เพิ่ม test ทีละ module, ตั้ง threshold +5% ต่อ sprint |
| ไม่มี integration test | ไม่ test DB/Kafka จริง | เพิ่ม test suite ที่ใช้ testcontainers |
| ไม่มี E2E test | ไม่ test full flow | เพิ่ม API test (Postman/k6) ใน CI |

### 2. Modules บางตัวยังว่าง

```
  ┌────────────┐  ┌────────────┐  ┌────────────┐
  │ ✅ auth    │  │ ✅ cmi     │  │ ✅ document│
  │  (complete)│  │  (complete)│  │  (complete)│
  └────────────┘  └────────────┘  └────────────┘

  ┌────────────┐  ┌────────────┐  ┌────────────┐
  │ ⏳ job     │  │ ⏳ policy  │  │ ⏳ payment │
  │ (placeholder) │ (future)   │  │  (future)  │
  └────────────┘  └────────────┘  └────────────┘
```

| ปัญหา | ผลกระทบ | แนวทาง |
|-------|---------|--------|
| `policy/`, `payment/` ว่าง | feature ไม่ครบ | ค่อย implement เมื่อ business requirement ชัด |
| `job/` เป็น placeholder | ยังไม่มี job processing จริง | เพิ่มเมื่อมี use case |

### 3. Kafka ยังไม่มี Schema Validation

| ปัญหา | ผลกระทบ | แนวทาง |
|-------|---------|--------|
| ไม่มี schema registry | message format อาจ break | เพิ่ม Avro/Protobuf + schema registry |
| ไม่มี message versioning | upgrade ยาก | ใส่ version field ใน event payload |

### 4. Database มี Single Point

| ปัญหา | ผลกระทบ | แนวทาง |
|-------|---------|--------|
| Read/Write ใช้ pool เดียวกัน | ไม่มี read-replica | เพิ่ม read-replica เมื่อ traffic สูง |
| ไม่มี connection failover | DB ล่ม = app ล่ม | เพิ่ม PgBouncer หรือ pgpool-II |
| ยังไม่มี DB backup strategy | data loss risk | เพิ่ม pg_dump cron หรือ WAL archiving |

### 5. Configuration / Secret Management

| ปัญหา | ผลกระทบ | แนวทาง |
|-------|---------|--------|
| Secret ใน `.env` file | ไม่ปลอดภัยสำหรับ production | ใช้ Vault / External Secrets Operator |
| ไม่มี config validation | ค่าผิดรู้ตอน runtime | เพิ่ม validation ตอน startup |

---

## 📈 สรุปเทียบ Scorecard

```
  Architecture    ██████████████████░░  9/10   ← ออกแบบดีมาก
  Security        ████████████████░░░░  8/10   ← ครบ แต่ยังไม่มี rate limit ในระดับ app
  DX (Dev Exp.)   ████████████████████  10/10  ← run.ps1, docs, testkit ครบ
  Observability   ████████████████░░░░  8/10   ← OTel + Grafana ครบ, ยังไม่มี alerting rules
  Testing         ██████████░░░░░░░░░░  5/10   ← coverage ต่ำ, ไม่มี integration test
  Infra/Deploy    ████████████████░░░░  8/10   ← K8s + Docker ครบ, ยังไม่ deploy จริง
  Documentation   ████████████████████  10/10  ← 16+ docs, AI instructions ครบ
  Production-Ready████████████░░░░░░░░  6/10   ← ต้องเพิ่ม test, alerting, secret mgmt
```

### Overall: **8/10** — โครงสร้างดีมาก พร้อมสำหรับ production หลังเพิ่ม test coverage + secret management

---

## 🗺️ Roadmap แนะนำ (Priority Order)

```
  NOW                    NEXT                     LATER
  ─────────────────────  ─────────────────────    ─────────────────
  ✅ Coverage → 50%      ⬜ Integration tests     ⬜ Schema registry
  ✅ Secret management   ⬜ Alerting rules        ⬜ Read-replica
  ✅ Config validation   ⬜ Rate limiting (app)   ⬜ DB failover
  ✅ Deploy staging      ⬜ E2E test suite        ⬜ Blue-green deploy
```

---

> 📝 เอกสารนี้สรุปจากการ review codebase ทั้งหมด — April 2026
