# AI_CONTEXT.md

## Purpose

This file defines the working rules for AI assistants and contributors in this project.

The goal is to keep changes consistent with the intended architecture:

* feature-first structure
* clear separation of responsibility inside each feature
* easy testing
* infrastructure can change without breaking business logic
* changes remain local to the related feature whenever possible

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
* Architecture style: Clean Architecture + feature-first
* Main flow inside a feature: `Handler -> Service -> Ports -> Repository/Integration`

Design intent:

* each business feature owns its own layers
* handler handles transport concerns
* service contains orchestration and business rules
* ports define contracts required by service
* repository/integration are adapter implementations inside the feature
* business logic should remain independent from infrastructure details
* cross-feature dependency should be kept minimal and explicit

---

## Project Structure

Example:

```text
internal/
  app/
  config/
  domain/
  job/
    handler/
      http/
        handler.go
        dto.go
        mapper.go
    service/
      service.go
      create_job.go
      create_job_flow.go
      get_job.go
      list_jobs.go
      testkit_test.go
      create_job_test.go
    ports/
      outbound.go
    repository/
      postgres/
        repository.go
        dto.go
        mapping.go
        repository_unit_test.go
        repository_integration_test.go
    integration/
      line/
        client.go
      email/
        sender.go

  user/
    handler/
      http/
        handler.go
    service/
      service.go
      create_user.go
      get_user.go
    ports/
      outbound.go
    repository/
      postgres/
        repository.go
    integration/
      cloudtask/
        client.go
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
* integration test guides
* conceptual references

### internal/app

* composition root
* system wiring
* dependency initialization across features

### internal/app/bootstrap.go

* wire all dependencies
* assemble feature handlers, services, repositories, integrations

### internal/app/connectors.go

* initialize DB/cache/external connectors shared by the application

### internal/config

* load environment and app config

### internal/domain

* core business entities and value objects shared across the system
* should remain framework/infrastructure independent

### internal/{feature}/handler

* transport entrypoints for that feature
* HTTP handlers
* request/response DTOs
* request validation at transport level
* response formatting
* mapping between transport DTOs and service input/output

### internal/{feature}/service

* orchestration
* business rules
* application flow coordination for that feature
* should not contain SQL, HTTP framework details, or SDK-specific details

### internal/{feature}/ports

* contracts required by the service layer
* define required outbound behavior before implementation

### internal/{feature}/repository

* DB/cache data access implementation for that feature
* persistence concerns only
* may contain DB DTOs and mapping

### internal/{feature}/integration

* external service adapters for that feature
* examples: LINE, email, cloud task, file move
* should not contain core business rules

### migrations

* SQL schema and migration files

### pkg/db

* shared DB connection helpers

---

## Dependency Rules

These rules are strict unless the user explicitly asks otherwise.

* Handler may call Service only.
* Handler must not call Repository directly.
* Service may depend on Domain and Ports.
* Service must not depend on concrete infrastructure adapters directly.
* Service must not contain SQL, HTTP framework details, or external service SDK details.
* Repository may depend on DB driver/query tools and Domain mapping.
* Repository must not contain heavy business rules.
* Integration adapters must not contain core business rules.
* Domain should not import handler, repository, or integration layers.
* Ports must be defined before implementing adapters when introducing a new dependency.
* Cross-feature calls should go through a clear contract/interface, not direct tight coupling by default.
* Avoid one feature reaching into another feature’s repository directly unless explicitly justified and kept minimal.

---

## Feature-First File Pattern

Example: Job feature

### `internal/job/handler/http`

* `handler.go`

  * HTTP entrypoints
* `dto.go`

  * request/response DTOs
* `mapper.go`

  * transport mapping

### `internal/job/service`

* `service.go`

  * facade
  * constructor
  * dependency holder
* `create_job.go`

  * public usecase entrypoint for `CreateJob`
* `create_job_flow.go`

  * detailed orchestration flow for `CreateJob`
* `get_job.go`

  * read capability logic
* `list_jobs.go`

  * list/search capability logic
* `testkit_test.go`

  * shared fake/test helpers
* `create_job_test.go`

  * behavior tests for `CreateJob`
* `get_job_test.go`

  * behavior tests for `GetJob`

### `internal/job/ports`

* `outbound.go`

  * contracts needed by job service

### `internal/job/repository/postgres`

* `repository.go`

  * concrete SQL access
* `dto.go`

  * DB-facing DTOs
* `mapping.go`

  * DTO <-> Domain mapping
* `mapping_test.go`

  * unit tests for mapping
* `repository_unit_test.go`

  * unit tests using sqlmock
* `repository_integration_test.go`

  * integration tests with real PostgreSQL

### `internal/job/integration`

* feature-owned external adapters
* examples:

  * `line/client.go`
  * `email/sender.go`
  * `cloudtask/client.go`

---

## Structure Principles

* Organize by feature first, then by layer inside the feature.
* Keep related code close together.
* One feature should be understandable without jumping across many top-level folders.
* `service.go` should stay thin.
* Long or multi-step flows should move to `*_flow.go`.
* Split by capability, not by technical keyword only.
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

* Test behavior, not implementation detail.
* Include both success path and failure path.
* Tests must be deterministic.
* Cleanup must be explicit.
* Never use production DB in integration tests.
* Credentials must come from environment, never hardcoded.

### Service Tests

* use fake/testkit helpers
* focus on input/output and orchestration behavior
* avoid coupling tests to internal private helper structure unless necessary

### Repository Unit Tests

* use `sqlmock`
* verify query behavior, args, scan/mapping path, and error path

### Repository Integration Tests

* run against a real PostgreSQL test database
* validate schema compatibility and actual SQL behavior
* use env from `.env.test` / `.env`
* cleanup with `TRUNCATE` or equivalent deterministic reset

---

## Change Rules for AI

When modifying code:

* preserve existing behavior unless the user explicitly requests a behavior change
* avoid unnecessary renames, moves, or abstraction
* do not introduce new frameworks or patterns unless requested
* prefer extending the current feature structure over creating parallel patterns
* if adding a new file, make sure its responsibility is obvious from the name
* if unsure where logic belongs:

  * business rule -> Service
  * transport concern -> Handler
  * persistence concern -> Repository
  * external provider concern -> Integration
  * contract/interface -> Ports

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

* Check the feature’s `ports` first for required contracts.
* Check `internal/domain` for business shape and language.
* Check neighboring files in the same feature before creating new patterns.
* Prefer consistency with the existing feature over theoretical purity.
* If ambiguity remains, state your assumption explicitly and proceed with the safest minimal implementation.