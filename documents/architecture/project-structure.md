# ANC Portal BE — Project Structure

> **Status:** Architecture Design v1.2  
> **Pattern:** Modular Monolith + Hexagonal Architecture (Ports & Adapters)  
> **Language:** Go 1.25 · Fiber v2 · pgx v5  
> **Last Updated:** 2026-03-31

---

## Table of Contents

- [ANC Portal BE — Project Structure](#anc-portal-be--project-structure)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Architecture Diagram](#architecture-diagram)
  - [Full Directory Tree](#full-directory-tree)
  - [Layer Descriptions](#layer-descriptions)
    - [`cmd/` — Application Entrypoints](#cmd--application-entrypoints)
    - [`config/` — Configuration Management](#config--configuration-management)
    - [`internal/modules/` — Domain Modules](#internalmodules--domain-modules)
    - [`internal/shared/` — Cross-Module Utilities](#internalshared--cross-module-utilities)
    - [`pkg/` — Reusable Libraries](#pkg--reusable-libraries)
    - [`deployments/` — Infrastructure as Code](#deployments--infrastructure-as-code)
  - [Module Structure Pattern](#module-structure-pattern)
  - [Dependency Flow](#dependency-flow)
  - [Key Design Decisions](#key-design-decisions)

---

## Overview

โปรเจกต์นี้ออกแบบเป็น **Modular Monolith** ที่ใช้ **Hexagonal Architecture** (Ports & Adapters)  
แต่ละ module แยก domain อิสระ สามารถ extract ไปเป็น microservice ได้ในอนาคต

```text
┌─────────────────────────────────────────────────────────┐
│                    cmd/ (Entrypoints)                    │
│        api · worker · migrate · seed · sync · import    │
├─────────────────────────────────────────────────────────┤
│  server/          │  config/         │  migrations/      │
│  (Fiber HTTP)     │  (Viper config)  │  (SQL migrations) │
├─────────────────────────────────────────────────────────┤
│              internal/ (Business Logic)                  │
│  ┌──────────────────────────────────────────────────┐   │
│  │  modules/                                         │   │
│  │  ┌─ auth ─┐  ┌─ cmi ──┐  ┌─ quotation ┐         │   │
│  │  │ domain │  │ domain │  │ domain     │  ...     │   │
│  │  │ ports  │  │ ports  │  │ ports      │         │   │
│  │  │ app    │  │ app    │  │ app        │         │   │
│  │  │adapters│  │adapters│  │ adapters   │         │   │
│  │  └────────┘  └────────┘  └────────────┘         │   │
│  ├──────────────────────────────────────────────────┤   │
│  │  shared/ (dto, enum, pagination, utils, module)  │   │
│  │  database/ (multi-driver: postgres + mysql)       │   │
│  │  sync/ (data synchronization)                    │   │
│  │  import/ (CSV data import)                       │   │
│  │  testkit/ (test assertion library)               │   │
│  └──────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────┤
│              pkg/ (Reusable Libraries)                   │
│  cache · httpclient · kafka · localcache · log          │
│  otel · retry · banner · buildinfo                      │
├─────────────────────────────────────────────────────────┤
│              deployments/ (Infrastructure)               │
│  docker · k8s (kustomize) · local · observability       │
└─────────────────────────────────────────────────────────┘
```

---

## Architecture Diagram

```text
                              ┌─────────────┐
                              │   Client     │
                              │  (Browser/   │
                              │   Mobile)    │
                              └──────┬───────┘
                                     │ HTTPS
                              ┌──────▼───────┐
                              │   Ingress    │
                              │   (NGINX)    │
                              └──────┬───────┘
                                     │
           ┌─────────────────────────▼─────────────────────────┐
           │                  Fiber HTTP Server                 │
           │  middleware: recover → requestid → access_log       │
           │              → compress → otel → cors → ratelimit │
           ├───────────────────────────────────────────────────┤
           │  /healthz    /ready    /metrics    /swagger/*     │
           │  /v1/auth/*  /v1/cmi/* /v1/quotation/*  ...      │
           └───────┬──────────┬──────────┬─────────────────────┘
                   │          │          │
        ┌──────────▼──┐ ┌────▼─────┐ ┌──▼──────────┐
        │ auth module │ │cmi module│ │quotation mod │ ...
        │  (handler)  │ │(handler) │ │  (handler)   │
        │  (service)  │ │(service) │ │  (service)   │
        │  (repo)     │ │(repo)    │ │  (repo)      │
        └──────┬──────┘ └────┬─────┘ └──────┬───────┘
               │             │              │
        ┌──────▼─────────────▼──────────────▼───────┐
        │     Database Layer (Multi-Driver)          │
        │  main DB (anc-portal) ← postgres           │
        │  external DB (meprakun) ← postgres/mysql   │
        └────────────────────────────────────────────┘
               │              │             │
        ┌──────▼──┐    ┌─────▼────┐  ┌─────▼─────┐
        │  Redis  │    │  Kafka   │  │   OTel    │
        │ (cache) │    │ (events) │  │ (traces)  │
        └─────────┘    └──────────┘  └───────────┘
```

---

## Full Directory Tree

```text
anc-portal-be/
│
│── .github/                           # ─── GitHub Automation ───
│   ├── dependabot.yml                 #   dependency auto-update (Go, Docker, Actions)
│   ├── release.yml                    #   PR categorization for release notes
│   └── workflows/
│       ├── ci.yml                     #   CI pipeline: lint → test → vuln → build → docker → scan → notify
│       ├── deploy-staging.yml         #   CD: auto-deploy to staging on develop push
│       ├── deploy-production.yml      #   CD: manual-approval deploy on v* tag
│       └── release.yml               #   auto-create GitHub Release with changelog
│
├── cmd/                               # ─── Application Entrypoints ───
│   ├── api/                           #   HTTP API server (Fiber)
│   │   └── main.go                    #     bootstrap: config → DB → cache → kafka → otel → server
│   ├── worker/                        #   Kafka consumer worker
│   │   └── main.go                    #     bootstrap: config → DB → kafka consumer → event router → health probe (:20001)
│   ├── migrate/                       #   Database migration CLI
│   │   └── main.go                    #     flags: --action (up/down/steps/version/force)
│   ├── seed/                          #   Seed data runner
│   │   └── main.go                    #     flags: --table (auth_user)
│   ├── sync/                          #   Data synchronization CLI
│   │   └── main.go                    #     flags: --table --mode --batch --since
│   └── import/                        #   CSV data importer
│       ├── main.go                    #     flags: --service --path --env
│       └── import_data_guide.md       #     usage guide
│
├── config/                            # ─── Configuration ───
│   ├── config.go                      #   Config struct definitions (Server, DB, Redis, Kafka, OTel...)
│   ├── loader.go                      #   Viper loader: YAML → env → defaults → validate
│   └── loader_external_db.go          #   External DB config parser (environment-based)
│
├── server/                            # ─── HTTP Server ───
│   ├── server.go                      #   Fiber app: middlewares, routes, module registration
│   ├── server_test.go                 #   health/ready/kafka endpoint tests
│   └── middleware/                     #   Custom Fiber middlewares
│       ├── auth.go                    #     JWT Bearer + API Key middleware (per-endpoint)
│       ├── auth_test.go               #     JWT + API Key middleware tests
│       ├── access_log.go              #     Structured request logging (zerolog)
│       └── access_log_test.go          #     access log middleware tests
│
├── internal/                          # ─── Internal Business Logic ───
│   │
│   ├── modules/                       # ── Domain Modules (Hexagonal Architecture) ──
│   │   │
│   │   ├── auth/                      #   🔐 Authentication & Authorization
│   │   │   ├── module.go              #     Register(router, deps) — wiring & routes
│   │   │   ├── domain/
│   │   │   │   └── auth.go            #     User, Session — pure domain models
│   │   │   ├── ports/
│   │   │   │   ├── user_repository.go #     UserRepository interface
│   │   │   │   └── token_signer.go    #     TokenSigner interface
│   │   │   ├── app/
│   │   │   │   ├── service.go         #     AuthService — login, verify, password check
│   │   │   │   ├── service_test.go    #     unit tests with fakes
│   │   │   │   └── fakes_test.go      #     fake implementations for testing
│   │   │   └── adapters/
│   │   │       ├── http/
│   │   │       │   ├── controller.go  #       AuthController interface
│   │   │       │   └── handler.go     #       Fiber HTTP handler
│   │   │       ├── postgres/
│   │   │       │   └── user_repository.go  #  pgx implementation
│   │   │       └── external/
│   │   │           ├── simple_token_signer.go    # dev token signer
│   │   │           └── static_user_repository.go # dev static users
│   │   │
│   │   ├── cmi/                       #   📋 CMI Policy Management
│   │   │   ├── module.go              #     Register (requires external DB "meprakun")
│   │   │   ├── integration_test.go    #     real DB integration test (env-gated)
│   │   │   ├── domain/
│   │   │   │   └── cmi.go             #     CMIPolicy, MotorInfo, InsuredInfo, etc.
│   │   │   ├── ports/
│   │   │   │   └── repository.go      #     CMIPolicyRepository interface
│   │   │   ├── app/
│   │   │   │   ├── service.go         #     CMIService — find policy by job ID
│   │   │   │   ├── service_test.go    #     unit tests
│   │   │   │   └── fakes_test.go      #     fake repository
│   │   │   └── adapters/
│   │   │       ├── http/
│   │   │       │   ├── controller.go  #       CMIController interface
│   │   │       │   └── handler.go     #       Fiber HTTP handler
│   │   │       └── postgres/
│   │   │           └── repository.go  #       pgx query (complex JOIN)
│   │   │
│   │   ├── quotation/                 #   💰 Quotation Management
│   │   │   ├── module.go              #     Register (requires external DB "meprakun")
│   │   │   ├── domain/
│   │   │   │   └── quotation.go       #     Quotation model
│   │   │   ├── ports/
│   │   │   │   └── repository.go      #     QuotationRepository interface
│   │   │   ├── app/
│   │   │   │   ├── service.go         #     QuotationService — find by ID/customer
│   │   │   │   ├── service_test.go    #     unit tests
│   │   │   │   └── fakes_test.go      #     fake repository
│   │   │   └── adapters/
│   │   │       ├── http/
│   │   │       │   ├── controller.go  #       QuotationController interface
│   │   │       │   └── handler.go     #       Fiber HTTP handler
│   │   │       └── postgres/
│   │   │           └── repository.go  #       pgx queries with pagination
│   │   │
│   │   ├── externaldb/                #   🔌 External Database Diagnostics
│   │   │   ├── module.go              #     Register — health check routes
│   │   │   ├── domain/
│   │   │   │   └── externaldb.go      #     DBStatus model
│   │   │   ├── ports/
│   │   │   │   └── db_provider.go     #     DBProvider interface
│   │   │   ├── app/
│   │   │   │   ├── service.go         #     ExternalDBService — list status
│   │   │   │   ├── service_test.go    #     unit tests
│   │   │   │   └── fakes_test.go      #     fake provider
│   │   │   └── adapters/
│   │   │       └── http/
│   │   │           ├── controller.go  #       Controller interface
│   │   │           └── handler.go     #       Fiber HTTP handler
│   │   │
│   │   ├── document/                  #   📄 Document Management (planned)
│   │   ├── job/                       #   ⚙️  Job Processing (planned)
│   │   ├── notification/              #   🔔 Notification System (planned)
│   │   ├── payment/                   #   💳 Payment Processing (planned)
│   │   ├── policy/                    #   📜 Policy Management (planned)
│   │   └── webhook/                   #   🔗 GitHub Webhook → Discord
│   │       ├── module.go              #     Register — webhook routes
│   │       ├── domain/
│   │       │   └── webhook.go         #     WebhookEvent model
│   │       ├── ports/
│   │       │   └── notifier.go        #     Notifier interface
│   │       ├── app/
│   │       │   └── service.go         #     WebhookService — verify + process + notify
│   │       └── adapters/
│   │           ├── http/handler.go    #       Fiber HTTP handler
│   │           └── discord/client.go  #       Discord webhook sender
│   │
│   ├── shared/                        # ── Shared Internal Packages ──
│   │   ├── dto/
│   │   │   ├── response.go            #   ApiResponse, ErrorResponse, ErrorResult, helpers
│   │   │   └── error_codes.go         #   Error Code Catalog — 17 TraceId constants (5 modules + auth middleware)
│   │   ├── enum/
│   │   │   ├── health.go              #   HealthOK, HealthNotReady
│   │   │   ├── response.go            #   StatusSuccess, StatusFail
│   │   │   ├── role.go                #   RoleAdmin, RoleOps, RoleViewer
│   │   │   └── stage.go               #   StageLocal, StageStaging, StageProduction
│   │   ├── module/
│   │   │   └── deps.go                #   Deps struct — shared DI container
│   │   ├── pagination/
│   │   │   ├── pagination.go          #   Request, Response[T], Defaults
│   │   │   ├── fiber.go               #   FromFiber() — parse query params
│   │   │   ├── sql.go                 #   Query builder (SQL-safe, AllowedColumns)
│   │   │   ├── pagination_test.go     #   response + defaults tests
│   │   │   └── sql_test.go            #   SQL builder + injection prevention tests
│   │   ├── validator/
│   │   │   ├── validator.go           #   go-playground/validator singleton + FormatErrors
│   │   │   ├── bind.go                #   BindAndValidate(c, &dto) Fiber helper
│   │   │   └── validator_test.go      #   5 validation tests
│   │   └── utils/
│   │       ├── doc.go                 #   package documentation
│   │       ├── id.go                  #   NewID(prefix) — crypto/rand based
│   │       ├── json.go                #   MaskJSON, PrettyJSONBytes
│   │       ├── pointer.go             #   Ptr[T], Deref[T], DerefOr[T]
│   │       ├── slice.go              #   Contains[T], Unique[T], Map[A,B], Filter[T]
│   │       └── string.go             #   Truncate
│   │
│   ├── database/                      # ── Database Layer (Multi-Driver) ──
│   │   ├── provider.go                #   Provider interface (Main, External, Read, Write)
│   │   ├── conn.go                    #   ExternalConn interface + type-safe helpers (PgxPool, SQLDB)
│   │   ├── manager.go                 #   Manager — multi-DB lifecycle (main + externals)
│   │   ├── postgres/
│   │   │   ├── connect.go             #   PostgreSQL connector — DSN build, pool tuning, OTel
│   │   │   ├── connect_test.go        #   MaskDSN tests
│   │   │   └── migrate.go             #   MigrateUp/Down/Steps/Force/Version
│   │   ├── mysql/
│   │   │   └── connect.go             #   MySQL connector — TLS, pool tuning, MultiStatements=false
│   │   └── seed/
│   │       ├── runner.go              #   Seed dispatcher
│   │       ├── auth_user_seed.go      #   User seed with bcrypt
│   │       └── auth_user_seed_test.go #   seed validation tests
│   │
│   ├── import/                        # ── CSV Data Import ──
│   │   ├── csv_reader.go              #   CSV parser with header normalization
│   │   ├── runner.go                  #   Import dispatcher by service type
│   │   ├── insurer_importer.go        #   insurer CSV → DB upsert
│   │   ├── insurer_installment_importer.go  # installment CSV → DB upsert
│   │   ├── province_importer.go       #   province CSV → DB upsert
│   │   └── user_importer.go           #   user CSV → DB upsert
│   │
│   ├── sync/                          # ── Data Synchronization ──
│   │   ├── syncer.go                  #   Syncer interface + SyncRequest/SyncResult
│   │   ├── registry.go                #   Syncer registry (name → impl)
│   │   ├── runner.go                  #   RunOne / RunAll with context cancellation
│   │   ├── quotation.go               #   QuotationSyncer — batch upsert from external DB
│   │   └── sync_test.go              #   registry + runner tests with fakes
│   │
│   └── testkit/                       # ── Test Assertion Library ──
│       ├── doc.go                     #   package documentation
│       ├── assert.go                  #   Equal, NotEqual, True, Nil, NoError, Contains, Len...
│       ├── must.go                    #   MustEqual, MustNoError — fatal on fail
│       ├── fixture.go                 #   Fixture(), LoadJSON(), Golden()
│       └── assert_test.go            #   31 tests for assertion functions
│
├── pkg/                               # ─── Reusable Libraries (importable) ───
│   │
│   ├── banner/                        #   🎨 Startup Banner
│   │   ├── banner.go                  #     Unicode box art, ANSI colors, NO_COLOR support
│   │   └── banner_test.go            #     alignment, border, row tests
│   │
│   ├── buildinfo/                     #   🏗️  Build Metadata
│   │   └── buildinfo.go               #     GitCommit, BuildTime (injected via ldflags)
│   │
│   ├── cache/                         #   🗄️  Redis Cache Abstraction
│   │   ├── cache.go                   #     Cache interface + Client (get/set/delete/JSON)
│   │   └── errors.go                  #     ErrCacheMiss sentinel error
│   │
│   ├── httpclient/                    #   🌐 HTTP Client
│   │   ├── client.go                  #     Functional options, retry (5xx only), OTel tracing, circuit breaker
│   │   ├── options.go                 #     BaseURL, Timeout, WithRetry, WithHeader, WithCircuitBreaker...
│   │   ├── errors.go                  #     ResponseError (IsServerError/IsClientError/IsCircuitOpen)
│   │   ├── client_test.go            #     GET/POST/PUT/PATCH/DELETE, retry, context tests
│   │   └── errors_test.go            #     error type tests
│   │
│   ├── kafka/                         #   📨 Kafka Event System
│   │   ├── producer.go                #     Producer — publish with RequireAll acks
│   │   ├── consumer.go                #     Consumer — at-least-once, DLQ support
│   │   ├── router.go                  #     Router — event type dispatch + fallback
│   │   ├── message.go                 #     Message envelope (type, key, payload, metadata)
│   │   ├── tracing.go                 #     W3C trace context propagation (inject/extract)
│   │   ├── message_test.go           #     creation, validation, router tests
│   │   └── tracing_test.go           #     trace roundtrip test
│   │
│   ├── localcache/                    #   ⚡ In-Memory Cache (Otter)
│   │   ├── localcache.go              #     Cache interface + Client (S3-FIFO eviction)
│   │   └── hybrid.go                  #     Hybrid L1(otter) + L2(Redis) with write-through
│   │
│   ├── log/                           #   📝 Structured Logging
│   │   └── logger.go                  #     Zerolog: JSON (prod) / pretty (dev), global singleton
│   │
│   ├── otel/                          #   📊 OpenTelemetry
│   │   ├── otel.go                    #     Init: OTLP/HTTP traces + Prometheus metrics
│   │   ├── middleware.go              #     Fiber middleware (W3C tracing, skip health/metrics)
│   │   ├── tracername.go              #     Central tracer name registry (sync.Map cached)
│   │   └── tracername_test.go        #     no-duplicate, no-empty tests
│   │
│   └── retry/                         #   🔄 Retry with Backoff
│       ├── retry.go                   #     Do(ctx, fn, opts) — Exponential/Constant/Linear/Custom
│       └── retry_test.go             #     success, exhaustion, context cancel, all strategies
│
├── migrations/                        # ─── Database Migrations (golang-migrate) ───
│   ├── 000001_create_users_table.up.sql
│   ├── 000001_create_users_table.down.sql
│   ├── 000002_create_insurer_tables.up.sql
│   ├── 000002_create_insurer_tables.down.sql
│   ├── 000003_create_province_table.up.sql
│   └── 000003_create_province_table.down.sql
│
├── base_data/                         # ─── Seed/Import CSV Data ───
│   ├── insurer_installment.csv
│   └── users.csv
│
├── testdata/                          # ─── Test Fixtures ───
│   └── cmi/
│       └── (test JSON fixtures)
│
├── docs/                              # ─── Swagger (auto-generated by swag init) ───
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
│
├── documents/                         # ─── Technical Documentation ───
│   ├── README.md                      #   สารบัญรวมทุกเอกสาร
│   ├── architecture/
│   │   ├── README.md                  #   architecture overview
│   │   ├── project-structure.md       #   ← this file
│   │   ├── database-concept.md        #   Internal & External DB concept
│   │   ├── kafka-concept.md           #   Kafka event-driven architecture concept
│   │   ├── microservice-readiness.md  #   microservice extraction guide
│   │   └── swagger-concept.md         #   Swagger/OpenAPI guide + Error Code Catalog
│   ├── cicd/
│   │   ├── ci-cd-guide.md             #   CI/CD pipeline guide (+ Local CI)
│   │   ├── ci-pipeline-stages.md      #   7 stages breakdown
│   │   ├── workflow-concept.md        #   push → CI → staging → tag → production
│   │   ├── dependabot-concept.md      #   Dependabot config & concept
│   │   └── github-actions-setup.md    #   GitHub Actions + Secrets setup
│   ├── observability/
│   │   ├── otel-tracing-guide.md      #   OpenTelemetry tracing ฉบับเต็ม
│   │   └── otel-grafana-quickstart.md #   Grafana + Tempo + Prometheus quick start
│   ├── infrastructure/
│   │   ├── deployment-guide.md        #   Docker + Kubernetes deployment
│   │   ├── kubernetes-guide.md        #   Kubernetes concepts + manifests
│   │   ├── resource-spec-guide.md     #   CPU/Memory requests & limits
│   │   ├── redis-cache-guide.md       #   Redis + Hybrid L1→L2 cache
│   │   └── discord-notification.md    #   Discord Webhook notification
│   ├── testing/
│   │   ├── unit-test-guide.md         #   unit test ฉบับเต็ม
│   │   ├── unit-test-cheatsheet.md    #   cheatsheet — patterns, examples, commands
│   │   └── code-coverage-concept.md   #   coverage คืออะไร + วิธีวัด
│   └── operations/
│       ├── incident-response-runbook.md #  incident response (SEV1-4)
│       └── iso-standards-checklist.md #   ISO 27001/9001 checklist
│
├── deployments/                       # ─── Infrastructure ───
│   ├── docker/
│   │   ├── Dockerfile                 #   multi-stage: builder → api + worker (Alpine, non-root)
│   │   └── Dockerfile.worker          #   standalone worker image (backward-compat)
│   ├── k8s/
│   │   ├── README.md
│   │   ├── base/                      #   Kustomize base manifests
│   │   │   ├── kustomization.yaml
│   │   │   ├── namespace.yaml
│   │   │   ├── configmap.yaml         #     full app config
│   │   │   ├── secret.yaml            #     credentials (→ External Secrets Operator)
│   │   │   ├── api-deployment.yaml    #     2 replicas, probes, security context
│   │   │   ├── api-service.yaml       #     ClusterIP service
│   │   │   ├── api-ingress.yaml       #     NGINX ingress, rate limit, TLS
│   │   │   ├── api-hpa.yaml           #     HPA 2-6, CPU 70% / Memory 80%
│   │   │   ├── api-pdb.yaml           #     PodDisruptionBudget
│   │   │   ├── worker-deployment.yaml #     Kafka consumer worker
│   │   │   ├── migrate-job.yaml       #     DB migration init job
│   │   │   └── sync-cronjob.yaml      #     periodic data sync
│   │   └── overlays/                  #   Kustomize per-environment overrides
│   │       ├── staging/
│   │       │   └── kustomization.yaml #     2-4 pods, OTel sample 50%, Swagger on
│   │       └── production/
│   │           └── kustomization.yaml #     3-8 pods, OTel sample 5%, Swagger off, CORS locked
│   ├── local/
│   │   ├── docker-compose.yaml        #   PostgreSQL 17 + Redis 7 + Kafka 3.9 (KRaft) + Kafka UI
│   │   ├── init-db.sql                #   create main + external databases
│   │   ├── .env                       #   local env vars
│   │   └── .env.example               #   env template
│   └── observability/
│       ├── docker-compose.yaml        #   OTel Collector + Prometheus + Tempo + Grafana
│       ├── otel-collector.yaml        #   collector config (receivers → processors → exporters)
│       ├── prometheus.yaml            #   scrape config
│       ├── tempo.yaml                 #   trace storage config
│       └── grafana/
│           └── provisioning/
│               └── datasources/
│                   └── datasources.yaml  # auto-provision Prometheus + Tempo
│
├── .github/                           # ─── Repo Config ───
├── .air.local.toml                    #   hot-reload config (air)
├── .dockerignore                      #   Docker build exclusions
├── .env.local                         #   local environment overrides
├── .env.local.example                 #   env template for developers
├── .gitignore                         #   Git ignore rules
├── .golangci.yml                      #   golangci-lint config (17 linters)
├── go.mod                             #   Go module definition
├── go.sum                             #   dependency checksums
├── Makefile                           #   build targets (Linux/macOS)
├── run.ps1                            #   build targets (Windows PowerShell)
└── README.md                          #   project overview
```

---

## Layer Descriptions

### `cmd/` — Application Entrypoints

แต่ละ binary เป็น **single-responsibility**:

| Binary | Purpose | Deploy As |
| ------ | ------- | --------- |
| `cmd/api` | HTTP API server | K8s Deployment |
| `cmd/worker` | Kafka event consumer | K8s Deployment |
| `cmd/migrate` | Database migration CLI | K8s Job (init) |
| `cmd/seed` | Insert seed/test data | Manual / CI |
| `cmd/sync` | Data synchronization | K8s CronJob |
| `cmd/import` | CSV data import | Manual |

### `config/` — Configuration Management

```text
YAML file → Environment Variables → Defaults → Struct Validation
   ↓              ↓                    ↓              ↓
 viper         godotenv             hardcoded    go-playground/validator
```

- `Config` struct มี validation tags ครบทุก field
- Production guard: JWT secret ต้องตั้งค่า, `StageStatus` ต้องเป็น `local|staging|production`

### `internal/modules/` — Domain Modules

ทุก module ใช้ **Hexagonal Architecture** pattern เดียวกัน 100%:

```text
module/
├── module.go           ← Wiring: สร้าง adapter → inject เข้า service → mount routes
├── domain/             ← Pure domain models (zero external dependencies)
├── ports/              ← Interfaces (inbound: controller, outbound: repository)
├── app/                ← Application service (business logic, depends on ports only)
│   ├── service.go
│   ├── service_test.go ← Unit tests with fakes
│   └── fakes_test.go   ← Fake implementations
└── adapters/           ← Concrete implementations
    ├── http/           ← Inbound: Fiber HTTP handler
    │   ├── controller.go  (interface)
    │   └── handler.go     (implementation)
    ├── postgres/       ← Outbound: pgx SQL repository
    │   └── repository.go
    └── external/       ← Outbound: 3rd party integrations
```

### `internal/shared/` — Cross-Module Utilities

| Package | Purpose |
| ------- | ------- |
| `dto` | API response envelope (`ApiResponse`, `Success`, `Error`) + Error Code Catalog |
| `enum` | String constants (roles, stages, health status) |
| `module` | `Deps` struct + `Middleware` (JWTAuth, APIKeyAuth) — shared DI container |
| `pagination` | Generic `Response[T]`, SQL-safe query builder |
| `validator` | Request body validation (`BindAndValidate`, go-playground/validator) |
| `utils` | Generics: `Ptr[T]`, `Contains[T]`, `NewID()`, `MaskJSON()` |

### `pkg/` — Reusable Libraries

Packages under `pkg/` ไม่ depend on `internal/` — สามารถ extract ออกเป็น Go module แยกได้:

| Package | Key Feature |
| ------- | ----------- |
| `cache` | Redis abstraction + `ErrCacheMiss` sentinel |
| `httpclient` | Functional options, smart retry (5xx only), OTel tracing, circuit breaker |
| `kafka` | Event envelope + DLQ + W3C trace propagation + health probe |
| `localcache` | Otter (S3-FIFO) + Hybrid L1/L2 write-through |
| `log` | Zerolog — JSON (prod) / pretty console (dev) |
| `otel` | OTLP/HTTP traces + Prometheus metrics + Fiber middleware |
| `retry` | Exponential / Constant / Linear / Custom backoff |
| `banner` | Unicode startup banner with ANSI colors |
| `buildinfo` | Git commit + build time via ldflags |

### `deployments/` — Infrastructure as Code

```text
deployments/
├── docker/          → Multi-stage Docker builds (Alpine, non-root)
├── k8s/
│   ├── base/        → Kustomize base (shared manifests)
│   └── overlays/    → Per-environment patches (staging, production)
├── local/           → Docker Compose for local development
└── observability/   → OTel Collector + Prometheus + Tempo + Grafana
```

---

## Module Structure Pattern

ทุก module ที่ implement แล้วจะมีโครงสร้างนี้ตรงกัน:

```text
┌──────────────────────────────────────────────────────────┐
│                     module.go (Wiring)                    │
│  func Register(router fiber.Router, deps module.Deps)    │
│                                                          │
│  1. repo := postgres.NewRepository(deps.DB.Main())       │
│  2. svc  := app.NewService(repo)                         │
│  3. ctrl := http.NewXxxController(svc)                   │
│  4. group := router.Group("/xxx")                        │
│  5. group.GET("/...", deps.Middleware.JWTAuth, ctrl.Find) │
└──────────────────────┬───────────────────────────────────┘
                       │
     ┌─────────────────┼─────────────────┐
     ▼                 ▼                 ▼
┌─────────┐     ┌───────────┐     ┌───────────┐
│ domain/ │     │  ports/   │     │ adapters/ │
│         │     │           │     │           │
│ Models  │◄────│ Interfaces│◄────│ Concrete  │
│ (pure)  │     │ (inbound  │     │ (http,    │
│         │     │  outbound)│     │  postgres, │
│         │     │           │     │  external) │
└─────────┘     └─────┬─────┘     └───────────┘
                      │
                ┌─────▼─────┐
                │   app/    │
                │           │
                │ Service   │
                │ (business │
                │  logic)   │
                │           │
                │ Uses ports│
                │ interfaces│
                └───────────┘
```

---

## Dependency Flow

```text
                    ┌──────────────┐
                    │  cmd/api     │
                    │  (bootstrap) │
                    └──────┬───────┘
                           │ creates
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
    ┌─────────┐      ┌──────────┐      ┌─────────┐
    │ config  │      │ server   │      │  pkg/*  │
    └─────────┘      └────┬─────┘      └─────────┘
                          │ registers
                  ┌───────┼───────┐
                  ▼       ▼       ▼
             ┌────────────────────────┐
             │  internal/modules/*    │
             │  (via module.Register) │
             └────────┬───────────────┘
                      │ depends on
         ┌────────────┼────────────┐
         ▼            ▼            ▼
    ┌──────────┐ ┌──────────┐ ┌──────────┐
    │ shared/  │ │ database/│ │  pkg/*   │
    │ dto,enum │ │ provider │ │ cache,   │
    │ paginate │ │ postgres │ │ kafka... │
    └──────────┘ └──────────┘ └──────────┘
```

**Rules:**

1. `internal/modules/*` → ห้าม import module อื่น (no cross-module dependency)
2. `internal/shared/*` → ห้าม import `internal/modules/*` (no upward dependency)
3. `pkg/*` → ห้าม import `internal/*` (public library, zero internal coupling)
4. `domain/` → ห้าม import อะไรนอกจาก stdlib (pure models)
5. `ports/` → import ได้เฉพาะ `domain/` (interface definitions)
6. `app/` → import ได้เฉพาะ `ports/` + `domain/` (business logic)
7. `adapters/` → import ได้ทุกอย่าง (concrete implementations)

---

## Key Design Decisions

| Decision | Rationale |
| -------- | --------- |
| **Modular Monolith** | พัฒนาง่าย, deploy ง่าย, แยก module ชัดเจน — extract เป็น microservice ได้ทีหลัง |
| **Hexagonal Architecture** | Testable (mock ports), Swappable (เปลี่ยน DB/cache ได้), Clean dependency direction |
| **`module.Deps` struct** | Dependency Injection แบบง่าย — ไม่ต้องใช้ DI framework |
| **`pkg/` vs `internal/`** | `pkg/` คือ library ที่ reuse ได้ข้าม project, `internal/` คือ business logic เฉพาะ |
| **Kustomize overlays** | Config per-environment โดยไม่ต้อง duplicate manifests |
| **Multi-Driver DB** | postgres (pgx v5) + mysql (go-sql-driver) — driver-agnostic `ExternalConn` interface |
| **Otter + Redis hybrid** | L1 (in-memory, microsecond) → L2 (Redis, millisecond) → DB (source of truth) |
| **Kafka with DLQ** | At-least-once delivery + Dead Letter Queue สำหรับ failed events |
| **OTel (not Datadog/NR)** | Vendor-neutral, W3C standard, works with Grafana/Jaeger/Tempo |
| **golangci-lint 17 linters** | gosec, bodyclose, noctx, sqlclosecheck — catch issues before PR |
| **Route-Level Auth** | JWT + API Key middleware applied per-endpoint via `Deps.Middleware` — ไม่ใช้ global auth |
| **`internal/testkit/`** | Zero-dependency assertion library — ไม่ต้อง import testify |

---

> **Note:** Modules ที่แสดงว่า *(planned)* — `document`, `job`, `notification`, `payment`, `policy` — มี directory placeholder ไว้แล้ว  
> เมื่อพร้อม implement ให้สร้างตาม pattern เดียวกัน: `domain/ → ports/ → app/ → adapters/ → module.go`
