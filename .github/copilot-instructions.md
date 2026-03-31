# ANC Portal Backend — AI Code Generation Rules

> ⚠️ ไฟล์นี้เป็น **workspace instructions** สำหรับ AI ทุกตัวที่ generate code ในโปรเจกต์นี้
> อ่านก่อน generate ทุกครั้ง

## สิ่งที่ห้ามทำ (Critical)

1. **ห้ามเพิ่ม external test dependency** — ไม่มี testify, gomock, mockery, ginkgo ใช้ `internal/testkit` เท่านั้น
2. **ห้าม import ข้าม layer** — domain ห้าม import ports, handler ห้าม import domain ตรง ต้องผ่าน service
3. **ห้าม raw type assertion บน ExternalConn** — ใช้ `database.PgxPool()` / `database.SQLDB()` เท่านั้น
4. **ห้ามสร้าง error ด้วย string matching** — ใช้ sentinel errors + `errors.Is()`
5. **ห้าม if-else สำหรับ error** — ใช้ early return เสมอ
6. **ห้ามใช้ `iota` สำหรับ enum** — ใช้ string constants ตรงๆ

## Architecture — Hexagonal (Ports & Adapters)

```
handler → service → port (interface) → domain
```

- `domain/` = pure structs, ไม่ import อะไรเลย
- `ports/` = Go interface เท่านั้น
- `app/` = business logic, inject interface ผ่าน constructor
- `adapters/http/` = Fiber handler
- `adapters/postgres/` = pgx repository
- `module.go` = wiring (composition root ของ module)

## สิ่งที่ต้องระวัง

รายละเอียดแยกตามหัวข้อใน `.github/instructions/`:

- [go-conventions.instructions.md](.github/instructions/go-conventions.instructions.md) — import order, naming, error handling
- [testing.instructions.md](.github/instructions/testing.instructions.md) — testkit, fakes, handler/repo test patterns
- [architecture.instructions.md](.github/instructions/architecture.instructions.md) — hexagonal rules, module registration
- [database.instructions.md](.github/instructions/database.instructions.md) — multi-driver, repository, SQL patterns
