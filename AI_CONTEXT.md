# AI_CONTEXT.md

## Purpose

This file defines the working rules for AI assistants and contributors in this project.

The goal is to keep changes consistent with the intended architecture:

* module-first structure (`internal/modules/{module}/`)
* hexagonal architecture: `domain/` → `ports/` → `app/` → `adapters/`
* clear separation of responsibility inside each module
* easy testing with hand-written fakes (no external test deps)
* infrastructure can change without breaking business logic
* changes remain local to the related module whenever possible

---

## Read First / Source of Truth

When working on this project, use these as the source of truth in order:

1. Existing code in the target feature/module
2. This `AI_CONTEXT.md`
3. Related documents under `/docs`
4. Explicit instructions from the current user prompt

If there is a conflict, do not guess silently.
State the conflict, then proceed with the safest minimal change.

---

## Architecture Overview

* Language: Go 1.25
* Architecture style: Modular Monolith + Hexagonal (Ports & Adapters), feature-first
* Main flow inside a module: `Handler (adapters/http) → Service (app) → Port (interface) → Repository/Integration (adapters)`

Design intent:

* each business module owns its own layers inside `internal/modules/{module}/`
* `adapters/http/` handles transport concerns (Fiber handlers)
* `app/` contains orchestration and business rules (service layer)
* `ports/` define contracts required by the service
* `adapters/postgres/` are repository implementations
* `adapters/external/` are non-DB adapter implementations (e.g. JWT signer)
* `domain/` contains pure structs (no imports)
* `module.go` is the composition root — wires dependencies
* business logic should remain independent from infrastructure details
* cross-module dependency should be kept minimal and explicit

---

## Project Structure

Actual structure:

```text
internal/
  database/          # Multi-driver DB layer (postgres, mysql)
    provider.go      # Interface — module calls this
    conn.go          # ExternalConn interface + type-safe helpers
    manager.go       # Orchestrator (composition root for DB)
    postgres/        # PostgreSQL driver
    mysql/           # MySQL driver
    seed/            # Seed data logic

  modules/
    auth/
      module.go      # Composition root — wires deps
      domain/
        auth.go      # Pure domain structs
      ports/
        token_signer.go
        user_repository.go
      app/
        service.go        # Business logic
        service_test.go   # Behavior tests
        fakes_test.go     # Hand-written fakes
      adapters/
        http/
          controller.go   # Route registration
          handler.go      # HTTP handlers + DTOs
        postgres/
          user_repository.go  # SQL implementation
        external/
          jwt_token_signer.go
          simple_token_signer.go

    cmi/
      module.go
      domain/
      ports/
      app/
      adapters/
        http/
          controller.go
          handler.go
          handler_test.go
          fakes_test.go
        postgres/
          repository.go
          repository_test.go

    document/        # Document management
    externaldb/      # External DB health check
    job/             # Job (placeholder)
    notification/    # Notification
    payment/         # Payment (future)
    policy/          # Policy (future)
    quotation/       # Quotation
    webhook/         # GitHub Webhook → Discord

  shared/
    dto/             # Error codes, response
    enum/            # String-based enums (no iota)
    module/          # Shared module deps
    pagination/      # Pagination helpers
    utils/           # ID, JSON, pointer, slice, string utils
    validator/       # Fiber request validation

  testkit/           # Test assertion library (Go Generics, zero deps)
  import/            # CSV importers
  sync/              # Data sync framework

config/              # Viper configuration loader
server/              # Fiber server + routing + middleware
pkg/
  otel/              # OpenTelemetry (tracing + metrics)
  kafka/             # Kafka producer/consumer + DLQ
  cache/             # Redis cache client
  localcache/        # Otter in-memory cache (L1→L2 hybrid)
  httpclient/        # HTTP client + retry + tracing + circuit breaker
  retry/             # Retry strategies
  log/               # zerolog wrapper
  banner/            # Startup banner
  buildinfo/         # Git commit + build time
```

---

## Layer Responsibilities

### cmd/api/main.go

* application entrypoint
* start server
* graceful shutdown

### docs

* architecture notes
* orchestration guides
* conceptual references

### config/

* load environment and app config (Viper)

### internal/database/

* Multi-driver database layer (Postgres + MySQL)
* `provider.go` — Interface (contract that modules call)
* `conn.go` — ExternalConn interface + type-safe helpers (`PgxPool()`, `SQLDB()`)
* `manager.go` — Composition root: switch-case driver, connect, lifecycle
* Modules use interface only, never import driver directly

### internal/domain/ (per module)

* core business entities and value objects
* should remain framework/infrastructure independent
* lives inside each module: `internal/modules/{module}/domain/`

### internal/modules/{module}/adapters/http/

* transport entrypoints for that module
* `controller.go` — route registration
* `handler.go` — HTTP handlers, DTOs, validation, response formatting

### internal/modules/{module}/app/

* orchestration
* business rules
* application flow coordination for that module
* should not contain SQL, HTTP framework details, or SDK-specific details

### internal/modules/{module}/ports/

* contracts required by the service layer
* define required outbound behavior before implementation

### internal/modules/{module}/adapters/postgres/

* DB data access implementation for that module
* persistence concerns only
* may contain DB DTOs and mapping

### internal/modules/{module}/adapters/external/

* non-DB adapter implementations (e.g. JWT signer, API clients)
* should not contain core business rules

### internal/modules/{module}/module.go

