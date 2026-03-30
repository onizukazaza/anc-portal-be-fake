# CI/CD Guide — ANC Portal Backend

> **v2.1** — Last updated: March 2026
>
> คู่มือ CI/CD Pipeline สำหรับ ANC Portal Backend
> ครอบคลุม: สถานะปัจจุบัน, GitHub Actions, Docker Build, K8s Deploy, Secret Management

---

## สารบัญ

1. [CI/CD คืออะไร](#1-cicd-คืออะไร)
2. [สถานะปัจจุบัน — อะไรพร้อมแล้ว](#2-สถานะปัจจุบัน--อะไรพร้อมแล้ว)
3. [อะไรที่ยังขาด](#3-อะไรที่ยังขาด)
4. [Branch Strategy](#4-branch-strategy)
5. [GitHub Actions — CI Pipeline](#5-github-actions--ci-pipeline)
6. [GitHub Actions — CD Pipeline](#6-github-actions--cd-pipeline)
7. [Docker Build Pipeline](#7-docker-build-pipeline)
8. [Kubernetes Deployment Pipeline](#8-kubernetes-deployment-pipeline)
9. [Database Migration Strategy](#9-database-migration-strategy)
10. [Secret Management](#10-secret-management)
11. [Testing Pipeline](#11-testing-pipeline)
12. [Monitoring & Alerting](#12-monitoring--alerting)
13. [Checklist & Runbook](#13-checklist--runbook)
14. [สรุป](#14-สรุป)

---

## 1. CI/CD คืออะไร

### Continuous Integration (CI)

ทุกครั้งที่ push code → ระบบ **อัตโนมัติ** ตรวจสอบว่า:

- Code compile ผ่าน
- Code style ถูกต้อง (lint)
- Unit test ผ่านทั้งหมด
- Coverage ไม่ต่ำกว่าเกณฑ์
- Docker image build ได้
- ไม่มีช่องโหว่ด้านความปลอดภัย

### Continuous Deployment (CD)

เมื่อ CI ผ่าน → ระบบ **deploy อัตโนมัติ** ไปยัง environment ที่กำหนด:

```
develop branch  → Staging
main branch     → Production (หลัง approval)
```

### ทำไมต้องมี CI/CD

| ปัญหา (ไม่มี CI/CD)            | แก้ได้ด้วย CI/CD          |
| ------------------------------- | ------------------------- |
| Push code เจ๊งแล้วไม่รู้        | Lint + Test ตรวจทุก push  |
| Deploy ด้วยมือ พลาดง่าย         | Auto deploy ทุกขั้นตอน    |
| ลืม run migration               | Migration Job อัตโนมัติ   |
| Image tag ไม่ตรงกับ code        | SHA-based tags            |
| ใครแก้อะไรเมื่อไหร่ไม่รู้        | Git SHA + Build metadata  |

---

## 2. สถานะปัจจุบัน — อะไรพร้อมแล้ว

### Readiness Score: 9/10

Infrastructure layer + automation layer **พร้อมใช้งาน** (CI/CD pipeline implement แล้ว)

### สิ่งที่พร้อมแล้ว

| หมวด | รายละเอียด | ไฟล์ |
|------|------------|------|
| **Local CI** | `run.ps1 ci` — 4 step pipeline + Discord notification + failure details | `run.ps1` |
| **Makefile** | 25+ targets ครบทุก workflow | `Makefile` |
| **Dockerfile** | Multi-stage 4 ขั้น (builder → runtime-base → api → worker) | `deployments/docker/Dockerfile` |
| **K8s Manifests** | 12 resources + Kustomize overlays (staging/production) | `deployments/k8s/` |
| **Health Checks** | liveness + readiness + startup probes | `api-deployment.yaml` |
| **HPA** | Auto-scaling 2-6 pods (staging 2-4, prod 3-8) | `api-hpa.yaml` |
| **PDB** | Pod Disruption Budget (minAvailable: 1/2) | `api-pdb.yaml` |
| **Build Metadata** | GitCommit + BuildTime via ldflags | `pkg/buildinfo/` |
| **Migration Job** | K8s Job สำหรับ pre-deploy migration | `migrate-job.yaml` |
| **Sync CronJob** | ทุก 6 ชม. sync external DB | `sync-cronjob.yaml` |
| **Security** | non-root, readOnlyRootFilesystem, no privilege escalation | ทุก Deployment |
| **Observability** | OTel → Tempo + Prometheus + Grafana | `deployments/observability/` |
| **Docker Compose** | Local dev stack (PostgreSQL + Redis + Kafka) | `deployments/local/` |

### Makefile Targets ที่ CI/CD ใช้ได้เลย

```bash
# CI
make test                # Run all unit tests
make build               # Build with ldflags (GitCommit, BuildTime)
make swagger             # Generate OpenAPI docs
make tidy                # go mod tidy

# Docker
make docker-build        # Build API image
make docker-build-worker # Build Worker image (--target worker)

# Database
make migrate             # Run migrations
make seed                # Seed initial data
```

### Dockerfile Architecture

```
┌──────────────────────────────────────────────────────────┐
│  Stage 1: builder (golang:1.25-alpine)                   │
│  ├── go mod download  (cached layer)                     │
│  ├── COPY source code                                    │
│  └── Build 5 binaries: api, worker, migrate, seed, sync │
├──────────────────────────────────────────────────────────┤
│  Stage 2: runtime-base (alpine:3.21)                     │
│  ├── ca-certificates, tzdata, curl                       │
│  └── non-root user: appuser                              │
├──────────────────────────────────────────────────────────┤
│  Stage 3: api (default target)                           │
│  ├── 5 binaries + migrations/ + config/                  │
│  ├── HEALTHCHECK: curl http://localhost:20000/healthz    │
│  └── ENTRYPOINT: ./api                                   │
├──────────────────────────────────────────────────────────┤
│  Stage 4: worker (--target worker)                       │
│  ├── worker binary + config/                             │
│  └── ENTRYPOINT: ./worker                                │
└──────────────────────────────────────────────────────────┘
```

---

## 3. อะไรที่ยังขาด

| รายการ | ความสำคัญ | สถานะ |
|--------|----------|-------|
| GitHub Actions Workflows | สูงมาก | ทำแล้ว — `ci.yml`, `deploy-staging.yml`, `deploy-production.yml`, `release.yml` |
| golangci-lint config (`.golangci.yml`) | สูง | ทำแล้ว — 17 linters |
| Secret encryption (ESO/Sealed Secrets) | สูง | Placeholder CHANGE_ME — ต้องตั้งค่า |
| Container image scanning | ปานกลาง | ทำแล้ว — Trivy ใน CI (API + Worker) |
| Vulnerability check | ปานกลาง | ทำแล้ว — govulncheck ใน CI |
| Test coverage reporting | ปานกลาง | ทำแล้ว — coverage artifact + summary |
| Discord CI/CD notification | ปานกลาง | ทำแล้ว — ทุก workflow + local `run.ps1 ci` |
| Alerting rules (PrometheusRule) | ปานกลาง | ยังไม่มี |
| Smoke test after deploy | ต่ำ | ทำแล้ว — staging healthz check |
| Canary / Blue-Green deploy | ต่ำ | ใช้ RollingUpdate |

---

## 4. Branch Strategy

### Git Flow (แนะนำ)

```
feature/*  ──→  develop  ──→  main
                  │              │
                  ▼              ▼
               Staging       Production
```

| Branch | Environment | Auto Deploy | Protection |
|--------|-------------|-------------|------------|
| `feature/*` | — | CI only (lint + test) | — |
| `develop` | Staging | auto deploy หลัง CI ผ่าน | require PR review |
| `main` | Production | deploy หลัง manual approval | require PR review + admin approval |
| `hotfix/*` | — | CI only | merge กลับทั้ง main + develop |

### Tag Convention

```
v1.0.0          → production release
v1.0.0-rc.1     → release candidate (staging test)
staging         → rolling tag สำหรับ staging image
```

### Commit Convention

```
feat: เพิ่ม feature ใหม่
fix: แก้ bug
docs: อัปเดตเอกสาร
refactor: ปรับ code ไม่เปลี่ยน behavior
test: เพิ่ม/แก้ test
ci: เปลี่ยน CI/CD config
chore: อื่นๆ (deps, config)
```

---

## 5. GitHub Actions — CI Pipeline

### Workflow: `.github/workflows/ci.yml`

**7 Jobs:** lint, test, vuln → build → docker → scan → notify

```yaml
name: CI

on:
  push:
    branches: [develop, main]
  pull_request:
    branches: [develop, main]

concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true

env:
  GO_VERSION: "1.25"

jobs:
  # ─────────────────────────────────
  # Job 1: Lint
  # ─────────────────────────────────
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest

  # ─────────────────────────────────
  # Job 2: Test
  # ─────────────────────────────────
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      - name: Run tests
        run: go test -race -coverprofile=coverage.out -covermode=atomic -count=1 ./...
      - name: Coverage summary
        run: |
          go tool cover -func=coverage.out | tail -1
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "## Test Coverage: ${COVERAGE}%" >> "$GITHUB_STEP_SUMMARY"
      - name: Upload coverage artifact
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: coverage-report
          path: coverage.out
          retention-days: 14

  # ─────────────────────────────────
  # Job 3: Vulnerability Check
  # ─────────────────────────────────
  vuln:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      - name: Install govulncheck
        run: go install golang.org/x/vuln/cmd/govulncheck@latest
      - name: Run govulncheck
        run: govulncheck ./...

  # ─────────────────────────────────
  # Job 4: Build
  # ─────────────────────────────────
  build:
    runs-on: ubuntu-latest
    needs: [lint, test, vuln]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      - name: Build all binaries
        run: |
          GIT_COMMIT=$(git rev-parse --short HEAD)
          BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
          LDFLAGS="-s -w -X .../pkg/buildinfo.GitCommit=${GIT_COMMIT} ..."
          go build -ldflags="${LDFLAGS}" -o /dev/null ./cmd/api
          go build -ldflags="${LDFLAGS}" -o /dev/null ./cmd/worker
          go build -ldflags="-s -w"     -o /dev/null ./cmd/migrate
          go build -ldflags="-s -w"     -o /dev/null ./cmd/seed
          go build -ldflags="-s -w"     -o /dev/null ./cmd/sync

  # ─────────────────────────────────
  # Job 5: Docker Build + Push (push only)
  # ─────────────────────────────────
  docker:
    runs-on: ubuntu-latest
    needs: [build]
    if: github.event_name == 'push'
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Build and push API image
        uses: docker/build-push-action@v6
        with:
          context: .
          file: deployments/docker/Dockerfile
          target: api
          push: true
          cache-from: type=gha
          cache-to: type=gha,mode=max
      - name: Build and push Worker image
        uses: docker/build-push-action@v6
        with:
          context: .
          file: deployments/docker/Dockerfile
          target: worker
          push: true
          cache-from: type=gha
          cache-to: type=gha,mode=max

  # ─────────────────────────────────
  # Job 6: Security Scan — Trivy
  # ─────────────────────────────────
  scan:
    runs-on: ubuntu-latest
    needs: [docker]
    if: github.event_name == 'push'
    steps:
      - name: Scan API image (Trivy)
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: ghcr.io/${{ github.repository }}:${{ github.ref_name }}
          format: table
          exit-code: 1
          severity: CRITICAL,HIGH
      - name: Upload SARIF to GitHub Security tab
        uses: github/codeql-action/upload-sarif@v3
        if: always()

  # ─────────────────────────────────
  # Job 7: Notify — Discord
  # ─────────────────────────────────
  notify:
    runs-on: ubuntu-latest
    if: always()         # ← ทำงานทุก event (push + PR)
    needs: [lint, test, vuln, build, docker, scan]
    steps:
      - name: Determine overall status + collect failed jobs
      - name: Build Discord embed payload (with PR info + failure details)
      - name: Send Discord notification
        if: vars.DISCORD_WEBHOOK_URL != ''
```

### CI Flow Diagram

```
Push / PR
    │
    ├── lint  (golangci-lint)     ─┐
    ├── test  (go test -race)     ─┼── parallel
    └── vuln  (govulncheck)       ─┘
              │
              ▼
          build (5 binaries with ldflags)
              │
              ▼  (push event only)
          docker (API + Worker → GHCR)
              │
              ▼  (push event only)
          scan (Trivy CRITICAL/HIGH → GitHub Security tab)
              │
              ▼  (always — push + PR)
          notify (Discord embed: status, jobs, PR info, failure details)
```

---

## 6. GitHub Actions — CD Pipeline

### Workflow: `.github/workflows/deploy-staging.yml`

```yaml
name: Deploy Staging

on:
  push:
    branches: [develop]
  workflow_dispatch:

env:
  KUBE_NAMESPACE: anc-portal

jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: staging
    steps:
      - uses: actions/checkout@v4

      - name: Set image tag
        id: tag
        run: echo "sha=$(git rev-parse --short HEAD)" >> "$GITHUB_OUTPUT"

      - name: Configure kubectl
        uses: azure/setup-kubectl@v4

      - name: Set kubeconfig
        run: echo "${{ secrets.KUBE_CONFIG_STAGING }}" | base64 -d > $HOME/.kube/config

      # Step 1: Run migration job
      - name: Run database migration
        run: |
          # ลบ job เก่าถ้ามี
          kubectl delete job stg-anc-portal-migrate -n $KUBE_NAMESPACE --ignore-not-found

          # Apply migration job with new image
          cd deployments/k8s/overlays/staging
          kustomize edit set image \
            ghcr.io/onizukazaza/anc-portal-be-fake=ghcr.io/onizukazaza/anc-portal-be-fake:${{ steps.tag.outputs.sha }}
          kustomize build . | kubectl apply -f - -l app.kubernetes.io/component=migration

          # Wait for migration to complete
          kubectl wait --for=condition=complete job/stg-anc-portal-migrate \
            -n $KUBE_NAMESPACE --timeout=120s

      # Step 2: Deploy API + Worker
      - name: Deploy application
        run: |
          cd deployments/k8s/overlays/staging
          kustomize build . | kubectl apply -f -

      # Step 3: Wait for rollout
      - name: Wait for rollout
        run: |
          kubectl rollout status deployment/stg-anc-portal-api \
            -n $KUBE_NAMESPACE --timeout=180s
          kubectl rollout status deployment/stg-anc-portal-worker \
            -n $KUBE_NAMESPACE --timeout=180s

      # Step 4: Smoke test
      - name: Smoke test
        run: |
          # ใช้ port-forward ตรวจ health
          kubectl port-forward svc/stg-anc-portal-api 20000:80 \
            -n $KUBE_NAMESPACE &
          sleep 5
          HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:20000/healthz)
          if [ "$HTTP_CODE" != "200" ]; then
            echo "::error::Smoke test failed — healthz returned ${HTTP_CODE}"
            exit 1
          fi
          echo "Smoke test passed — healthz returned 200"
```

### Workflow: `.github/workflows/deploy-production.yml`

```yaml
name: Deploy Production

on:
  push:
    tags: ["v*"]
  workflow_dispatch:
    inputs:
      tag:
        description: "Image tag to deploy"
        required: true

env:
  KUBE_NAMESPACE: anc-portal

jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: production    # requires manual approval
    steps:
      - uses: actions/checkout@v4

      - name: Determine tag
        id: tag
        run: |
          if [ -n "${{ github.event.inputs.tag }}" ]; then
            echo "image_tag=${{ github.event.inputs.tag }}" >> "$GITHUB_OUTPUT"
          else
            echo "image_tag=${GITHUB_REF_NAME}" >> "$GITHUB_OUTPUT"
          fi

      - name: Configure kubectl
        uses: azure/setup-kubectl@v4

      - name: Set kubeconfig
        run: echo "${{ secrets.KUBE_CONFIG_PRODUCTION }}" | base64 -d > $HOME/.kube/config

      # Step 1: Migration
      - name: Run database migration
        run: |
          kubectl delete job prod-anc-portal-migrate -n $KUBE_NAMESPACE --ignore-not-found
          cd deployments/k8s/overlays/production
          kustomize edit set image \
            ghcr.io/onizukazaza/anc-portal-be-fake=ghcr.io/onizukazaza/anc-portal-be-fake:${{ steps.tag.outputs.image_tag }}
          kustomize build . | kubectl apply -f - -l app.kubernetes.io/component=migration
          kubectl wait --for=condition=complete job/prod-anc-portal-migrate \
            -n $KUBE_NAMESPACE --timeout=180s

      # Step 2: Deploy
      - name: Deploy application
        run: |
          cd deployments/k8s/overlays/production
          kustomize build . | kubectl apply -f -

      # Step 3: Rollout
      - name: Wait for rollout
        run: |
          kubectl rollout status deployment/prod-anc-portal-api \
            -n $KUBE_NAMESPACE --timeout=300s
          kubectl rollout status deployment/prod-anc-portal-worker \
            -n $KUBE_NAMESPACE --timeout=300s

      # Step 4: Verify
      - name: Verify deployment
        run: |
          kubectl get pods -n $KUBE_NAMESPACE -l app.kubernetes.io/part-of=anc-portal
          echo "Production deploy completed — tag: ${{ steps.tag.outputs.image_tag }}"
```

### CD Flow Diagram

```
develop push ──→ CI ผ่าน ──→ Deploy Staging (auto)
                                  │
                                  ├── 1. Migration Job
                                  ├── 2. Apply Deployments
                                  ├── 3. Wait Rollout
                                  └── 4. Smoke Test

main tag (v*) ──→ CI ผ่าน ──→ Deploy Production (manual approval)
                                  │
                                  ├── 1. Migration Job
                                  ├── 2. Apply Deployments
                                  ├── 3. Wait Rollout
                                  └── 4. Verify Pods
```

---

## 7. Docker Build Pipeline

### Image Naming Convention

| Image | Registry | ตัวอย่าง Tag |
|-------|----------|-------------|
| API | `ghcr.io/onizukazaza/anc-portal-be-fake` | `abc1234`, `develop`, `v1.0.0`, `latest` |
| Worker | `ghcr.io/onizukazaza/anc-portal-be-fake` | `abc1234-worker`, `develop-worker` |

### Tag Strategy

```
Push to develop  → ghcr.io/onizukazaza/anc-portal-be-fake:develop
                 → ghcr.io/onizukazaza/anc-portal-be-fake:<short-sha>

Push tag v1.0.0  → ghcr.io/onizukazaza/anc-portal-be-fake:v1.0.0
                 → ghcr.io/onizukazaza/anc-portal-be-fake:latest
```

### Build Optimization

โปรเจกต์ใช้ optimization เหล่านี้แล้ว:

1. **Dependency caching** — `go mod download` แยก layer ก่อน COPY source
2. **Multi-stage build** — builder ไม่ถูก ship ไป production
3. **Stripped binaries** — `-ldflags="-s -w"` ลดขนาด binary
4. **Single build layer** — build 5 binaries ใน `RUN` เดียว
5. **Minimal runtime** — `alpine:3.21` + เฉพาะ ca-certificates, tzdata, curl
6. **GitHub Actions cache** — `cache-from: type=gha` ลดเวลา rebuild

### Image Security

| มาตรการ | สถานะ | รายละเอียด |
|---------|--------|-----------|
| Non-root user | พร้อม | `appuser:appgroup` |
| Read-only filesystem | พร้อม | `readOnlyRootFilesystem: true` |
| No privilege escalation | พร้อม | `allowPrivilegeEscalation: false` |
| .dockerignore | พร้อม | exclude .git, .env*, test, docs |
| Container scanning | ทำแล้ว | Trivy ใน CI workflow |

### แนะนำ: เพิ่ม Container Scanning

```yaml
# เพิ่มเป็น step ใน CI workflow
- name: Run Trivy vulnerability scanner
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: ghcr.io/onizukazaza/anc-portal-be-fake:${{ github.sha }}
    format: table
    exit-code: 1
    severity: CRITICAL,HIGH
```

---

## 8. Kubernetes Deployment Pipeline

### Resource Overview

```
deployments/k8s/
├── base/                          # Shared resources
│   ├── kustomization.yaml         # Resource list + image tags
│   ├── namespace.yaml             # anc-portal namespace
│   ├── configmap.yaml             # 40+ env vars (non-secret)
│   ├── secret.yaml                # DB_PASSWORD, JWT_SECRET_KEY, etc.
│   ├── api-deployment.yaml        # API (2 replicas, RollingUpdate)
│   ├── api-service.yaml           # ClusterIP:80
│   ├── api-ingress.yaml           # NGINX + TLS
│   ├── api-hpa.yaml               # 2-6 pods (CPU 70% / Mem 80%)
│   ├── api-pdb.yaml               # minAvailable: 1
│   ├── worker-deployment.yaml     # Worker (1 replica)
│   ├── migrate-job.yaml           # Pre-deploy migration
│   └── sync-cronjob.yaml          # Every 6h data sync
└── overlays/
    ├── staging/kustomization.yaml
    └── production/kustomization.yaml
```

### Staging vs Production

| Config | Staging | Production |
|--------|---------|------------|
| API replicas | 2 | 3 |
| Worker replicas | 1 | 2 |
| HPA range | 2–4 pods | 3–8 pods |
| PDB minAvailable | 1 | 2 |
| CPU request/limit | 100m / 500m | 200m / 1000m |
| Memory request/limit | 128Mi / 256Mi | 256Mi / 512Mi |
| OTEL_SAMPLE_RATIO | 0.5 | 0.05 |
| SWAGGER_ENABLED | true | false |
| Ingress host | api-staging.anc-portal.example.com | api.portal.anc.co.th |
| Image tag strategy | `staging` / `<sha>` | `v1.0.0` / `<sha>` |
| namePrefix | `stg-` | `prod-` |

### Rolling Update Strategy

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1        # เพิ่ม pod ใหม่ 1 ตัวก่อน
    maxUnavailable: 0  # ไม่มี pod ที่ unavailable ระหว่าง update
```

ผลลัพธ์: **Zero-downtime deployment** — pod ใหม่ต้อง ready ก่อน ค่อยลบ pod เก่า

### Deploy Commands (Manual)

```bash
# Staging
cd deployments/k8s/overlays/staging
kustomize edit set image ghcr.io/onizukazaza/anc-portal-be-fake=ghcr.io/onizukazaza/anc-portal-be-fake:<tag>
kustomize build . | kubectl apply -f -

# Production
cd deployments/k8s/overlays/production
kustomize edit set image ghcr.io/onizukazaza/anc-portal-be-fake=ghcr.io/onizukazaza/anc-portal-be-fake:v1.0.0
kustomize build . | kubectl apply -f -
```

### Rollback

```bash
# ดู revision history
kubectl rollout history deployment/stg-anc-portal-api -n anc-portal

# Rollback ไป revision ก่อนหน้า
kubectl rollout undo deployment/stg-anc-portal-api -n anc-portal

# Rollback ไป revision เฉพาะ
kubectl rollout undo deployment/stg-anc-portal-api -n anc-portal --to-revision=3
```

---

## 9. Database Migration Strategy

### ระบบ Migration ปัจจุบัน

โปรเจกต์มี **3 วิธี** ในการ run migration:

| วิธี | ใช้เมื่อ | Command |
|------|--------|---------|
| CLI โดยตรง | Local development | `make migrate` หรือ `go run ./cmd/migrate` |
| K8s Job | Staging/Production deploy | `kubectl apply -f migrate-job.yaml` |
| Docker exec | Container debugging | `docker exec <container> ./migrate --action up` |

### Migration Files

```
migrations/
├── 000001_create_users_table.up.sql
├── 000001_create_users_table.down.sql
├── 000002_create_insurer_tables.up.sql
├── 000002_create_insurer_tables.down.sql
├── 000003_create_province_table.up.sql
└── 000003_create_province_table.down.sql
```

### CI/CD Migration Flow

```
1. CI ผ่าน
2. Docker image push (มี ./migrate binary อยู่ใน image)
3. CD workflow:
   a. ลบ migration job เก่า
   b. Apply migration job ด้วย image tag ใหม่
   c. Wait for job complete (timeout 120s staging / 180s prod)
   d. ถ้า migration fail → deploy หยุด ไม่ rollout app
   e. ถ้า migration pass → deploy API + Worker
```

### กฎสำคัญ

1. **Migration ต้อง backward compatible** — ห้าม rename/drop column ที่ app version เก่ายังใช้
2. **เขียน down migration เสมอ** — สำหรับ rollback
3. **Test migration ที่ staging ก่อนทุกครั้ง**
4. **ใช้ transaction** — ถ้า migration fail กลางทาง จะ rollback ทั้ง migration
5. **backoffLimit: 3** — retry 3 ครั้งก่อน fail

---

## 10. Secret Management

### สถานะปัจจุบัน

```yaml
# deployments/k8s/base/secret.yaml — ⚠️ Placeholder values
stringData:
  DB_USER: "anc_app"
  DB_PASSWORD: "CHANGE_ME"        # ← ต้องเปลี่ยน
  REDIS_PASSWORD: ""
  JWT_SECRET_KEY: "CHANGE_ME"     # ← ต้องเปลี่ยน
```

### ปัญหา

- Secret เป็น plaintext ใน Git (base64 ≠ encryption)
- Placeholder `CHANGE_ME` ไม่มีกลไกบังคับให้เปลี่ยน
- ไม่มี secret rotation

### แนวทางแก้ไข (เลือก 1)

#### Option A: External Secrets Operator (ESO) — แนะนำ

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: anc-portal-secret
  namespace: anc-portal
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager    # หรือ vault, gcp-sm
    kind: ClusterSecretStore
  target:
    name: anc-portal-secret
  data:
    - secretKey: DB_PASSWORD
      remoteRef:
        key: anc-portal/db-password
    - secretKey: JWT_SECRET_KEY
      remoteRef:
        key: anc-portal/jwt-secret
    - secretKey: REDIS_PASSWORD
      remoteRef:
        key: anc-portal/redis-password
```

ข้อดี: secret ไม่อยู่ใน Git, auto-refresh, audit trail

#### Option B: Sealed Secrets (Bitnami)

```bash
# Encrypt secret ด้วย public key ของ cluster
kubeseal --format yaml < secret.yaml > sealed-secret.yaml
# sealed-secret.yaml เก็บใน Git ได้ปลอดภัย (encrypted)
```

#### Option C: GitHub Secrets + CI Inject

```yaml
# ใน CD workflow
- name: Create K8s secret
  run: |
    kubectl create secret generic anc-portal-secret \
      --from-literal=DB_PASSWORD=${{ secrets.DB_PASSWORD }} \
      --from-literal=JWT_SECRET_KEY=${{ secrets.JWT_SECRET_KEY }} \
      --from-literal=REDIS_PASSWORD=${{ secrets.REDIS_PASSWORD }} \
      -n anc-portal --dry-run=client -o yaml | kubectl apply -f -
```

### CI/CD Secrets ที่ต้องตั้งค่าใน GitHub

| Secret Name | ใช้ใน | คำอธิบาย |
|-------------|------|---------|
| `KUBE_CONFIG_STAGING` | CD staging | kubeconfig (base64) สำหรับ staging cluster |
| `KUBE_CONFIG_PRODUCTION` | CD production | kubeconfig (base64) สำหรับ production cluster |
| `DB_PASSWORD` | CD (ถ้าใช้ Option C) | Database password |
| `JWT_SECRET_KEY` | CD (ถ้าใช้ Option C) | JWT signing key |
| `REDIS_PASSWORD` | CD (ถ้าใช้ Option C) | Redis password |

---

## 11. Testing Pipeline

### Test Pyramid

```
         ╱  E2E  ╲           ← น้อย / ช้า / Staging only
        ╱──────────╲
       ╱ Integration ╲       ← ปานกลาง / CI (กับ DB)
      ╱────────────────╲
     ╱    Unit Tests    ╲    ← มาก / เร็ว / ทุก push
    ╱────────────────────╲
```

### CI Test Targets

```bash
# Unit tests (ไม่ต้องพึ่ง external service)
go test -race -coverprofile=coverage.out ./...

# Integration tests (ต้องมี PostgreSQL)
go test -race -tags=integration ./internal/modules/cmi/...
```

### Coverage Strategy

| Stage | เกณฑ์ | Action |
|-------|-------|--------|
| เริ่มต้น (ปัจจุบัน) | 40% | Warning |
| 3 เดือน | 60% | CI fail ถ้าต่ำกว่า |
| 6 เดือน | 70% | CI fail ถ้าต่ำกว่า |

### golangci-lint Config (ใช้งานแล้ว)

ไฟล์ `.golangci.yml` อยู่ที่ root — มี 17 linters:

```yaml
run:
  timeout: 5m
  go: "1.25"

linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - typecheck
    - bodyclose
    - noctx
    - gosec
    - prealloc
    - exportloopref
    - gocritic

linters-settings:
  govet:
    shadow: true
  gosec:
    excludes:
      - G104  # unhandled errors ที่เป็น defer
  gocritic:
    enabled-tags:
      - diagnostic
      - performance

issues:
  exclude-dirs:
    - docs
    - testdata
    - tmp
  max-issues-per-linter: 50
  max-same-issues: 5
```

---

## 12. Monitoring & Alerting

### Observability Stack ปัจจุบัน

```
App (OTel SDK)
    │
    ▼
OTel Collector
    ├──→ Tempo (traces)
    ├──→ Prometheus (metrics)
    └──→ Grafana (dashboards)
```

ดูรายละเอียด: [OTel Tracing Guide](../integrations/otel-tracing-guide.md)

### แนะนำ: Alerting Rules

```yaml
# alerting-rules.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: anc-portal-alerts
spec:
  groups:
    - name: anc-portal
      rules:
        # Pod restart มากกว่า 3 ครั้งใน 15 นาที
        - alert: HighPodRestartRate
          expr: rate(kube_pod_container_status_restarts_total{namespace="anc-portal"}[15m]) > 0.2
          for: 5m
          labels:
            severity: warning

        # API response time > 2s (p95)
        - alert: HighLatency
          expr: histogram_quantile(0.95, rate(http_server_duration_milliseconds_bucket{service="anc-portal-be"}[5m])) > 2000
          for: 5m
          labels:
            severity: warning

        # Error rate > 5%
        - alert: HighErrorRate
          expr: |
            sum(rate(http_server_duration_milliseconds_count{service="anc-portal-be",http_status_code=~"5.."}[5m]))
            /
            sum(rate(http_server_duration_milliseconds_count{service="anc-portal-be"}[5m]))
            > 0.05
          for: 5m
          labels:
            severity: critical

        # Migration job failed
        - alert: MigrationJobFailed
          expr: kube_job_status_failed{namespace="anc-portal",job_name=~".*migrate.*"} > 0
          for: 1m
          labels:
            severity: critical
```

### CI/CD Notifications

ทุก workflow มี Discord notification:

| Workflow | Discord | รายละเอียด |
|----------|---------|------------|
| CI (`ci.yml`) | ✅ Job `notify` | Rich embed: status emoji, per-job icons, PR info, failure details, link to logs |
| Deploy Staging | ✅ Success + Failure | แจ้ง deploy สำเร็จ/ล้มเหลว |
| Deploy Production | ✅ Success + Failure | แจ้ง deploy สำเร็จ/ล้มเหลว |
| Release | ✅ | แจ้ง release ใหม่ + changelog link |
| Local CI (`run.ps1 ci`) | ✅ | แจ้งผลรวม + failure details (code block, max 1000 chars) |

#### CI Discord Notification Features

- **Status emoji**: ✅ = passed, ❌ = failed, ⚠️ = cancelled
- **Per-job status icons**: แสดง icon ของทุก job (lint, test, vuln, build, docker, scan)
- **PR info**: ถ้าเป็น pull_request event — แสดง PR number, title, branch direction
- **Failure details**: เมื่อ fail — แสดงชื่อ job ที่ fail + link ไปดู logs
- **ทำงานทุก event**: ทั้ง `push` และ `pull_request` (ไม่จำกัดแค่ push)

#### ตัวอย่าง Discord Embed (CI Failed)

```
❌ CI Pipeline Failed
──────────────────────────
Branch:  `feature/auth`
Commit:  `a1b2c3d`
Author:  onchainjpeg
Message: fix: update login handler

Jobs:  ❌ Lint  ✅ Test  ✅ Vuln
       ⏭️ Build  ⏭️ Docker  ⏭️ Scan

⚠ Failed Steps: Lint
   View Logs → [link]
```

---

## 13. Checklist & Runbook

### Pre-Implementation Checklist

| # | รายการ | ความสำคัญ | ประมาณเวลา |
|---|--------|----------|-----------|
| 1 | สร้าง `.golangci.yml` | สูง | 30 นาที |
| 2 | สร้าง `.github/workflows/ci.yml` | สูงมาก | 2 ชม. |
| 3 | สร้าง `.github/workflows/deploy-staging.yml` | สูงมาก | 2 ชม. |
| 4 | สร้าง `.github/workflows/deploy-production.yml` | สูง | 1 ชม. |
| 5 | ตั้ง GitHub Secrets (KUBE_CONFIG, etc.) | สูง | 30 นาที |
| 6 | ตั้ง GitHub Environment + protection rules | สูง | 30 นาที |
| 7 | แก้ secret.yaml → ใช้ ESO หรือ Sealed Secrets | สูง | 2 ชม. |
| 8 | ~~เพิ่ม Trivy container scanning~~ | ~~ปานกลาง~~ | ✅ ทำแล้ว |
| 9 | เพิ่ม Prometheus alerting rules | ปานกลาง | 1 ชม. |
| 10 | ~~เพิ่ม Discord notification~~ | ~~ต่ำ~~ | ✅ ทำแล้ว |

### Deploy Runbook — Staging

```
1. ✅ Push code ไป develop branch
2. ✅ CI ผ่าน (lint + test + build + docker push)
3. ✅ CD auto-trigger
4. ✅ Migration job complete
5. ✅ API + Worker rollout complete
6. ✅ Smoke test healthz = 200
7. ✅ ตรวจ Grafana dashboards ว่าไม่มี error spike
```

### Deploy Runbook — Production

```
1. ✅ Merge develop → main (PR + review)
2. ✅ Tag version: git tag v1.0.0 && git push --tags
3. ✅ CI ผ่าน
4. ✅ CD trigger → รอ manual approval ใน GitHub
5. ✅ Approve deployment
6. ✅ Migration job complete
7. ✅ API + Worker rollout complete
8. ✅ Verify pods running
9. ✅ ตรวจ Grafana dashboards 15 นาที
10. ✅ แจ้งทีมว่า deploy สำเร็จ
```

### Rollback Runbook

```
1. ❌ พบปัญหาหลัง deploy
2. 🔍 ตรวจ logs: kubectl logs -n anc-portal -l app.kubernetes.io/name=anc-portal-api
3. 🔄 Rollback: kubectl rollout undo deployment/prod-anc-portal-api -n anc-portal
4. 🔄 Rollback worker: kubectl rollout undo deployment/prod-anc-portal-worker -n anc-portal
5. ⏳ Wait rollout: kubectl rollout status deployment/prod-anc-portal-api -n anc-portal
6. ✅ Verify: kubectl get pods -n anc-portal
7. 📋 ถ้า migration ต้อง rollback: kubectl run migrate-down --image=<old-tag> -- ./migrate --action down
8. 📢 แจ้งทีม + สร้าง post-mortem
```

---

## 14. สรุป

### What We Have (พร้อมแล้ว)

- Makefile 25+ targets ครอบคลุมทุก workflow
- Multi-stage Dockerfile (4 stages, 5 binaries)
- Kustomize base + overlays (staging/production)
- Health probes (liveness + readiness + startup)
- HPA (auto-scaling) + PDB (disruption protection)
- Zero-downtime RollingUpdate strategy
- Build metadata injection (GitCommit + BuildTime)
- Security hardening (non-root, read-only FS, no privilege escalation)
- Observability stack (OTel → Tempo + Prometheus + Grafana)
- Migration Job + Sync CronJob
- **CI Pipeline** — lint + test + vuln check + build + docker push + Trivy scan + **Discord notify**
- **CD Staging** — auto deploy หลัง CI ผ่าน + migration + smoke test + Discord notify
- **CD Production** — tag-based deploy + manual approval + pod verification + Discord notify
- **Release** — auto changelog + GitHub Release + Discord notify
- **golangci-lint** — 17 linters (errcheck, govet, staticcheck, gosec, etc.)
- **Dependabot** — auto-update Go modules + Actions + Docker images
- **Discord Notifications** — ทุก workflow + local `run.ps1 ci` (with failure details)
- **Local CI** — `run.ps1 ci` พร้อม failure details display + Discord notification

### What We Need (ต้องทำเพิ่ม)

- Secret management — ESO หรือ Sealed Secrets (แทน plaintext secret.yaml)
- Alerting rules — PrometheusRule
- GitHub Secrets + Environment — ตั้งค่าบน GitHub Settings

### Priority Order

```
Phase 1: ✅ golangci-lint + CI workflow           — ทำแล้ว
Phase 2: ✅ CD staging + CD production              — ทำแล้ว
Phase 3: ✅ Trivy scanning + Dependabot             — ทำแล้ว
Phase 4: ✅ Discord notification (CI + CD + local)  — ทำแล้ว
Phase 5: ⏳ Secret management + Alerting + GitHub Settings — รอตั้งค่า
```

### Local CI Pipeline (`run.ps1 ci`)

นอกจาก GitHub Actions แล้ว ยังมี **local CI** สำหรับรันบนเครื่อง developer:

```powershell
.\run.ps1 ci
```

**4 Steps:** Lint → Test → Vuln → Build (fail-fast — หยุดทันทีที่ step แรก fail)

ตัวอย่างผลลัพธ์:

```
  CI Pipeline - ANC Portal Backend
  =================================

  [1/4] Lint PASS (2.6s)
  [2/4] Test PASS (6.7s)
  [3/4] Vuln PASS (8.4s)
  [4/4] Build PASS (3.4s)

  +---------+--------+---------+
  | Step    | Status | Time    |
  +---------+--------+---------+
  | Lint    | PASS   | 2.6s    |
  | Test    | PASS   | 6.7s    |
  | Vuln    | PASS   | 8.4s    |
  | Build   | PASS   | 3.4s    |
  +---------+--------+---------+

  PIPELINE PASSED (total: 21.2s)

  Sending to Discord... sent!
```

เมื่อ fail จะแสดง **Failure Details** พร้อมข้อความ error จาก tool ที่ล้มเหลว และส่ง Discord notification พร้อม failure details (code block, ตัดที่ 1000 ตัวอักษร)

---

> v2.1 — March 2026 | ANC Portal Backend Team
