# Kubernetes Deployment — ANC Portal Backend

> **v2.0** — Last updated: March 2026
>
> Kubernetes manifests สำหรับ deploy ด้วย Kustomize (base + overlays)
>
> ดู Deployment Guide ฉบับเต็ม: [documents/infrastructure/deployment-guide.md](../../documents/infrastructure/deployment-guide.md)

---

## สารบัญ

- [Kubernetes Deployment — ANC Portal Backend](#kubernetes-deployment--anc-portal-backend)
  - [สารบัญ](#สารบัญ)
  - [โครงสร้างไฟล์](#โครงสร้างไฟล์)
  - [Prerequisites](#prerequisites)
  - [Quick Start](#quick-start)
  - [Resource Overview](#resource-overview)
  - [Deploy ทีละขั้นตอน](#deploy-ทีละขั้นตอน)
    - [ขั้นตอนที่ 1: สร้าง Namespace](#ขั้นตอนที่-1-สร้าง-namespace)
    - [ขั้นตอนที่ 2: สร้าง Secret (ใส่ค่าจริง)](#ขั้นตอนที่-2-สร้าง-secret-ใส่ค่าจริง)
    - [ขั้นตอนที่ 3: รัน Migration](#ขั้นตอนที่-3-รัน-migration)
    - [ขั้นตอนที่ 4: Deploy ทั้งหมด](#ขั้นตอนที่-4-deploy-ทั้งหมด)
    - [ขั้นตอนที่ 5: ตรวจสอบ](#ขั้นตอนที่-5-ตรวจสอบ)
  - [Overlay: Staging vs Production](#overlay-staging-vs-production)
  - [Database Migration](#database-migration)
    - [Migration ใน CI/CD Pipeline](#migration-ใน-cicd-pipeline)
  - [Scaling](#scaling)
    - [Auto-scaling (HPA)](#auto-scaling-hpa)
    - [Manual scaling](#manual-scaling)
  - [Health Checks \& Probes](#health-checks--probes)
    - [Worker Probes](#worker-probes)
  - [Secrets Management](#secrets-management)
    - [วิธีที่ 1: kubectl (Dev/Staging)](#วิธีที่-1-kubectl-devstaging)
    - [วิธีที่ 2: External Secrets Operator (Production — แนะนำ)](#วิธีที่-2-external-secrets-operator-production--แนะนำ)
    - [วิธีที่ 3: Sealed Secrets (GitOps)](#วิธีที่-3-sealed-secrets-gitops)
  - [Monitoring \& Observability](#monitoring--observability)
    - [Dashboard ที่แนะนำ](#dashboard-ที่แนะนำ)
  - [Troubleshooting](#troubleshooting)
    - [Pod ไม่ขึ้น](#pod-ไม่ขึ้น)
    - [ปัญหาที่พบบ่อย](#ปัญหาที่พบบ่อย)
    - [Debug เร็ว](#debug-เร็ว)
  - [สิ่งที่ต้องเปลี่ยนก่อน Deploy จริง](#สิ่งที่ต้องเปลี่ยนก่อน-deploy-จริง)

---

## โครงสร้างไฟล์

```
deployments/k8s/
├── base/                          ← Shared manifests (Kustomize base)
│   ├── kustomization.yaml         ← Kustomize entry point
│   ├── namespace.yaml             ← Namespace: anc-portal
│   ├── configmap.yaml             ← Environment variables (non-secret)
│   ├── secret.yaml                ← Sensitive values (placeholder)
│   ├── api-deployment.yaml        ← API server pods
│   ├── api-service.yaml           ← ClusterIP Service (port 80 → 20000)
│   ├── api-hpa.yaml               ← HorizontalPodAutoscaler
│   ├── api-pdb.yaml               ← PodDisruptionBudget
│   ├── api-ingress.yaml           ← Ingress (NGINX) — ⚠️ แก้ host ก่อนใช้
│   ├── worker-deployment.yaml     ← Kafka consumer pods
│   ├── migrate-job.yaml           ← Database migration Job
│   └── sync-cronjob.yaml          ← Sync CronJob (ทุก 6 ชม.)
│
└── overlays/                      ← Environment-specific overrides
    ├── staging/
    │   └── kustomization.yaml     ← Staging: 2-4 pods, swagger on
    └── production/
        └── kustomization.yaml     ← Production: 3-8 pods, PDB=2, high resources
```

---

## Prerequisites

| เครื่องมือ | เวอร์ชัน | หมายเหตุ |
|---|---|---|
| kubectl | ≥ 1.28 | K8s CLI |
| kustomize | ≥ 5.0 (built-in kubectl) | Template overlays |
| Docker | ≥ 24.0 | Build images |
| Kubernetes Cluster | ≥ 1.28 | GKE / EKS / AKS / local |
| Metrics Server | ติดตั้งแล้ว | จำเป็นสำหรับ HPA |

---

## Quick Start

```bash
# 1. Build & push images (multi-target Dockerfile)
docker build -f deployments/docker/Dockerfile -t ghcr.io/onizukazaza/anc-portal-be-fake:staging .
docker build -f deployments/docker/Dockerfile --target worker -t ghcr.io/anc-portal/anc-portal-worker:staging .
docker push ghcr.io/onizukazaza/anc-portal-be-fake:staging
docker push ghcr.io/anc-portal/anc-portal-worker:staging

# 2. Preview manifests (dry-run)
kubectl kustomize deployments/k8s/overlays/staging

# 3. Deploy staging
kubectl apply -k deployments/k8s/overlays/staging

# 4. ดูสถานะ
kubectl get all -n anc-portal
```

---

## Resource Overview

| Resource | ชื่อ | หน้าที่ |
|---|---|---|
| **Namespace** | `anc-portal` | แยก resources ออกจาก namespace อื่น |
| **ConfigMap** | `anc-portal-config` | env ทั้งหมดที่ไม่ใช่ secret (DB host, Redis, Kafka, OTel, etc.) |
| **Secret** | `anc-portal-secret` | DB password, JWT secret key |
| **Deployment** | `anc-portal-api` | API server (Fiber HTTP, port 20000) |
| **Service** | `anc-portal-api` | ClusterIP เปิด port 80 → target 20000 |
| **HPA** | `anc-portal-api` | Auto-scale ตาม CPU/Memory |
| **Deployment** | `anc-portal-worker` | Kafka consumer process |
| **Job** | `anc-portal-migrate` | รัน `cmd/migrate` ครั้งเดียว (one-time) |
| **CronJob** | `anc-portal-sync` | Sync external DB → main DB (ทุก 6 ชม.) |
| **PDB** | `anc-portal-api` | ป้องกัน disruption ระหว่าง node drain |
| **Ingress** | `anc-portal-api` | Expose API ออกนอก cluster (NGINX) |

---

## Deploy ทีละขั้นตอน

### ขั้นตอนที่ 1: สร้าง Namespace

```bash
kubectl apply -f deployments/k8s/base/namespace.yaml
```

### ขั้นตอนที่ 2: สร้าง Secret (ใส่ค่าจริง)

```bash
# แก้ค่าใน secret.yaml หรือสร้างจาก command line:
kubectl create secret generic anc-portal-secret \
  --namespace=anc-portal \
  --from-literal=DB_USER=anc_app \
  --from-literal=DB_PASSWORD='<รหัสผ่านจริง>' \
  --from-literal=REDIS_PASSWORD='<รหัสผ่านจริง>' \
  --from-literal=JWT_SECRET_KEY='<secret จริง>'
```

### ขั้นตอนที่ 3: รัน Migration

```bash
kubectl apply -f deployments/k8s/base/migrate-job.yaml

# รอให้เสร็จ
kubectl wait --for=condition=complete job/anc-portal-migrate -n anc-portal --timeout=120s

# ดู log
kubectl logs -n anc-portal job/anc-portal-migrate
```

### ขั้นตอนที่ 4: Deploy ทั้งหมด

```bash
# Staging
kubectl apply -k deployments/k8s/overlays/staging

# Production
kubectl apply -k deployments/k8s/overlays/production
```

### ขั้นตอนที่ 5: ตรวจสอบ

```bash
# ดู pods
kubectl get pods -n anc-portal -w

# ดู API logs
kubectl logs -n anc-portal -l app.kubernetes.io/name=anc-portal-api -f

# ดู Worker logs
kubectl logs -n anc-portal -l app.kubernetes.io/name=anc-portal-worker -f

# ทดสอบ health
kubectl port-forward -n anc-portal svc/anc-portal-api 8080:80
curl http://localhost:8080/healthz
curl http://localhost:8080/ready
```

---

## Overlay: Staging vs Production

| ค่า | Staging | Production |
|---|---|---|
| **API replicas** | 2 | 3 |
| **HPA min/max** | 2–4 | 3–8 |
| **Worker replicas** | 1 | 2 |
| **CPU request/limit** | 100m / 500m | 200m / 1000m |
| **Memory request/limit** | 128Mi / 256Mi | 256Mi / 512Mi |
| **DB max connections** | 20 | 30 |
| **OTel sample ratio** | 50% | 5% |
| **Swagger** | เปิด | ปิด |
| **CORS origins** | `*` | `https://portal.anc.co.th` |
| **Image tag** | `staging` | `v1.0.0` |
| **PDB minAvailable** | 1 | 2 |
| **Ingress host** | `api-staging.anc-portal.example.com` | `api.portal.anc.co.th` |

---

## Database Migration

Migration ใช้ Kubernetes Job — รันครั้งเดียวก่อน deploy API:

```bash
# รัน migration up
kubectl apply -f deployments/k8s/base/migrate-job.yaml

# ดู status
kubectl get jobs -n anc-portal

# ดู log
kubectl logs -n anc-portal job/anc-portal-migrate

# ลบ job เก่า (ก่อนรัน migration ใหม่)
kubectl delete job anc-portal-migrate -n anc-portal
```

> **หมายเหตุ:** Job มี `ttlSecondsAfterFinished: 300` — จะถูกลบอัตโนมัติหลังสำเร็จ 5 นาที

### Migration ใน CI/CD Pipeline

```yaml
# ตัวอย่าง GitHub Actions step
- name: Run migration
  run: |
    kubectl apply -f deployments/k8s/base/migrate-job.yaml
    kubectl wait --for=condition=complete job/anc-portal-migrate -n anc-portal --timeout=120s
```

---

## Scaling

### Auto-scaling (HPA)

HPA ตั้งค่าไว้แล้วใน `api-hpa.yaml`:

```bash
# ดู HPA status
kubectl get hpa -n anc-portal

# ดู metrics จริง
kubectl describe hpa anc-portal-api -n anc-portal
```

| Metric | Target | Scale Up | Scale Down |
|---|---|---|---|
| CPU | 70% avg | +2 pods / 60s | -1 pod / 120s |
| Memory | 80% avg | +2 pods / 60s | -1 pod / 120s |

### Manual scaling

```bash
# Scale API
kubectl scale deployment anc-portal-api -n anc-portal --replicas=4

# Scale Worker (เพิ่ม consumer)
kubectl scale deployment anc-portal-worker -n anc-portal --replicas=3
```

---

## Health Checks & Probes

API Deployment มี 3 probes ที่ map กับ endpoint ในโปรเจค:

| Probe | Endpoint | เงื่อนไข | หน้าที่ |
|---|---|---|---|
| **startupProbe** | `GET /healthz` | fail 10 ครั้ง → restart | รอ app boot (DB connect, etc.) |
| **livenessProbe** | `GET /healthz` | fail 3 ครั้ง → restart | ตรวจว่า process ยัง alive |
| **readinessProbe** | `GET /ready` | fail 2 ครั้ง → ถอดจาก Service | ตรวจว่าพร้อมรับ traffic |

> `/healthz` ตรวจ: DB + Redis  
> `/ready` ตรวจ: DB + Redis + return timestamp

### Worker Probes

Worker มี HTTP health probe server บน port 20001 — ตรวจสถานะ Kafka consumer:

| Probe | Endpoint | เงื่อนไข | หน้าที่ |
|---|---|---|---|
| **livenessProbe** | `GET /healthz` (port 20001) | fail 3 ครั้ง → restart | ตรวจว่า consumer ยัง alive |
| **readinessProbe** | `GET /healthz` (port 20001) | fail 2 ครั้ง → ถอดจาก Service | ตรวจว่า consumer พร้อมรับ messages |

> `/healthz` ตรวจ: `consumer.IsHealthy()` (atomic.Bool — true หลัง first successful fetch)

---

## Secrets Management

### วิธีที่ 1: kubectl (Dev/Staging)

```bash
kubectl create secret generic anc-portal-secret \
  --namespace=anc-portal \
  --from-literal=DB_USER=anc_app \
  --from-literal=DB_PASSWORD='my-password' \
  --from-literal=JWT_SECRET_KEY='my-jwt-secret'
```

### วิธีที่ 2: External Secrets Operator (Production — แนะนำ)

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: anc-portal-secret
  namespace: anc-portal
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager  # หรือ gcp-secret-manager, azure-keyvault
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
```

### วิธีที่ 3: Sealed Secrets (GitOps)

```bash
kubeseal --format=yaml < deployments/k8s/base/secret.yaml > sealed-secret.yaml
```

---

## Monitoring & Observability

โปรเจคมี OTel + Prometheus built-in — เพิ่ม ServiceMonitor เพื่อให้ Prometheus scrape:

```yaml
# เพิ่มถ้าใช้ Prometheus Operator
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: anc-portal-api
  namespace: anc-portal
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: anc-portal-api
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
```

### Dashboard ที่แนะนำ

| เครื่องมือ | หน้าที่ | Endpoint |
|---|---|---|
| **Prometheus** | Metrics scraping | `GET /metrics` |
| **Grafana** | Dashboard visualization | Connect to Prometheus |
| **Tempo** | Distributed tracing | OTel exporter → Tempo |
| **Loki** | Log aggregation | stdout → Loki |

---

## Troubleshooting

### Pod ไม่ขึ้น

```bash
# ดู events
kubectl describe pod <pod-name> -n anc-portal

# ดู logs
kubectl logs <pod-name> -n anc-portal --previous

# ดู events ทั้ง namespace
kubectl get events -n anc-portal --sort-by='.lastTimestamp'
```

### ปัญหาที่พบบ่อย

| อาการ | สาเหตุ | แก้ไข |
|---|---|---|
| `CrashLoopBackOff` | DB connect ไม่ได้ | ตรวจ ConfigMap: DB_HOST, DB_PORT + Secret: DB_PASSWORD |
| `ImagePullBackOff` | Image ไม่เจอ | ตรวจ image name + imagePullSecrets |
| Readiness fail | Redis ไม่พร้อม | ตรวจ REDIS_HOST + REDIS_PASSWORD |
| HPA ไม่ scale | ไม่มี Metrics Server | `kubectl top pods` ถ้า error = ติดตั้ง metrics-server |
| OOMKilled | Memory ไม่พอ | เพิ่ม `resources.limits.memory` |

### Debug เร็ว

```bash
# เข้า shell ใน pod
kubectl exec -it <pod-name> -n anc-portal -- /bin/sh

# Port-forward ดู API ตรง
kubectl port-forward -n anc-portal svc/anc-portal-api 8080:80

# ดู config ที่ inject เข้า pod
kubectl exec <pod-name> -n anc-portal -- env | sort
```

---

## สิ่งที่ต้องเปลี่ยนก่อน Deploy จริง

| # | ไฟล์ | สิ่งที่ต้องเปลี่ยน |
|---|---|---|
| 1 | `base/secret.yaml` | ใส่ DB_PASSWORD, JWT_SECRET_KEY จริง หรือใช้ External Secrets |
| 2 | `base/api-deployment.yaml` | เปลี่ยน `image:` เป็น registry จริง |
| 3 | `base/worker-deployment.yaml` | เปลี่ยน `image:` เป็น registry จริง |
| 4 | `base/configmap.yaml` | ใส่ DB_HOST, REDIS_HOST ที่ถูกต้องตาม cluster |
| 5 | `overlays/production/` | ใส่ CORS domain จริง, image tag จริง |
| 6 | `base/api-ingress.yaml` | เปลี่ยน host เป็น domain จริง + สร้าง TLS secret |
| 7 | (เพิ่มเอง) | imagePullSecret ถ้าใช้ private registry |

---

> **v2.0** — March 2026 | ANC Portal Backend Team
