---
description: "Use when writing or reviewing Go code. Covers import ordering, naming conventions, error handling patterns, enums, and code style rules specific to this project."
applyTo: "**/*.go"
---

# Go Conventions — สิ่งที่ต้องระวัง

## 1. Import Order (3 กลุ่ม แยกบรรทัดว่าง)

```go
import (
    // กลุ่ม 1: Standard library
    "context"
    "errors"
    "fmt"

    // กลุ่ม 2: External packages
    "github.com/gofiber/fiber/v2"
    "github.com/jackc/pgx/v5"

    // กลุ่ม 3: Internal packages
    "github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/domain"
    "github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
)
```

- ห้ามปนกลุ่ม — ต้องมีบรรทัดว่างคั่นทุกกลุ่ม
- Alias ใช้เมื่อ pkg/ packages ชื่อชนกัน: `appOtel "...pkg/otel"`

## 2. Naming

| สิ่งที่ตั้งชื่อ | Convention | ตัวอย่าง |
|----------------|-----------|---------|
| Types/Structs | PascalCase | `Service`, `Handler`, `CMIPolicyRepository` |
| Interfaces | PascalCase + suffix ตรง | `UserRepository`, `TokenSigner` |
| Functions | PascalCase, verb-noun | `FindByUsername()`, `VerifyAccessToken()` |
| Package | lowercase, สั้น | `app`, `domain`, `ports`, `postgres` |
| Constants (trace) | kebab-case string | `"cmi-job-not-found"` |
| Variables | camelCase | `userID`, `tokenString` |
| Receivers | ตัวอักษรเดียว | `(r *Repository)`, `(s *Service)`, `(h *Handler)` |
| Fakes (test) | `fake` prefix | `fakeUserRepo`, `fakeRow` |

## 3. Error Handling

### ห้ามทำ

```go
// ❌ if-else
if err != nil {
    return nil, err
} else {
    return result, nil
}

// ❌ string matching
if err.Error() == "not found" { ... }

// ❌ สร้าง error ใน handler
errors.New("something went wrong")  // ใน handler ให้ใช้ dto.ErrorWithTrace
```

### ต้องทำ

```go
// ✅ early return
if err != nil {
    return nil, err
}
return result, nil

// ✅ sentinel errors ใน app/
var ErrInvalidCredentials = errors.New("invalid credentials")

// ✅ errors.Is()
if errors.Is(err, app.ErrInvalidCredentials) {
    return dto.ErrorWithTrace(c, fiber.StatusUnauthorized, "invalid credentials", dto.TraceAuthBadLogin)
}

// ✅ wrap (เฉพาะเมื่อเพิ่ม context)
return fmt.Errorf("redis ping failed: %w", err)
```

## 4. Enum — String Constants เท่านั้น

```go
// ✅ ถูก
const (
    ResponseOK    = "OK"
    ResponseError = "ERROR"
    RoleAdmin     = "admin"
)

// ❌ ห้ามใช้ iota
const (
    RoleAdmin = iota  // ห้าม!
    RoleUser
)
```

## 5. HTTP Response — ใช้ dto helpers เท่านั้น

```go
// ✅ Success
return dto.Success(c, fiber.StatusOK, result)

// ✅ Error พร้อม trace
return dto.ErrorWithTrace(c, fiber.StatusNotFound, "job not found", dto.TraceCMIJobNotFound)

// ❌ ห้าม c.JSON() ตรง
return c.Status(200).JSON(map[string]any{...})
```

## 6. Trace ID — ประกาศใน `internal/shared/dto/error_codes.go` เท่านั้น

- เพิ่ม trace ID ใหม่ต้องไปเพิ่มที่ `error_codes.go` (single source of truth)
- Format: `"module-description"` เช่น `"cmi-job-not-found"`, `"auth-token-invalid"`
- ห้ามสร้าง trace string ใน handler โดยตรง

## 7. Logging — zerolog

```go
import "github.com/onizukazaza/anc-portal-be-fake/pkg/log"

log.L().Info().Str("job_id", jobID).Msg("processing job")
log.L().Error().Err(err).Msg("failed to fetch user")
```

- ห้ามใช้ `fmt.Println`, `log.Print` จาก standard library
- ใช้ structured fields (`.Str()`, `.Int()`, `.Err()`) ไม่ใช่ `Msgf()`

## 8. Context

- ทุก method ที่เป็น I/O (DB, cache, HTTP, Kafka) ต้องรับ `context.Context` เป็น parameter แรก
- Handler ใช้ `c.UserContext()` ไม่ใช่ `context.Background()`
- Test ใช้ `context.Background()` ได้
