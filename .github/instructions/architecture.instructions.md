---
description: "Use when creating new modules, modifying module structure, or wiring dependencies. Covers Hexagonal Architecture rules, module registration, dependency injection, and layer boundaries."
applyTo: "internal/modules/**"
---

# Architecture — สิ่งที่ต้องระวัง

## 1. Hexagonal Architecture — Dependency Direction

```
domain ← ports ← app ← adapters
  (pure)   (interface)  (logic)  (HTTP/DB)
```

### ห้าม import ข้าม layer

| จาก | ไป | ได้ไหม |
|-----|-----|--------|
| `domain/` | `ports/` | ❌ domain ห้าม import อะไรเลย |
| `domain/` | `app/` | ❌ |
| `ports/` | `domain/` | ✅ ports อ้าง domain types ได้ |
| `ports/` | `app/` | ❌ |
| `app/` | `domain/` | ✅ |
| `app/` | `ports/` | ✅ service ใช้ port interfaces |
| `adapters/http/` | `app/` | ✅ handler เรียก service |
| `adapters/http/` | `domain/` | ❌ ต้องผ่าน service |
| `adapters/postgres/` | `domain/` | ✅ repo สร้าง domain structs |
| `adapters/postgres/` | `app/` | ❌ repo ไม่รู้จัก service |

## 2. โครงสร้าง Module

```
internal/modules/{name}/
├── domain/              ← Pure structs, ไม่ import ใดๆ
│   └── {name}.go
├── ports/               ← Go interfaces เท่านั้น (contract)
│   └── repository.go
├── app/                 ← Business logic
│   ├── service.go       ← รับ ports ผ่าน constructor
│   ├── service_test.go
│   └── fakes_test.go
├── adapters/
│   ├── http/            ← Fiber handler
│   │   ├── controller.go
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── fakes_test.go
│   ├── postgres/        ← pgx repository
│   │   ├── repository.go
│   │   └── repository_test.go
│   └── external/        ← External service adapters (JWT, etc.)
└── module.go            ← Composition root (wiring)
```

## 3. Domain Layer

```go
// domain/user.go — Pure struct, ไม่มี import
package domain

type User struct {
    ID           string
    Username     string
    PasswordHash string
    Roles        []string
}
```

### กฎ

- **ห้าม** import package ใดเลย (ยกเว้น standard types)
- **ห้าม** มี method ที่เรียก I/O
- **ห้าม** มี tag json/mapstructure (นั่นเป็นหน้าที่ adapter)

## 4. Ports Layer

```go
// ports/repository.go — Interface เท่านั้น
package ports

type UserRepository interface {
    FindByUsername(ctx context.Context, username string) (*domain.User, error)
}
```

### กฎ

- **ต้อง** เป็น Go interface
- **ต้อง** มี `context.Context` เป็น parameter แรกของทุก method
- **ห้าม** มี implementation ใดๆ
- Interface เล็ก: 1-3 methods (Interface Segregation)

## 5. App Layer (Service)

```go
// app/service.go
type Service struct {
    users  ports.UserRepository   // ← inject interface
    tokens ports.TokenSigner      // ← inject interface
}

func NewService(users ports.UserRepository, tokens ports.TokenSigner) *Service {
    return &Service{users: users, tokens: tokens}
}
```

### กฎ

- **ต้อง** รับ dependency เป็น interface ผ่าน constructor
- **ห้าม** สร้าง concrete type ภายใน service
- Sentinel errors ประกาศที่ `app/` เช่น `var ErrInvalidCredentials = errors.New(...)`

## 6. Adapters Layer

### HTTP Handler

```go
func (h *Handler) GetByID(c *fiber.Ctx) error {
    id := c.Params("id")
    result, err := h.service.GetByID(c.UserContext(), id)
    if errors.Is(err, app.ErrNotFound) {
        return dto.ErrorWithTrace(c, fiber.StatusNotFound, "not found", dto.TraceXxxNotFound)
    }
    if err != nil {
        return dto.ErrorWithTrace(c, fiber.StatusInternalServerError, "internal error", dto.TraceXxxInternalError)
    }
    return dto.Success(c, fiber.StatusOK, result)
}
```

### Repository

```go
func (r *Repository) FindByID(ctx context.Context, id string) (*domain.Item, error) {
    ctx, span := appOtel.Tracer(appOtel.TracerXxxRepo).Start(ctx, "FindByID")
    defer span.End()

    const q = `SELECT id, name FROM items WHERE id = $1`
    // ...
}
```

## 7. Module Registration (module.go)

```go
func Register(router fiber.Router, deps module.Deps) {
    repo := postgres.NewRepository(deps.DB.Main())
    svc := app.NewService(repo)
    ctrl := http.NewController(svc)

    group := router.Group("/items")
    group.Get("/:id", deps.Middleware.Auth, ctrl.GetByID)
}
```

### กฎ

- สร้าง concrete types ที่ `module.go` เท่านั้น (composition root)
- Wire ผ่าน `module.Deps` struct (Config, DB, Cache, Middleware)
- middleware inject ตอน route registration
- ห้ามมี business logic ใน module.go

## 8. Shared Package Rules

| Package | ใช้ได้จาก | หน้าที่ |
|---------|----------|--------|
| `shared/dto/` | adapters/http | ApiResponse, ErrorWithTrace, trace IDs |
| `shared/enum/` | ทุก layer | String constants (ResponseOK, roles) |
| `shared/validator/` | adapters/http | BindAndValidate() |
| `shared/pagination/` | app, adapters | Pagination helpers |
