# Database Concept — Internal & External

## ภาพรวม

ระบบแบ่ง database ออกเป็น 2 ประเภท:

| ประเภท | หน้าที่ | Driver | ตัวอย่าง |
|--------|---------|--------|----------|
| **Internal (Main)** | เก็บข้อมูลหลักของระบบ (users, jobs, sessions) | postgres เท่านั้น | `deps.DB.Main()` |
| **External** | เชื่อมต่อ database ภายนอก (ระบบเดิม, partner) | postgres หรือ mysql | `deps.DB.External("meprakun")` |

---

## โครงสร้างไฟล์

```
internal/database/
├── provider.go          # Provider interface — สัญญาที่ทุก module ใช้
├── conn.go              # ExternalConn interface + type-safe helpers
├── manager.go           # Manager — สร้าง connection ทั้ง main + external
├── postgres/
│   └── connect.go       # PostgreSQL connector (pgxpool)
└── mysql/
    └── connect.go       # MySQL connector (database/sql)
```

---

## แนวคิดหลัก

### 1. Provider Interface — สัญญาระหว่าง database กับ module

```go
type Provider interface {
    Main() *pgxpool.Pool                       // internal database
    External(name string) (ExternalConn, error) // external database by name
    Read() *pgxpool.Pool                       // read pool (= Main)
    Write() *pgxpool.Pool                      // write pool (= Main)
    HealthCheck(ctx context.Context) error
    Close()
}
```

- Module ขึ้นตรงกับ `Provider` interface เท่านั้น ไม่ import `Manager` โดยตรง
- ทำให้ test ง่าย — mock `Provider` ได้เลย

### 2. ExternalConn Interface — driver-agnostic

```go
type ExternalConn interface {
    Health(ctx context.Context) error
    Close()
    Driver() string                                                // "postgres" | "mysql"
    Diagnostic(ctx context.Context) (dbName, version string, err error)
}
```

- ทุก driver (postgres, mysql) implement interface นี้
- Module ไม่ต้องรู้ว่าข้างหลังเป็น driver อะไร

### 3. Type-safe Helpers — แปลง ExternalConn เป็น concrete type

```go
pool, err := database.PgxPool(conn)   // → *pgxpool.Pool (postgres)
db, err   := database.SQLDB(conn)     // → *sql.DB       (mysql)
```

- ใช้แทน type assertion ตรง ๆ → ปลอดภัยกว่า, มี error message ชัดเจน

### 4. Manager — ศูนย์กลางการ connect

```
NewManager(ctx, cfg)
  ├── connect Main (postgres)
  └── for each ExternalDB in config
        └── connectExternal(driver)
              ├── "postgres" → postgres.NewWithConfig()
              └── "mysql"    → mysql.NewWithConfig()
```

- สร้างครั้งเดียวตอน server start
- ส่งผ่าน `module.Deps` ไปทุก module

---

## กฎการใช้งาน

### ✅ ต้องทำ

| กฎ | เหตุผล |
|----|--------|
| ใช้ `deps.DB.Main()` สำหรับข้อมูลภายใน | Main เป็น postgres pool พร้อมใช้ |
| ใช้ `deps.DB.External("name")` แล้วแปลงด้วย `database.PgxPool()` หรือ `database.SQLDB()` | type-safe, มี error handling |
| ตรวจ error ทุกครั้งที่เรียก `External()` | อาจไม่มี config สำหรับชื่อนั้น |
| ส่ง pool/db ให้ repository — ไม่ส่ง `ExternalConn` ตรง ๆ | repository ต้องการ concrete type |
| ใช้ `context.Context` ทุก query | รองรับ timeout + tracing |

### ❌ ห้ามทำ

| กฎ | เหตุผล |
|----|--------|
| ห้าม type assert `ExternalConn` เอง เช่น `conn.(*postgres.DB)` | fragile, driver อาจเปลี่ยน |
| ห้าม import `internal/database/postgres` หรือ `mysql` ใน module | module ต้องผ่าน interface เท่านั้น |
| ห้ามสร้าง connection ใหม่เองใน module | ใช้ pool ที่ Manager สร้างไว้ |
| ห้าม hardcode DSN ใน code | ทุกค่าต้องมาจาก config/env |
| ห้ามเปิด `MultiStatements=true` (MySQL) | ป้องกัน SQL injection via stacked queries |

---

## ตัวอย่างการใช้งานใน Module

### Internal Database (auth module)

