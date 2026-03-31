---
description: "Use when writing database queries, repositories, migrations, or working with multi-driver database connections. Covers pgx patterns, ExternalConn usage, SQL conventions, and connection pool rules."
applyTo: ["internal/database/**", "internal/modules/**/adapters/postgres/**", "migrations/**"]
---

# Database — สิ่งที่ต้องระวัง

## 1. Multi-Driver — ห้าม Raw Type Assertion

```go
// ❌ ห้าม
pool := conn.(*pgxConn).pool

// ✅ ใช้ type-safe helpers
pool, err := database.PgxPool(conn)    // สำหรับ postgres
db, err := database.SQLDB(conn)        // สำหรับ mysql
```

- `database.PgxPool()` / `database.SQLDB()` มี nil guard ป้องกัน panic
- ตรวจ driver ก่อนใช้: `conn.Driver()` returns `"postgres"` or `"mysql"`

## 2. Repository Pattern

```go
type Repository struct {
    pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
    return &Repository{pool: pool}
}

func (r *Repository) FindByID(ctx context.Context, id string) (*domain.Item, error) {
    // 1. Start OTel span
    ctx, span := appOtel.Tracer(appOtel.TracerXxxRepo).Start(ctx, "FindByID")
    defer span.End()

    // 2. SQL เป็น const ในระดับ method
    const q = `SELECT id, name, created_at FROM items WHERE id = $1`

    // 3. Query + Scan
    var item domain.Item
    err := r.pool.QueryRow(ctx, q, id).Scan(&item.ID, &item.Name, &item.CreatedAt)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, nil  // ไม่เจอ → return nil, nil (ไม่ใช่ error)
    }
    if err != nil {
        return nil, err  // error อื่น → propagate ตรงๆ
    }
    return &item, nil
}
```

### กฎสำคัญ

- **Receiver**: ตัวอักษรเดียว `(r *Repository)`
- **OTel span**: ทุก public method ต้องมี span
- **SQL const**: ประกาศ const ที่ method level ไม่ใช่ package level
- **No rows**: return `nil, nil` (handler เช็ค nil → 404)
- **Error**: return ตรงๆ — handler จะ map เป็น HTTP status
- **Parameterized query**: ใช้ `$1, $2, ...` เสมอ — ห้าม string concatenation

## 3. SQL Conventions

```sql
-- ✅ ใช้ parameterized queries
SELECT * FROM users WHERE id = $1 AND status = $2

-- ❌ ห้าม string interpolation
SELECT * FROM users WHERE id = '" + id + "'   -- SQL Injection!
```

- ใช้ `$1, $2, ...` สำหรับ pgx (PostgreSQL)
- ใช้ `?, ?, ...` สำหรับ MySQL (`database/sql`)
- Query ยาว (>50 บรรทัด) → แยกเป็น SQL fragment functions

### SQL Fragment Pattern

```go
func sqlSelectFields() string { return `j.id, j.name, j.status` }
func sqlFromJoins() string    { return `FROM job j LEFT JOIN ...` }

func buildQuery() string {
    return fmt.Sprintf("SELECT %s %s WHERE j.id = $1",
        sqlSelectFields(), sqlFromJoins())
}
```

## 4. External Database Connection

```go
// ดึง external connection
extConn, err := deps.DB.External("partner_a")
if err != nil {
    return err
}

// ตรวจ driver แล้วใช้ type-safe helper
switch extConn.Driver() {
case "postgres":
    pool, err := database.PgxPool(extConn)
case "mysql":
    db, err := database.SQLDB(extConn)
}
```

## 5. Scan Complex Objects

```go
func scanItem(row pgx.Row) (*domain.Item, error) {
    var (
        item      domain.Item
        motorJSON []byte
        agentJSON []byte
    )

    err := row.Scan(
        &item.ID,
        &item.Name,
        &motorJSON,    // JSON column → []byte
        &agentJSON,
        &item.CreatedAt,
    )
    if err != nil {
        return nil, err
    }

    // unmarshal JSON fields
    if err := unmarshalIfNotNil(motorJSON, &item.Motor); err != nil {
        return nil, fmt.Errorf("unmarshal motor: %w", err)
    }

    return &item, nil
}

func unmarshalIfNotNil(data []byte, dest any) error {
    if len(data) == 0 {
        return nil
    }
    return json.Unmarshal(data, dest)
}
```

### กฎ Scan

- JSON columns → scan เป็น `[]byte` แล้ว unmarshal ทีหลัง
- Nullable columns → ใช้ pointer types (`*int`, `*string`)
- ต้อง handle nil JSON gracefully (ห้าม panic)
- Wrap unmarshal error ด้วย field name: `fmt.Errorf("unmarshal motor: %w", err)`

## 6. Migration Files

```
migrations/
├── 000001_create_users_table.up.sql
├── 000001_create_users_table.down.sql
├── 000002_create_insurer_tables.up.sql
└── 000002_create_insurer_tables.down.sql
```

- Format: `{sequence}_{description}.{up|down}.sql`
- ทุก up ต้องมี down คู่กัน (reversible)
- ห้ามแก้ไข migration ที่ apply ไปแล้ว — สร้างใหม่เสมอ
