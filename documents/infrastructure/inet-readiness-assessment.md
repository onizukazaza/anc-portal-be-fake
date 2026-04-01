# INET Cloud Readiness Assessment — ANC Portal Backend

> ประเมินความพร้อมของระบบ ANC Portal Backend สำหรับการ deploy บน **INET (Internet Thailand PCL)**
>
> วันที่ประเมิน: เมษายน 2026

---

## สารบัญ

1. [สรุปผลประเมิน (Executive Summary)](#1-สรุปผลประเมิน)
2. [ข้อมูลระบบ](#2-ข้อมูลระบบ)
3. [บริการ INET ที่เกี่ยวข้อง](#3-บริการ-inet-ที่เกี่ยวข้อง)
4. [Resource Sizing — ประเมินทรัพยากร](#4-resource-sizing)
5. [Mapping: ความต้องการระบบ → บริการ INET](#5-mapping-ความต้องการระบบ--บริการ-inet)
6. [Deployment Options](#6-deployment-options)
7. [Checklist ความพร้อม](#7-checklist-ความพร้อม)
8. [สิ่งที่ต้องเตรียมเพิ่มเติม](#8-สิ่งที่ต้องเตรียมเพิ่มเติม)
9. [Risk & Limitation](#9-risk--limitation)
10. [แผนดำเนินการแนะนำ](#10-แผนดำเนินการแนะนำ)

---

## 1. สรุปผลประเมิน

| หมวด | สถานะ | หมายเหตุ |
|------|--------|----------|
| Containerization | ✅ พร้อม | Multi-stage Dockerfile, Alpine-based, non-root |
| Kubernetes Manifests | ✅ พร้อม | Kustomize base + overlays (staging/production) |
| Database | ✅ พร้อม | PostgreSQL 17, มี migration + seed |
| CI/CD Pipeline | ✅ พร้อม | GitHub Actions 7 stages + deploy workflows |
| Observability | ✅ พร้อม | OpenTelemetry + Prometheus metrics (toggle) |
| Security | ✅ พร้อม | Non-root, TLS Ingress, rate limiting |
| TLS/SSL Certificate | ⚠️ ต้องเตรียม | ต้องจัด cert สำหรับ domain จริง |
| Domain Name | ⚠️ ต้องเตรียม | ต้องซื้อ/ชี้ domain มาที่ INET |
| Secrets Management | ⚠️ ต้องเตรียม | ค่า placeholder ต้องเปลี่ยนเป็นค่าจริง |
| Container Registry | ⚠️ ต้องตัดสินใจ | ใช้ GHCR หรือ INET private registry |

**ผลประเมินรวม: ระบบมีความพร้อม ~80%** — ส่วน core (code, Docker, K8s) พร้อมครบ เหลือเฉพาะ infrastructure provisioning บน INET

---

## 2. ข้อมูลระบบ

### 2.1 Application Stack

| Component | Technology | Version |
|-----------|-----------|---------|
| Language | Go | 1.25 |
| Web Framework | Fiber | v2.52.12 |
| Database | PostgreSQL | 17 |
| Cache | Redis | 7 |
| Message Queue | Apache Kafka (KRaft) | 3.9.0 |
| Observability | OpenTelemetry + Prometheus + Grafana + Tempo | - |
| Container Runtime | Docker (Alpine-based) | - |
| Orchestration | Kubernetes + Kustomize | - |

### 2.2 Binaries ที่ build ได้

| Binary | คำอธิบาย | Port |
|--------|----------|------|
| `api` | HTTP API server (Fiber) | 20000 |
| `worker` | Background Kafka consumer | 20001 (health) |
| `migrate` | Database migration runner | - |
| `seed` | Seed data loader | - |
| `sync` | External DB sync (CronJob) | - |

### 2.3 Docker Image

- **Base Image:** `alpine:3.21` (runtime ~7 MB)
- **Builder:** `golang:1.25-alpine`
- **ขนาดโดยประมาณ:** API image ~30-50 MB, Worker image ~15-25 MB
- **Security:** non-root user (UID 10001), `readOnlyRootFilesystem`, `allowPrivilegeEscalation: false`

### 2.4 Feature Toggles

Redis, Kafka, OpenTelemetry สามารถ **ปิดได้** ผ่าน environment variables:

```env
REDIS_ENABLED=false      # ไม่ต้องมี Redis
KAFKA_ENABLED=false      # ไม่ต้องมี Kafka
OTEL_ENABLED=false       # ไม่ต้องมี OTel
```

> **Minimum deployment:** เฉพาะ API + PostgreSQL (ไม่ต้องมี Redis/Kafka/OTel)

---

## 3. บริการ INET ที่เกี่ยวข้อง

INET (inet.co.th) เป็น Cloud Service Provider อันดับ 1 ของไทย ให้บริการ:

| บริการ INET | ประเภท | เกี่ยวข้องกับระบบ |
|-------------|--------|-------------------|
| **Enterprise Cloud (VM)** | IaaS | ✅ รัน K8s cluster หรือ Docker |
| **Opensource Cloud** | IaaS | ✅ ทางเลือกราคาประหยัด |
| **Container as a Service** | CaaS | ✅ **เหมาะสมที่สุด** — รัน container โดยตรง |
| **Database as a Service (DBaaS)** | PaaS | ✅ PostgreSQL managed service |
| **Backup as a Service** | BaaS | ✅ สำรองข้อมูล DB + volumes |
| **Disaster Recovery as a Service** | DRaaS | ✅ HA / failover |
| **INET S3** | Storage | ⬜ อาจใช้เก็บ documents/attachments ในอนาคต |
| **Cyber Security** | Security | ⬜ เสริม WAF/DDoS protection |

### มาตรฐานที่ INET ผ่าน (เกี่ยวข้องกับ compliance)

- ISO/IEC 27001:2022 — Information Security Management
- ISO 27017:2015 — Cloud Security
- ISO 27018:2019 — PII Protection in Cloud
- ISO 22301:2019 — Business Continuity
- PCI-DSS — Payment Card Industry Security
- SOC2 Type II — Data Privacy & Security
- CSA-STAR — Cloud Security Alliance
- SLA 99.95% guarantee

---

## 4. Resource Sizing

### 4.1 Staging Environment

| Component | Replicas | CPU Request | CPU Limit | Memory Request | Memory Limit |
|-----------|----------|-------------|-----------|----------------|--------------|
| API | 2 | 100m | 500m | 128Mi | 256Mi |
| Worker | 1 | 50m | 250m | 64Mi | 128Mi |
| **Sync CronJob** | - | ตาม worker | ตาม worker | ตาม worker | ตาม worker |

**รวม Staging (minimum):**
- CPU: ~250m request, ~1250m limit
- Memory: ~320Mi request, ~640Mi limit
- HPA: scale ได้ถึง 4 API pods

### 4.2 Production Environment

| Component | Replicas | CPU Request | CPU Limit | Memory Request | Memory Limit |
|-----------|----------|-------------|-----------|----------------|--------------|
| API | 3 | 200m | 1000m | 256Mi | 512Mi |
| Worker | 2 | 50m | 250m | 64Mi | 128Mi |
| **Sync CronJob** | - | ตาม worker | ตาม worker | ตาม worker | ตาม worker |

**รวม Production (minimum):**
- CPU: ~700m request, ~3500m limit
- Memory: ~896Mi request, ~1792Mi limit
- HPA: scale ได้ถึง 8 API pods
- PDB: เก็บ pod พร้อมใช้อย่างน้อย 2 ตัวเสมอ

### 4.3 Infrastructure Dependencies

| Service | Staging Spec (แนะนำ) | Production Spec (แนะนำ) |
|---------|----------------------|------------------------|
| PostgreSQL 17 | 2 vCPU, 4 GB RAM, 50 GB SSD | 4 vCPU, 8 GB RAM, 100 GB SSD + replica |
| Redis 7 | 1 vCPU, 1 GB RAM | 2 vCPU, 2 GB RAM |
| Kafka 3.9 | 2 vCPU, 2 GB RAM, 20 GB | 2 vCPU, 4 GB RAM, 50 GB (optional) |
| OTel Collector | 1 vCPU, 512 MB RAM | 2 vCPU, 1 GB RAM |

> **Minimum (ปิด Redis/Kafka/OTel):** เฉพาะ K8s nodes + PostgreSQL

---

## 5. Mapping: ความต้องการระบบ → บริการ INET

### Option A: Container as a Service (แนะนำ)

```
┌─────────────────────────────────────────────┐
│            INET Container Service           │
│                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │ API Pod  │  │ API Pod  │  │ API Pod  │  │
│  │ (Fiber)  │  │ (Fiber)  │  │ (Fiber)  │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  │
│       └──────────────┼──────────────┘       │
│                      │                      │
│  ┌──────────┐  ┌─────┴────┐                 │
│  │ Worker   │  │ CronJob  │                 │
│  │ (Kafka)  │  │ (Sync)   │                 │
│  └──────────┘  └──────────┘                 │
└──────────────────┬──────────────────────────┘
                   │
    ┌──────────────┼──────────────┐
    ▼              ▼              ▼
┌────────┐  ┌──────────┐  ┌──────────┐
│  INET  │  │   INET   │  │  Redis   │
│  DBaaS │  │  S3/BaaS │  │ (on VM)  │
│ (PG17) │  │  backup  │  │          │
└────────┘  └──────────┘  └──────────┘
```

**ข้อดี:**
- INET จัดการ container orchestration ให้
- ใช้ K8s manifests ที่มีอยู่ได้เลย (Kustomize)
- Auto-scaling, health checks พร้อม

### Option B: Enterprise Cloud VM + self-managed K8s

```
┌─────────────────────────────────────────────┐
│         INET Enterprise Cloud (VM)          │
│                                             │
│  ┌──────────────────────────────────────┐   │
│  │      K8s Cluster (self-managed)      │   │
│  │                                      │   │
│  │  Master Node (1x) + Worker Nodes     │   │
│  │  ┌─────┐ ┌─────┐ ┌──────┐ ┌─────┐  │   │
│  │  │ API │ │ API │ │Worker│ │Sync │  │   │
│  │  └─────┘ └─────┘ └──────┘ └─────┘  │   │
│  └──────────────────────────────────────┘   │
│                                             │
│  ┌────────────┐  ┌───────┐  ┌───────┐      │
│  │ PostgreSQL │  │ Redis │  │ Kafka │      │
│  │    VM      │  │  VM   │  │  VM   │      │
│  └────────────┘  └───────┘  └───────┘      │
└─────────────────────────────────────────────┘
```

**ข้อดี:**
- ควบคุมได้เต็มที่ (custom K8s config)
- เหมาะถ้ามีหลาย service ในอนาคต

**ข้อเสีย:**
- ต้องดูแล K8s cluster เอง (upgrade, patching)
- ต้องการคนดูแล infra

### Option C: VM + Docker Compose (เริ่มต้นง่าย)

```
┌─────────────────────────────────────────────┐
│        INET Cloud VM (1 เครื่อง)            │
│                                             │
│  docker-compose up                          │
│  ┌─────┐ ┌──────┐ ┌────────────┐ ┌───────┐ │
│  │ API │ │Worker│ │ PostgreSQL │ │ Redis │ │
│  └─────┘ └──────┘ └────────────┘ └───────┘ │
│                                             │
└─────────────────────────────────────────────┘
```

**ข้อดี:**
- เริ่มต้นง่ายที่สุด ใช้ VM เครื่องเดียว
- ค่าใช้จ่ายต่ำสุด

**ข้อเสีย:**
- ไม่มี auto-scaling, HA
- Single point of failure

---

## 6. Deployment Options

### เปรียบเทียบ 3 ทางเลือก

| เกณฑ์ | Option A: CaaS | Option B: VM + K8s | Option C: VM + Compose |
|-------|---------------|-------------------|----------------------|
| **ความยาก** | ปานกลาง | ยาก | ง่าย |
| **Auto-scaling** | ✅ ได้ | ✅ ได้ (ตั้งเอง) | ❌ ไม่ได้ |
| **High Availability** | ✅ ได้ | ✅ ได้ (ตั้งเอง) | ❌ ไม่ได้ |
| **ค่าใช้จ่าย** | ปานกลาง | สูง | ต่ำ |
| **ใช้ K8s manifests ที่มี** | ✅ ใช้ได้เลย | ✅ ใช้ได้เลย | ❌ ใช้ compose แทน |
| **ดูแลง่าย** | ✅ INET ดูแล infra | ❌ ดูแลเอง | ✅ ง่ายแต่จำกัด |
| **เหมาะกับ** | Production ที่ต้องการ HA | Large-scale, multi-service | Dev/Staging, MVP |

### คำแนะนำ

| สถานการณ์ | ทางเลือกแนะนำ |
|-----------|--------------|
| เริ่มต้น / Staging / ทดสอบก่อน | **Option C** — VM 1 เครื่อง + Docker Compose |
| Production ทั่วไป | **Option A** — Container as a Service |
| Enterprise / multi-team / compliance สูง | **Option B** — VM + self-managed K8s |

---

## 7. Checklist ความพร้อม

### 7.1 ✅ สิ่งที่พร้อมแล้ว

- [x] Multi-stage Dockerfile (API target + Worker target)
- [x] Docker image ขนาดเล็ก (~30-50 MB) Alpine-based
- [x] Non-root container (UID 10001)
- [x] Security context (no privilege escalation)
- [x] Kubernetes manifests ครบ (Deployment, Service, Ingress, HPA, PDB, CronJob, ConfigMap, Secret)
- [x] Kustomize overlays สำหรับ staging และ production
- [x] Health check endpoints (`/healthz`, `/ready`)
- [x] HEALTHCHECK ใน Dockerfile
- [x] Rolling update strategy (maxSurge:1, maxUnavailable:0)
- [x] Environment-based configuration (ConfigMap + Secret)
- [x] Feature toggles สำหรับ Redis, Kafka, OTel
- [x] CI/CD pipeline (GitHub Actions: lint, test, vuln scan, build, docker, security scan)
- [x] Database migration system (`cmd/migrate`)
- [x] Seed data system (`cmd/seed`)
- [x] Rate limiting Ingress annotations
- [x] TLS/SSL redirect configured
- [x] CORS configuration
- [x] Graceful shutdown (`terminationGracePeriodSeconds`)

### 7.2 ⚠️ สิ่งที่ต้องเตรียมก่อน deploy จริง

- [ ] Domain name จริง (แทน `api.anc-portal.example.com`)
- [ ] TLS certificate สำหรับ domain (cert-manager หรือ manual)
- [ ] ค่า Secret จริง (`DB_PASSWORD`, `JWT_SECRET_KEY`, `REDIS_PASSWORD`)
- [ ] Container registry ที่ INET เข้าถึงได้ (GHCR public หรือ INET private registry)
- [ ] PostgreSQL instance บน INET (DBaaS หรือ VM)
- [ ] Network firewall rules (allow port 20000, DB port)
- [ ] DNS record ชี้ domain → INET IP
- [ ] Backup policy สำหรับ database
- [ ] Monitoring/Alerting setup (ถ้าเปิด OTel)
- [ ] CI/CD credentials (`KUBE_CONFIG_STAGING` / `KUBE_CONFIG_PRODUCTION`)

---

## 8. สิ่งที่ต้องเตรียมเพิ่มเติม

### 8.1 Domain & DNS

```
ซื้อ domain → ชี้ A record / CNAME → INET Load Balancer IP
ตัวอย่าง: api.portal.anc.co.th → 203.xxx.xxx.xxx
```

### 8.2 TLS Certificate

มี 2 ทางเลือก:
1. **cert-manager + Let's Encrypt** — ฟรี, auto-renew (ต้องติดตั้งใน K8s)
2. **ซื้อ SSL cert** แล้วใส่เป็น K8s Secret — ง่ายกว่า, เสียค่าใช้จ่าย

### 8.3 Secrets Management

ค่าปัจจุบันเป็น placeholder — ก่อน deploy ต้อง:

```yaml
# เปลี่ยนจาก
DB_PASSWORD: "CHANGE_ME"
JWT_SECRET_KEY: "CHANGE_ME"

# เป็นค่าจริง (ใช้ External Secrets Operator หรือ manual)
DB_PASSWORD: "<strong_random_password>"
JWT_SECRET_KEY: "<strong_random_jwt_secret>"
```

### 8.4 Container Registry Access

INET cluster ต้องสามารถ pull image ได้:
- **GHCR (public repo):** ไม่ต้องตั้งค่าเพิ่ม
- **GHCR (private repo):** ต้องสร้าง `imagePullSecret` ใน K8s
- **INET private registry:** push image ไป INET registry แทน

### 8.5 CI/CD Integration กับ INET

```yaml
# GitHub Actions → deploy ไป INET K8s
# ต้องตั้ง GitHub Secrets:
KUBE_CONFIG_STAGING: <base64 kubeconfig จาก INET staging>
KUBE_CONFIG_PRODUCTION: <base64 kubeconfig จาก INET production>
```

---

## 9. Risk & Limitation

| ความเสี่ยง | ระดับ | วิธีลด |
|------------|-------|-------|
| INET ไม่มี managed K8s (ต้อง self-manage) | ปานกลาง | ใช้ Container as a Service หรือ VM + k3s |
| Kafka ไม่มี managed service บน INET | ต่ำ | ปิด Kafka ได้ (`KAFKA_ENABLED=false`) หรือรันเป็น container |
| Vendor lock-in | ต่ำ | ระบบเป็น standard Docker/K8s, ย้ายได้ทุก cloud |
| Network latency (ถ้า external DB อยู่ต่าง DC) | ปานกลาง | ใช้ INET DC เดียวกัน หรือ VPN |
| ค่าใช้จ่าย scale-up ตอน peak | ปานกลาง | ตั้ง HPA maxReplicas + resource limits |

---

## 10. แผนดำเนินการแนะนำ

### Phase 1: Staging (สัปดาห์ที่ 1-2)

```
1. สมัครบริการ INET Cloud (VM หรือ Container Service)
2. สร้าง PostgreSQL instance (DBaaS หรือ VM)
3. สร้าง VM / Container cluster สำหรับ staging
4. ตั้ง domain + DNS (staging subdomain)
5. Push Docker image ไป registry
6. Deploy ด้วย Kustomize staging overlay
7. ทดสอบ healthz + API endpoints
```

### Phase 2: Production (สัปดาห์ที่ 3-4)

```
1. สร้าง production PostgreSQL (+ replica)
2. ตั้งค่า TLS certificate
3. Deploy ด้วย Kustomize production overlay
4. ตั้ง HPA + PDB
5. ตั้ง backup policy (INET Backup as a Service)
6. เปิด Redis + OTel (ถ้าต้องการ)
7. ตั้ง CI/CD credentials สำหรับ auto-deploy
```

### Phase 3: Optimization (สัปดาห์ที่ 5+)

```
1. เปิด Kafka (ถ้าต้องการ event-driven)
2. ตั้ง Disaster Recovery (INET DRaaS)
3. ตั้ง monitoring alerting (Grafana + Tempo)
4. Performance tuning (DB connection pool, HPA thresholds)
5. Security hardening (WAF, DDoS protection)
```

---

## INET Spec แนะนำ (ประมาณการ)

### Staging (Minimum)

| Resource | Spec | ประมาณการ |
|----------|------|----------|
| VM / Container Service | 2 vCPU, 4 GB RAM | สำหรับรัน API + Worker |
| PostgreSQL (DBaaS) | 2 vCPU, 4 GB RAM, 50 GB SSD | Main database |
| Storage | 20 GB SSD | Docker images + logs |
| Bandwidth | 100 Mbps | ตามแพ็กเกจ |

### Production (แนะนำ)

| Resource | Spec | ประมาณการ |
|----------|------|----------|
| Container / K8s Nodes | 4 vCPU, 8 GB RAM (x2 nodes) | สำหรับ 3-8 API pods + 2 Worker |
| PostgreSQL (DBaaS) | 4 vCPU, 8 GB RAM, 100 GB SSD + replica | HA database |
| Redis VM | 2 vCPU, 2 GB RAM | Cache layer |
| Storage | 50 GB SSD | Images + logs + volumes |
| Bandwidth | 200 Mbps+ | ตามแพ็กเกจ |
| Backup | INET BaaS | Daily DB backup + 7 day retention |

---

> **สรุป:** ระบบ ANC Portal Backend มีความพร้อมสูงสำหรับ deploy บน INET Cloud
> Docker image, K8s manifests, CI/CD pipeline พร้อมครบ
> สิ่งที่ต้องเตรียมเพิ่มเติมเป็นเรื่อง infrastructure provisioning (domain, TLS, secrets, DB)
> แนะนำเริ่มจาก **Container as a Service + DBaaS** สำหรับ production
> หรือ **VM + Docker Compose** สำหรับทดสอบก่อน deploy จริง