```go
func Register(router fiber.Router, deps module.Deps, tokenSigner ports.TokenSigner) {
    // >> ใช้ Main() ตรง ๆ — ได้ *pgxpool.Pool
    userRepo := postgres.NewUserRepository(deps.DB.Main())
    service  := app.NewService(userRepo, tokenSigner)
    // ...
}
```

### External Database — Postgres (cmi module)

```go
func Register(router fiber.Router, deps module.Deps) {
    // >> ดึง ExternalConn
    conn, err := deps.DB.External("meprakun")
    if err != nil {
        return // ไม่มี config → skip module
    }

    // >> แปลงเป็น *pgxpool.Pool ด้วย helper
    pool, err := database.PgxPool(conn)
    if err != nil {
        return
    }

    repo := cmipg.NewCMIPolicyRepository(pool)
    // ...
}
```

### External Database — MySQL (ตัวอย่าง)

```go
func Register(router fiber.Router, deps module.Deps) {
    conn, err := deps.DB.External("legacy-system")
    if err != nil {
        return
    }

    // >> แปลงเป็น *sql.DB ด้วย helper
    db, err := database.SQLDB(conn)
    if err != nil {
        return
    }

    repo := mysqlrepo.NewOrderRepository(db)
    // ...
}
```

---

## การตั้งค่า Environment

### Internal (Main) Database

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=admin
DB_PASSWORD=secret
DB_DBNAME=anc_portal
DB_SSLMODE=disable
DB_SCHEMA=public
DB_MAX_CONNS=10
DB_MIN_CONNS=2
DB_MAX_CONN_LIFETIME=30m
DB_MAX_CONN_IDLE_TIME=5m
```

### External Databases

```env
# ลงทะเบียนชื่อ (comma-separated)
EXTERNAL_DBS=meprakun,legacy

# แต่ละตัวใช้ prefix: EXTERNAL_DBS_{NAME}_
EXTERNAL_DBS_MEPRAKUN_DRIVER=postgres
EXTERNAL_DBS_MEPRAKUN_HOST=10.0.1.50
EXTERNAL_DBS_MEPRAKUN_PORT=5432
EXTERNAL_DBS_MEPRAKUN_USER=readonly
EXTERNAL_DBS_MEPRAKUN_PASSWORD=secret
EXTERNAL_DBS_MEPRAKUN_DBNAME=meprakun_db
EXTERNAL_DBS_MEPRAKUN_SSLMODE=disable
EXTERNAL_DBS_MEPRAKUN_SCHEMA=public
EXTERNAL_DBS_MEPRAKUN_MAX_CONNS=5
EXTERNAL_DBS_MEPRAKUN_MIN_CONNS=1
EXTERNAL_DBS_MEPRAKUN_MAX_CONN_LIFETIME=30m
EXTERNAL_DBS_MEPRAKUN_MAX_CONN_IDLE_TIME=5m

EXTERNAL_DBS_LEGACY_DRIVER=mysql
EXTERNAL_DBS_LEGACY_HOST=10.0.2.100
EXTERNAL_DBS_LEGACY_PORT=3306
# ... (same pattern)
```

> `DRIVER` ไม่ใส่ = default เป็น `postgres`

---

## Data Flow Diagram

```
┌─────────────┐
│   Module     │  deps.DB.Main()         → *pgxpool.Pool → Repository
│  (auth, job) │
└──────────────┘

┌─────────────┐
│   Module     │  deps.DB.External("x")  → ExternalConn
│ (cmi, qt)   │        │
└──────────────┘        ├── database.PgxPool(conn)  → *pgxpool.Pool → PG Repository
                        └── database.SQLDB(conn)    → *sql.DB       → MySQL Repository

┌─────────────┐
│ externaldb   │  deps.DB.External("x")  → ExternalConn.Diagnostic() → health check
│  (admin)    │
└──────────────┘
```

---

## Health Check

`/v1/external-db/health` เรียก `ExternalConn.Diagnostic()` ของทุกตัว:

```json
{
  "data": [
    {
      "name": "meprakun",
      "driver": "postgres",
      "status": "healthy",
      "database": "meprakun_db",
      "version": "PostgreSQL 15.4"
    }
  ]
}
```

---

## สรุป

1. **Main** = internal postgres — ใช้ `deps.DB.Main()` ตรง ๆ
2. **External** = multi-driver — ใช้ `deps.DB.External(name)` + helper แปลง type
3. **Module ไม่รู้จัก driver** — ผ่าน interface หมด
4. **Manager สร้างครั้งเดียว** — ส่งผ่าน `Deps` ไปทุก module
5. **Config จาก env** — ห้าม hardcode connection string