* composition root for the module
* wires dependencies (inject interfaces via constructor)

### internal/shared/

* cross-module utilities: DTOs, enums, pagination, validation, utils

### internal/testkit/

* test assertion library using Go Generics
* zero external dependencies

### migrations/

* SQL schema and migration files

---

## Dependency Rules

These rules are strict unless the user explicitly asks otherwise.

* Handler may call Service only (via `app/` layer).
* Handler must not call Repository directly.
* Service may depend on Domain and Ports.
* Service must not depend on concrete infrastructure adapters directly.
* Service must not contain SQL, HTTP framework details, or external service SDK details.
* Repository may depend on DB driver/query tools and Domain mapping.
* Repository must not contain heavy business rules.
* Integration/External adapters must not contain core business rules.
* Domain should not import handler, repository, or integration layers.
* Domain must not import anything outside the module (pure structs only).
* Ports must be defined before implementing adapters when introducing a new dependency.
* Cross-module calls should go through a clear contract/interface, not direct tight coupling by default.
* Avoid one module reaching into another module's repository directly.

---

## Module-First File Pattern

Example: Auth module

### `internal/modules/auth/module.go`

* composition root for the module
* wires dependencies (inject interfaces via constructor)

### `internal/modules/auth/domain/`

* `auth.go` — pure domain structs

### `internal/modules/auth/ports/`

* `token_signer.go` — contract for token signing
* `user_repository.go` — contract for user persistence

### `internal/modules/auth/app/`

* `service.go`

  * facade
  * constructor
  * dependency holder
* `service_test.go`

  * behavior tests
* `fakes_test.go`

  * hand-written fakes for testing

### `internal/modules/auth/adapters/http/`

* `controller.go`

  * route registration
* `handler.go`

  * HTTP handlers + DTOs

### `internal/modules/auth/adapters/postgres/`

* `user_repository.go`

  * concrete SQL access

### `internal/modules/auth/adapters/external/`

* `jwt_token_signer.go` — JWT implementation
* `simple_token_signer.go` — simple implementation for testing

  * behavior tests

---

## Structure Principles

* Organize by module first (`internal/modules/{module}/`), then by layer inside.
* Each module follows hexagonal: `domain/` → `ports/` → `app/` → `adapters/`
* `module.go` is the composition root — wires dependencies for that module.
* Keep related code close together.
* One module should be understandable without jumping across many top-level folders.
* Tests should be grouped by behavior/capability.
* Do not create extra files unless they improve clarity, testability, or separation of responsibility.

---

## Coding Rules

* Do not write SQL in Service.
* Do not place infrastructure details inside Service.
* Do not place heavy business rules in Repository.
* Define interface/contract in the feature’s `ports` before implementing adapter when introducing a new outbound dependency.
* Wrap errors with useful context for debugging.

  * good examples: `create job`, `find job by id`, `list jobs`
* File names must reflect responsibility clearly.
* Prefer one primary intention per file.
* Avoid oversized files with mixed responsibilities.
* When changing a feature, update or add tests for that feature immediately.
* If a flow becomes long or hard to read, extract it into a flow file and name helpers clearly.
* Follow the existing style of the surrounding feature before inventing a new structure.
* Keep changes minimal and local unless broader refactor is explicitly requested.

---

## Testing Rules

### General

* **No external test dependencies** — no testify, gomock, mockery, ginkgo
* Use `internal/testkit` for assertions (Go Generics, zero deps)
* Test behavior, not implementation detail.
* Include both success path and failure path.
* Tests must be deterministic.
* Use hand-written fakes (struct fields for return values), not mocks
* Use `fakes_test.go` in the same package for test helpers

### Service Tests (app/)

* use hand-written fakes implementing port interfaces
* focus on input/output and orchestration behavior
* `fakes_test.go` + `service_test.go` pattern

### Handler Tests (adapters/http/)

* use `setupApp + doRequest` pattern with Fiber + httptest
* test HTTP status codes, response format, trace_id
* `fakes_test.go` + `handler_test.go` pattern

### Repository Tests (adapters/postgres/)

* use `fakeRow` from testkit for pgx.Row scan tests
* verify query behavior, scan/mapping path, and error path
* `repository_test.go` pattern

---

## Change Rules for AI

When modifying code:

* preserve existing behavior unless the user explicitly requests a behavior change
* avoid unnecessary renames, moves, or abstraction
* do not introduce new frameworks or patterns unless requested
* prefer extending the current feature structure over creating parallel patterns
* if adding a new file, make sure its responsibility is obvious from the name
* if unsure where logic belongs:

  * business rule -> Service (app/)
  * transport concern -> Handler (adapters/http/)
  * persistence concern -> Repository (adapters/postgres/)
  * external provider concern -> External Adapter (adapters/external/)
  * contract/interface -> Ports (ports/)

---

## Definition of Done

A task is not complete until:

* code placement matches the intended layer inside the correct feature
* new/changed behavior has tests updated or added
* errors include useful context
* no business logic leaks into repository/integration unnecessarily
* service does not depend on concrete infrastructure directly
* tests remain deterministic
* changes are minimal and readable

---

## When Unsure

* Check the module's `ports/` first for required contracts.
* Check `internal/modules/{module}/domain/` for business shape and language.
* Check neighboring files in the same module before creating new patterns.
* Prefer consistency with the existing module over theoretical purity.
* If ambiguity remains, state your assumption explicitly and proceed with the safest minimal implementation.