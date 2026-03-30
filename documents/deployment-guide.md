# Deployment Guide — ANC Portal Backend

> **v2.1** — Last updated: March 2026
>
> คู่มือการ deploy ตั้งแต่ Local Development จนถึง Production
> ครอบคลุม Docker, Kubernetes, และ CI/CD pipeline

---

## สารบัญ

1. [ภาพรวม Environments](#1-ภาพรวม-environments)
2. [Local Development](#2-local-development)
3. [Docker Build](#3-docker-build)
4. [Staging Deploy](#4-staging-deploy)
5. [Production Deploy](#5-production-deploy)
6. [Database Migration](#6-database-migration)
7. [Environment Variables](#7-environment-variables)
8. [Health Checks](#8-health-checks)
9. [Rollback Strategy](#9-rollback-strategy)
10. [Checklist ก่อน Deploy](#10-checklist-ก่อน-deploy)

---

## 1. ภาพรวม Environments

```
Local (dev machine)  →  Staging (K8s)  →  Production (K8s)
     │                      │                    │
  Docker Compose       Kustomize overlay    Kustomize overlay
  .env.local           staging/             production/
  Hot reload (air)     2-4 pods             3-8 pods + HPA
  All infra optional   Swagger เปิด         Swagger ปิด
```

| Environment | Infrastructure | Replicas | Swagger | OTel Sample |
|---|---|---|---|---|
| **Local** | Docker Compose | 1 | เปิด | 100% |
| **Staging** | Kubernetes (Kustomize) | 2-4 | เปิด | 50% |
| **Production** | Kubernetes (Kustomize) | 3-8 | ปิด | 5-10% |

---

## 2. Local Development

### Prerequisites

| เครื่องมือ | วิธีติดตั้ง |
|---|---|
| Go 1.25+ | https://go.dev/dl/ |
| Docker Desktop | https://docker.com |
| Air (hot reload) | `go install github.com/air-verse/air@latest` |

### ขั้นตอน

```powershell
# 1. เปิด infrastructure (PostgreSQL, Redis, Kafka)
cd deployments/local
docker compose up -d

# 2. กลับไป root
cd ../..

# 3. Copy environment file
copy .env.example .env.local

# 4. รัน migration
.\run.ps1 migrate

# 5. Seed ข้อมูลเริ่มต้น (optional)
.\run.ps1 seed

# 6. รัน API server (hot reload)
.\run.ps1 dev

# 7. รัน Worker (tab ใหม่)
.\run.ps1 worker

# 8. รัน Local CI (ตรวจสอบก่อน push)
.\run.ps1 ci
# Lint → Test → Vuln → Build + Discord notification
```

### Observability Stack (optional)

```powershell
cd deployments/observability
docker compose up -d
# Grafana: http://localhost:3001
# Prometheus: http://localhost:9090
```

### ตรวจสอบระบบ

```powershell
# Health check
curl http://localhost:20000/healthz

# Readiness
curl http://localhost:20000/ready

# Swagger UI
start http://localhost:20000/swagger/index.html
```

---

## 3. Docker Build

### Multi-stage Dockerfile

```
Dockerfile
├── Stage 1: builder      ← go build (API + Worker)
└── Stage 2: runtime       ← scratch/distroless (binary only)
```

### Build Commands

```bash
# API image
docker build -f deployments/docker/Dockerfile \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -t ghcr.io/onizukazaza/anc-portal-be-fake:latest .

# Worker image
docker build -f deployments/docker/Dockerfile.worker \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -t ghcr.io/anc-portal/anc-portal-worker:latest .
```

### Build Info

`-ldflags` inject build information ที่แสดงใน startup banner:

```
Build ··············· a1b2c3d (2026-03-28T10:30:00Z)
```

### Makefile Shortcuts

```bash
make build           # Build API binary
make build-worker    # Build Worker binary
make docker-build    # Build Docker image (API)
```

---

## 4. Staging Deploy

### Prerequisites

- kubectl ≥ 1.28
- Access to Kubernetes cluster
- Container registry access (ghcr.io)

### ขั้นตอน

```bash
# 1. Build & push images
docker build -f deployments/docker/Dockerfile -t ghcr.io/onizukazaza/anc-portal-be-fake:staging .
docker push ghcr.io/onizukazaza/anc-portal-be-fake:staging

# 2. สร้าง Secrets (ครั้งแรกเท่านั้น)
kubectl create secret generic anc-portal-secret \
  --namespace=anc-portal \
  --from-literal=DB_USER=anc_app \
  --from-literal=DB_PASSWORD='<password>' \
  --from-literal=JWT_SECRET_KEY='<jwt-secret>'

# 3. รัน Migration
kubectl apply -f deployments/k8s/base/migrate-job.yaml
kubectl wait --for=condition=complete job/anc-portal-migrate -n anc-portal --timeout=120s

# 4. Deploy
kubectl apply -k deployments/k8s/overlays/staging

# 5. ตรวจสอบ
kubectl get pods -n anc-portal -w
kubectl logs -n anc-portal -l app.kubernetes.io/name=anc-portal-api -f
```

### Preview Manifests (dry-run)

```bash
kubectl kustomize deployments/k8s/overlays/staging
```

---

## 5. Production Deploy

### ความแตกต่างจาก Staging

| ค่า | Staging | Production |
|---|---|---|
| Replicas | 2-4 | 3-8 (HPA) |
| CPU | 100m-500m | 200m-1000m |
| Memory | 128Mi-256Mi | 256Mi-512Mi |
| OTel Sample | 50% | 5% |
| Swagger | เปิด | ปิด |
| PDB minAvailable | 1 | 2 |
| CORS | `*` | `https://portal.anc.co.th` |
| Secrets | kubectl | External Secrets Operator |

### ขั้นตอน

```bash
# 1. Tag image ด้วย semantic version
docker tag ghcr.io/onizukazaza/anc-portal-be-fake:staging ghcr.io/onizukazaza/anc-portal-be-fake:v1.0.0
docker push ghcr.io/onizukazaza/anc-portal-be-fake:v1.0.0

# 2. รัน Migration (ก่อน deploy เสมอ)
kubectl apply -f deployments/k8s/base/migrate-job.yaml
kubectl wait --for=condition=complete job/anc-portal-migrate -n anc-portal --timeout=120s

# 3. Deploy
kubectl apply -k deployments/k8s/overlays/production

# 4. ดู rollout status
kubectl rollout status deployment/anc-portal-api -n anc-portal

# 5. ตรวจ health
kubectl port-forward -n anc-portal svc/anc-portal-api 8080:80
curl http://localhost:8080/healthz
```

### Secrets Management (Production)

แนะนำใช้ **External Secrets Operator** สำหรับ production:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: anc-portal-secret
  namespace: anc-portal
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: ClusterSecretStore
  target:
    name: anc-portal-secret
  data:
    - secretKey: DB_PASSWORD
      remoteRef:
        key: anc-portal/db-password
```

---

## 6. Database Migration

### รัน Migration

```bash
# Local
.\run.ps1 migrate

# Kubernetes
kubectl apply -f deployments/k8s/base/migrate-job.yaml
kubectl wait --for=condition=complete job/anc-portal-migrate -n anc-portal --timeout=120s
kubectl logs -n anc-portal job/anc-portal-migrate
```

### กฎสำคัญ

- รัน migration **ก่อน** deploy API เสมอ
- Migration ต้องเป็น backward-compatible (ไม่ลบ column ที่ code เก่ายังใช้)
- ใช้ Kubernetes Job — รันครั้งเดียวแล้วจบ
- Job มี `ttlSecondsAfterFinished: 300` — ลบอัตโนมัติหลังสำเร็จ 5 นาที

---

## 7. Environment Variables

### ตัวแปรหลัก

| Variable | ตัวอย่าง | คำอธิบาย |
|---|---|---|
| `STAGE_STATUS` | `local` / `staging` / `production` | กำหนด logging format + behavior |
| `SERVER_PORT` | `20000` | HTTP server port |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_NAME` | `anc_portal` | Database name |
| `DB_MAX_CONNS` | `20` | Connection pool max |
| `DB_MIN_CONNS` | `5` | Connection pool min |

### Feature Toggles

| Variable | Default | คำอธิบาย |
|---|---|---|
| `REDIS_ENABLED` | `false` | เปิด/ปิด Redis cache |
| `KAFKA_ENABLED` | `false` | เปิด/ปิด Kafka messaging |
| `OTEL_ENABLED` | `false` | เปิด/ปิด OpenTelemetry |
| `LOCAL_CACHE_ENABLED` | `false` | เปิด/ปิด Otter (L1 cache) |
| `SWAGGER_ENABLED` | `true` | เปิด/ปิด Swagger UI |
| `RATE_LIMIT_ENABLED` | `false` | เปิด/ปิด rate limiting |

> ทุก feature toggle สามารถปิดได้ — ระบบยังทำงานได้ปกติ

---

## 8. Health Checks

### API Server

| Endpoint | ตรวจสอบ | ใช้สำหรับ |
|---|---|---|
| `GET /healthz` | DB + Redis connectivity | Liveness probe |
| `GET /ready` | DB + Redis + timestamp | Readiness probe |
| `GET /metrics` | Prometheus metrics | Monitoring scrape |

### Worker (Kafka Consumer)

| Endpoint | ตรวจสอบ | ใช้สำหรับ |
|---|---|---|
| `GET /healthz` (port 20001) | `consumer.IsHealthy()` (atomic.Bool) | Liveness + Readiness probe |

> Worker มี HTTP health server แยกบน port 20001 — return 200 เมื่อ consumer healthy, 503 เมื่อ not ready

### ตัวอย่าง Response

```json
// GET /healthz (healthy)
{ "status": "ok" }

// GET /healthz (degraded — Redis ไม่เชื่อม)
{ "status": "degraded", "error": "redis: connection refused" }
```

---

## 9. Rollback Strategy

### Kubernetes Rolling Update (default)

```bash
# ดู revision history
kubectl rollout history deployment/anc-portal-api -n anc-portal

# Rollback ไป revision ก่อนหน้า
kubectl rollout undo deployment/anc-portal-api -n anc-portal

# Rollback ไป revision เฉพาะ
kubectl rollout undo deployment/anc-portal-api -n anc-portal --to-revision=3
```

### Database Migration Rollback

```bash
# รัน migration down (ระวัง — อาจลบ data)
# ต้องแก้ migrate-job.yaml ให้รัน "down" แทน "up"
```

> **กฎ:** Migration ควร backward-compatible เสมอ เพื่อให้ rollback API ได้โดยไม่ต้อง rollback DB

---

## 10. Checklist ก่อน Deploy

### Staging

- [ ] Local CI ผ่านทั้งหมด (`run.ps1 ci` — Lint, Test, Vuln, Build)
- [ ] Unit tests ผ่านทั้งหมด (`go test ./...`)
- [ ] Docker build สำเร็จ
- [ ] Migration tested locally
- [ ] Environment variables ครบ (ConfigMap + Secret)
- [ ] Health check endpoints ตอบกลับถูกต้อง

### Production

- [ ] ผ่าน staging testing แล้ว
- [ ] Image tag เป็น semantic version (`v1.0.0`) ไม่ใช่ `latest`
- [ ] Migration backward-compatible
- [ ] Secrets ใช้ External Secrets Operator (ไม่ใช่ plain kubectl)
- [ ] OTel sample ratio ลดเหลือ 5-10%
- [ ] Swagger ปิด (`SWAGGER_ENABLED=false`)
- [ ] CORS origins เฉพาะ domain จริง
- [ ] PDB minAvailable ≥ 2
- [ ] HPA configured + Metrics Server ready
- [ ] Rollback plan พร้อม

---

> **v2.1** — March 2026 | ANC Portal Backend Team
