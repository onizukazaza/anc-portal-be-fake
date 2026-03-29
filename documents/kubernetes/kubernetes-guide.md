# Kubernetes (K8s) Complete Guide — ANC Portal Backend

> **v1.0** — Last updated: March 2026
>
> คู่มือ Kubernetes ฉบับสมบูรณ์ — เขียนสำหรับผู้เริ่มต้น
> ครอบคลุมตั้งแต่แนวคิดพื้นฐาน, ศัพท์เฉพาะทาง, ไปจนถึงการ Deploy จริงบน Cluster
>
> อ้างอิงจาก config จริงของโปรเจค ANC Portal Backend

---

## สารบัญ

1. [Kubernetes คืออะไร?](#1-kubernetes-คืออะไร)
2. [ศัพท์เฉพาะทาง (Glossary)](#2-ศัพท์เฉพาะทาง-glossary)
3. [เครื่องมือที่ต้องติดตั้ง (Prerequisites)](#3-เครื่องมือที่ต้องติดตั้ง-prerequisites)
4. [สถาปัตยกรรม K8s ของโปรเจค](#4-สถาปัตยกรรม-k8s-ของโปรเจค)
5. [โครงสร้างไฟล์ K8s ในโปรเจค](#5-โครงสร้างไฟล์-k8s-ในโปรเจค)
6. [Kustomize คืออะไร และใช้ยังไง](#6-kustomize-คืออะไร-และใช้ยังไง)
7. [อธิบายไฟล์ Base ทีละไฟล์](#7-อธิบายไฟล์-base-ทีละไฟล์)
   - [7.1 Namespace](#71-namespace)
   - [7.2 ConfigMap](#72-configmap)
   - [7.3 Secret](#73-secret)
   - [7.4 Deployment — API Server](#74-deployment--api-server)
   - [7.5 Service](#75-service)
   - [7.6 HorizontalPodAutoscaler (HPA)](#76-horizontalpodautoscaler-hpa)
   - [7.7 PodDisruptionBudget (PDB)](#77-poddisruptionbudget-pdb)
   - [7.8 Ingress](#78-ingress)
   - [7.9 Deployment — Worker](#79-deployment--worker)
   - [7.10 Job — Migration](#710-job--migration)
   - [7.11 CronJob — Sync](#711-cronjob--sync)
8. [Overlays — การตั้งค่าตาม Environment](#8-overlays--การตั้งค่าตาม-environment)
9. [Docker Image — การ Build](#9-docker-image--การ-build)
10. [คำสั่ง kubectl ที่ใช้บ่อย](#10-คำสั่ง-kubectl-ที่ใช้บ่อย)
11. [ขั้นตอนการ Deploy ตั้งแต่เริ่มต้น (Step-by-Step)](#11-ขั้นตอนการ-deploy-ตั้งแต่เริ่มต้น-step-by-step)
12. [Health Checks & Probes อธิบาย](#12-health-checks--probes-อธิบาย)
13. [Resource Management — CPU & Memory](#13-resource-management--cpu--memory)
14. [Scaling — การขยายระบบ](#14-scaling--การขยายระบบ)
15. [Secrets Management — จัดการข้อมูลลับ](#15-secrets-management--จัดการข้อมูลลับ)
16. [Monitoring & Observability](#16-monitoring--observability)
17. [Rollback — ย้อนกลับเวอร์ชัน](#17-rollback--ย้อนกลับเวอร์ชัน)
18. [Troubleshooting — แก้ปัญหา](#18-troubleshooting--แก้ปัญหา)
19. [ตารางเปรียบเทียบ Spec ทุก Environment](#19-ตารางเปรียบเทียบ-spec-ทุก-environment)
20. [Checklist ก่อน Deploy](#20-checklist-ก่อน-deploy)

---

## 1. Kubernetes คืออะไร?

**Kubernetes** (เรียกย่อว่า **K8s**) คือระบบ **Container Orchestration** ที่ช่วยจัดการ container (เช่น Docker container) โดยอัตโนมัติ

### ทำไมต้องใช้ K8s?

| ปัญหาเดิม (ไม่มี K8s) | K8s ช่วยได้ |
|---|---|
| App crash → ต้องรัน manual | **Self-healing** — restart ให้อัตโนมัติ |
| Traffic พุ่ง → server รับไม่ไหว | **Auto-scaling** — เพิ่ม pod อัตโนมัติ |
| Deploy เวอร์ชันใหม่ → downtime | **Rolling Update** — ค่อยๆ เปลี่ยนทีละตัว |
| Config ต่างกันในแต่ละ environment | **ConfigMap / Secret** — จัดการแยกจาก code |
| Container กระจายหลาย server | **Service Discovery** — หากัน เชื่อมกันได้ |

### ภาพรวมอย่างง่าย

```
Developer → push code → Build Docker Image → Push to Registry → K8s ดึง image → สร้าง Pod → รับ traffic

                                 ┌──────────────────────┐
   Internet ──▶ Ingress ──▶ Service ──▶ Pod 1 (API)
                                       ──▶ Pod 2 (API)
                                       ──▶ Pod 3 (API)
                                 └──────────────────────┘
```

---

## 2. ศัพท์เฉพาะทาง (Glossary)

### 2.1 Core Concepts (แนวคิดหลัก)

| ศัพท์ | อ่านว่า | ความหมาย |
|---|---|---|
| **Cluster** | คลัสเตอร์ | กลุ่มของเครื่อง (Node) ที่ K8s จัดการ ทำงานร่วมกันเสมือนเครื่องเดียว |
| **Node** | โนด | เครื่องจริงหรือ VM ที่รัน container ได้ มี 2 แบบ: Master Node (สมอง) กับ Worker Node (แรงงาน) |
| **Pod** | พ็อด | หน่วยเล็กสุดของ K8s — กล่องที่หุ้ม container ไว้ 1 pod = 1+ container (ปกติ 1 ตัว) |
| **Container** | คอนเทนเนอร์ | กล่องที่มี app + dependencies ทั้งหมด รันได้เหมือนกันทุกเครื่อง (เช่น Docker container) |
| **Namespace** | เนมสเปซ | พื้นที่แยก resources เหมือน "โฟลเดอร์" — ป้องกัน resources ชนกัน |
| **Label** | เลเบล | tag ที่ติดบน resource (key=value) เพื่อจัดกลุ่ม/เลือก เช่น `app=anc-portal-api` |
| **Selector** | ซีเล็กเตอร์ | เงื่อนไขเลือก resource ตาม label เช่น "เลือก Pod ที่มี label `app=api`" |

### 2.2 Workloads (ตัวทำงาน)

| ศัพท์ | ความหมาย | ใช้ในโปรเจคนี้ |
|---|---|---|
| **Deployment** | สั่งให้ K8s สร้างและดูแล Pod — ถ้า Pod ตาย สร้างใหม่ให้ | `anc-portal-api` (API), `anc-portal-worker` (Kafka consumer) |
| **ReplicaSet** | ชุดสำเนาของ Pod — Deployment สร้างให้อัตโนมัติ | (K8s จัดการภายใน) |
| **Job** | ทำงานครั้งเดียวแล้วจบ (one-shot) | `anc-portal-migrate` — รัน database migration |
| **CronJob** | Job ที่รันตามตารางเวลา (เหมือน crontab) | `anc-portal-sync` — sync ข้อมูลทุก 6 ชั่วโมง |

### 2.3 Networking (เครือข่าย)

| ศัพท์ | ความหมาย | ใช้ในโปรเจคนี้ |
|---|---|---|
| **Service** | ชื่อถาวร (DNS) ที่ชี้ไปยังกลุ่ม Pod — ถ้า Pod เปลี่ยน IP ก็ยังเข้าถึงได้ | `anc-portal-api` — ClusterIP port 80 → 20000 |
| **ClusterIP** | Service type ที่ใช้ภายใน cluster เท่านั้น (ข้างนอกเข้าไม่ได้) | Service ของ API |
| **Ingress** | ประตูรับ traffic จากอินเทอร์เน็ตเข้า cluster — เหมือน reverse proxy | `anc-portal-api` — NGINX Ingress Controller |
| **Ingress Controller** | ตัวจัดการ Ingress จริงๆ (ต้องติดตั้งเพิ่ม) | NGINX Ingress Controller |
| **Port-forward** | เปิดท่อจากเครื่อง local เข้าไปยัง Pod/Service ใน cluster | ใช้ debug: `kubectl port-forward svc/api 8080:80` |

### 2.4 Configuration (ตั้งค่า)

| ศัพท์ | ความหมาย | ใช้ในโปรเจคนี้ |
|---|---|---|
| **ConfigMap** | เก็บ config (key-value) ที่ไม่ลับ แยกจาก code | `anc-portal-config` — DB host, Redis, Kafka, OTel, etc. |
| **Secret** | เก็บข้อมูลลับ (password, key) — encode base64 | `anc-portal-secret` — DB password, JWT secret |
| **envFrom** | inject ConfigMap/Secret ทั้งหมดเป็น environment variables เข้า container | ใช้ทั้ง configMapRef + secretRef |

### 2.5 Scaling & Reliability (ขยาย & เสถียร)

| ศัพท์ | ความหมาย | ใช้ในโปรเจคนี้ |
|---|---|---|
| **HPA** | HorizontalPodAutoscaler — เพิ่ม/ลด Pod อัตโนมัติตาม CPU/Memory | API: min 2 → max 6 pods |
| **PDB** | PodDisruptionBudget — กำหนดว่าต้องมี Pod ขั้นต่ำกี่ตัวเสมอ (กัน downtime ตอน maintenance) | API: minAvailable=1 |
| **Replica** | จำนวนสำเนาของ Pod | API: 2 replicas (base), Worker: 1 replica |
| **Rolling Update** | วิธี deploy ที่ค่อยๆ เปลี่ยน Pod ทีละตัว — ไม่มี downtime | maxSurge=1, maxUnavailable=0 |
| **Metrics Server** | component ที่รวบรวม CPU/Memory metric จากทุก Node — จำเป็นสำหรับ HPA | ต้องติดตั้งใน cluster |

### 2.6 Health Checks (ตรวจสุขภาพ)

| ศัพท์ | ความหมาย | ใช้ในโปรเจคนี้ |
|---|---|---|
| **Liveness Probe** | ตรวจว่า process ยัง alive ไหม — ถ้า fail → K8s restart pod | `GET /healthz` ทุก 15 วินาที |
| **Readiness Probe** | ตรวจว่าพร้อมรับ traffic ไหม — ถ้า fail → ถอดออกจาก Service (หยุดส่ง request มา) | `GET /ready` ทุก 10 วินาที |
| **Startup Probe** | ตรวจตอน app เริ่มต้น — ให้เวลา boot (connect DB, warm up) ก่อนเริ่มตรวจ liveness | `GET /healthz` ทุก 5 วินาที |

### 2.7 Tools (เครื่องมือ)

| ศัพท์ | ความหมาย |
|---|---|
| **kubectl** | CLI สำหรับสั่งงาน K8s cluster (อ่านว่า "คิวบ์ คอนโทรล" หรือ "คิวบ์ ซีทีแอล") |
| **Kustomize** | เครื่องมือ template K8s YAML แบบ overlay — ไม่ต้องแก้ไฟล์ base |
| **Helm** | เครื่องมือ template K8s อีกแบบ — ใช้ chart + values (โปรเจคนี้ไม่ได้ใช้) |
| **Docker** | Platform สร้างและรัน container image |
| **Container Registry** | ที่เก็บ Docker image เช่น ghcr.io, Docker Hub, ECR, GCR |

### 2.8 คำย่อที่เจอบ่อย

| คำย่อ | ชื่อเต็ม | ความหมาย |
|---|---|---|
| **K8s** | Kubernetes | K + 8 ตัวอักษร + s |
| **HPA** | HorizontalPodAutoscaler | ระบบ auto-scaling |
| **PDB** | PodDisruptionBudget | กันไม่ให้ Pod ถูกลบพร้อมกันหมด |
| **OTel** | OpenTelemetry | มาตรฐานเก็บ traces, metrics, logs |
| **CORS** | Cross-Origin Resource Sharing | ควบคุมว่า domain ไหนเข้าถึง API ได้ |
| **TLS** | Transport Layer Security | เข้ารหัสการสื่อสาร (HTTPS) |
| **DNS** | Domain Name System | ระบบแปลงชื่อ domain เป็น IP |
| **CI/CD** | Continuous Integration / Continuous Deployment | ระบบ build + deploy อัตโนมัติ |

---

## 3. เครื่องมือที่ต้องติดตั้ง (Prerequisites)

| เครื่องมือ | เวอร์ชันขั้นต่ำ | วิธีติดตั้ง | หน้าที่ |
|---|---|---|---|
| **kubectl** | ≥ 1.28 | [kubernetes.io/docs/tasks/tools](https://kubernetes.io/docs/tasks/tools/) | CLI สั่งงาน K8s |
| **Docker Desktop** | ≥ 24.0 | [docker.com/products/docker-desktop](https://www.docker.com/products/docker-desktop/) | Build image + local K8s |
| **Kustomize** | ≥ 5.0 | มาพร้อม kubectl (built-in) | Template overlays |
| **Kubernetes Cluster** | ≥ 1.28 | GKE / EKS / AKS / Docker Desktop / Minikube | Cluster จริง |
| **Metrics Server** | — | ติดตั้งใน cluster | จำเป็นสำหรับ HPA |

### ตรวจสอบการติดตั้ง

```powershell
# ตรวจ kubectl
kubectl version --client

# ตรวจว่า connect cluster ได้
kubectl cluster-info

# ตรวจ Kustomize version
kubectl kustomize --help

# ตรวจ Docker
docker version

# ตรวจ Metrics Server (ในcluster)
kubectl top nodes
```

### ตั้งค่า kubectl context

```powershell
# ดู context ทั้งหมด (cluster ที่ connect ได้)
kubectl config get-contexts

# เปลี่ยน context (เลือก cluster)
kubectl config use-context <context-name>

# ดู context ปัจจุบัน
kubectl config current-context
```

---

## 4. สถาปัตยกรรม K8s ของโปรเจค

```
                            Internet
                               │
                               ▼
                    ┌─── Ingress (NGINX) ───┐
                    │  api.anc-portal.com    │
                    │  TLS termination       │
                    │  Rate limit: 50 rps    │
                    └──────────┬─────────────┘
                               │
                               ▼
                    ┌─── Service (ClusterIP) ─┐
                    │  anc-portal-api          │
                    │  port 80 → 20000         │
                    └──────────┬───────────────┘
                               │
                    ┌──────────┼──────────┐
                    ▼          ▼          ▼
              ┌─────────┐ ┌─────────┐ ┌─────────┐
              │  Pod 1  │ │  Pod 2  │ │  Pod 3  │    ← API Server (Fiber HTTP)
              │  (API)  │ │  (API)  │ │  (API)  │      HPA: auto 2-6 pods
              └────┬────┘ └────┬────┘ └────┬────┘
                   │           │           │
         ┌─────────────────────────────────────┐
         │              envFrom                 │
         │  ConfigMap ──▶ env vars (non-secret) │
         │  Secret    ──▶ env vars (sensitive)  │
         └─────────────────────────────────────┘
                   │           │
              ┌────┴────┐ ┌───┴───────┐
              │PostgreSQL│ │   Redis   │
              │  (DB)    │ │  (Cache)  │
              └─────────┘ └───────────┘
                               │
                    ┌──────────┴──────────┐
                    ▼                     ▼
              ┌─────────┐          ┌───────────┐
              │  Worker  │◄────────│   Kafka    │
              │  (Pod)   │ consume │  (Queue)   │
              └─────────┘          └───────────┘

    ┌─────────────────┐     ┌──────────────────┐
    │ Job: migrate     │     │ CronJob: sync    │
    │ (one-shot)       │     │ (ทุก 6 ชม.)      │
    │ run DB migration │     │ sync external DB │
    └─────────────────┘     └──────────────────┘
```

### Components สรุป

| Component | K8s Resource | หน้าที่ |
|---|---|---|
| **API Server** | Deployment + Service + Ingress + HPA + PDB | รับ HTTP request จากผู้ใช้ |
| **Worker** | Deployment | Kafka consumer — ประมวลผล background jobs |
| **Migration** | Job | รัน database migration ครั้งเดียว |
| **Sync** | CronJob | Sync ข้อมูลจาก external DB ทุก 6 ชั่วโมง |
| **Config** | ConfigMap | เก็บ env vars ที่ไม่ลับ |
| **Secrets** | Secret | เก็บ password, JWT key |

---

## 5. โครงสร้างไฟล์ K8s ในโปรเจค

```
deployments/k8s/
├── base/                              ← ⭐ ไฟล์ base (ใช้ร่วมทุก environment)
│   ├── kustomization.yaml             ← Entry point — ลิสต์ resources ทั้งหมด
│   ├── namespace.yaml                 ← Namespace: anc-portal
│   ├── configmap.yaml                 ← Environment variables (ไม่ลับ)
│   ├── secret.yaml                    ← Sensitive values (placeholder)
│   ├── api-deployment.yaml            ← API server pods (2 replicas)
│   ├── api-service.yaml               ← ClusterIP Service (80 → 20000)
│   ├── api-hpa.yaml                   ← HorizontalPodAutoscaler (2-6 pods)
│   ├── api-pdb.yaml                   ← PodDisruptionBudget (min 1)
│   ├── api-ingress.yaml               ← Ingress (NGINX, TLS, rate-limit)
│   ├── worker-deployment.yaml         ← Kafka consumer pods (1 replica)
│   ├── migrate-job.yaml               ← Database migration Job
│   └── sync-cronjob.yaml              ← Sync CronJob (ทุก 6 ชม.)
│
└── overlays/                          ← ⭐ การปรับแต่ง per-environment
    ├── staging/
    │   └── kustomization.yaml         ← Staging: 2-4 pods, swagger เปิด
    ├── uat/
    │   └── kustomization.yaml         ← UAT: 2-4 pods, swagger เปิด, Kafka/Redis แยก
    └── production/
        └── kustomization.yaml         ← Production: 3-8 pods, resource สูง, PDB=2
```

### หลักการ: Base + Overlay

```
base/ (ค่าเริ่มต้น) ─────┬──▶ overlays/staging/    (override ค่าสำหรับ staging)
                          ├──▶ overlays/uat/         (override ค่าสำหรับ uat)
                          └──▶ overlays/production/  (override ค่าสำหรับ production)
```

> **เปรียบเทียบง่ายๆ:** ไฟล์ base เหมือน "template" ส่วน overlay เหมือน "แก้ค่าทับ" — ไม่ต้อง copy ไฟล์ซ้ำ

---

## 6. Kustomize คืออะไร และใช้ยังไง

### Kustomize คืออะไร?

**Kustomize** คือเครื่องมือจัดการ K8s YAML แบบ **"patching"** — เขียน base YAML ครั้งเดียว แล้วสร้าง overlay เพื่อ override ค่าตาม environment

### เปรียบเทียบกับวิธีอื่น

| วิธี | ข้อดี | ข้อเสีย |
|---|---|---|
| **Copy YAML แยกแต่ละ env** | ง่ายเข้าใจ | ต้องแก้ทุกไฟล์ ลืมแก้ = bug |
| **Helm (chart + values)** | มี logic (if/else, loop) | ซับซ้อน, ต้องเรียน template syntax |
| **Kustomize (base + overlay)** ✅ | ไม่ต้อง template, แก้เฉพาะที่ต้องการ | ไม่มี logic (if/else) |

### คำสั่ง Kustomize

```powershell
# 1. ดูตัวอย่าง (preview) — ไม่ deploy จริง, แค่แสดง YAML ที่ merge แล้ว
kubectl kustomize deployments/k8s/overlays/staging

# 2. Deploy จริง (apply)
kubectl apply -k deployments/k8s/overlays/staging

# 3. ลบ resources ที่ deploy ไว้
kubectl delete -k deployments/k8s/overlays/staging

# 4. ดูความต่างก่อนและหลัง (diff)
kubectl diff -k deployments/k8s/overlays/staging
```

### kustomization.yaml อธิบาย

```yaml
# deployments/k8s/base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: anc-portal         # ← กำหนด namespace ให้ทุก resource

resources:                     # ← ลิสต์ไฟล์ YAML ที่จะ deploy
  - namespace.yaml
  - configmap.yaml
  - secret.yaml
  - api-deployment.yaml
  - api-service.yaml
  - api-hpa.yaml
  - api-pdb.yaml
  - api-ingress.yaml
  - worker-deployment.yaml
  - migrate-job.yaml
  - sync-cronjob.yaml

commonLabels:                  # ← label ที่จะใส่ให้ทุก resource อัตโนมัติ
  app.kubernetes.io/managed-by: kustomize

images:                        # ← กำหนด image tag กลาง
  - name: ghcr.io/onizukazaza/anc-portal-be-fake
    newTag: latest
```

```yaml
# deployments/k8s/overlays/staging/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base                  # ← ชี้ไปยัง base (inherit ทุกอย่าง)

namePrefix: stg-               # ← เติม prefix ให้ชื่อ resource ทุกตัว
                                #   เช่น anc-portal-api → stg-anc-portal-api

patches:                        # ← แก้ค่าเฉพาะจุดที่ต้องการ
  - target:
      kind: Deployment
      name: anc-portal-api
    patch: |
      - op: replace             # ← operation: replace, add, remove
        path: /spec/replicas    # ← JSON path ของค่าที่จะแก้
        value: 2                # ← ค่าใหม่

images:
  - name: ghcr.io/onizukazaza/anc-portal-be-fake
    newTag: staging             # ← override image tag เป็น "staging"
```

### Patch Operations ที่ใช้

| Operation | ความหมาย | ตัวอย่าง |
|---|---|---|
| `replace` | แก้ค่าเดิม | เปลี่ยน replicas จาก 2 เป็น 3 |
| `add` | เพิ่มค่าใหม่ | เพิ่ม env var ใหม่ใน ConfigMap |
| `remove` | ลบค่า | ลบ annotation ที่ไม่ต้องการ |

---

## 7. อธิบายไฟล์ Base ทีละไฟล์

### 7.1 Namespace

**ไฟล์:** `base/namespace.yaml`

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: anc-portal              # ← ชื่อ namespace
  labels:
    app.kubernetes.io/part-of: anc-portal
```

**อธิบาย:**
- **Namespace** = พื้นที่แยกสำหรับ resources ของเรา
- ทุก resource (Pod, Service, ConfigMap, etc.) จะอยู่ใน namespace `anc-portal`
- ป้องกัน resources ชนกับ namespace อื่น (เช่น `default`, `kube-system`)

**คำสั่ง:**
```powershell
# สร้าง namespace
kubectl apply -f deployments/k8s/base/namespace.yaml

# ดู namespaces ทั้งหมด
kubectl get namespaces

# ตั้ง default namespace (ไม่ต้องพิมพ์ -n ทุกครั้ง)
kubectl config set-context --current --namespace=anc-portal
```

---

### 7.2 ConfigMap

**ไฟล์:** `base/configmap.yaml`

**ConfigMap** เก็บ environment variables ที่ **ไม่เป็นความลับ** — inject เข้า container ผ่าน `envFrom`

| กลุ่ม | ตัวแปร | ค่า | คำอธิบาย |
|---|---|---|---|
| **App** | `STAGE_STATUS` | `staging` | Environment ปัจจุบัน (logging format เปลี่ยนตามค่านี้) |
| | `SERVER_PORT` | `20000` | Port ที่ API server listen |
| | `SERVER_BODY_LIMIT` | `10485760` | ขนาด request body สูงสุด (10 MB) |
| | `SERVER_TIMEOUT` | `30s` | Request timeout |
| | `SERVER_ALLOW_ORIGINS` | `*` | CORS — domain ที่อนุญาตเข้าถึง API |
| **Database** | `DB_HOST` | `postgresql.anc-portal.svc.cluster.local` | ⚠️ DNS ภายใน cluster (ต้องเปลี่ยนตาม DB จริง) |
| | `DB_PORT` | `5432` | PostgreSQL port |
| | `DB_NAME` | `anc_portal` | ชื่อ database |
| | `DB_SSL_MODE` | `require` | เข้ารหัสการเชื่อมต่อ DB |
| | `DB_MAX_CONNS` | `20` | Connection pool สูงสุด |
| | `DB_MIN_CONNS` | `5` | Connection pool ขั้นต่ำ |
| | `DB_MAX_CONN_LIFETIME` | `30m` | อายุ connection สูงสุด |
| | `DB_MAX_CONN_IDLE_TIME` | `5m` | ปล่อย connection ที่ว่างเกิน 5 นาที |
| | `DB_CONNECT_TIMEOUT` | `10s` | timeout เชื่อมต่อ DB |
| | `DB_STATEMENT_TIMEOUT` | `30s` | timeout ต่อ SQL query |
| **Redis** | `REDIS_ENABLED` | `true` | เปิด/ปิด Redis cache |
| | `REDIS_HOST` | `redis.anc-portal.svc.cluster.local` | Redis DNS ภายใน cluster |
| | `REDIS_PORT` | `6379` | Redis port |
| | `REDIS_DB` | `0` | Redis database index |
| | `REDIS_KEY_PREFIX` | `anc:` | prefix ของ key (ป้องกัน key ชน) |
| **Local Cache** | `LOCAL_CACHE_ENABLED` | `true` | เปิด Otter (in-memory L1 cache) |
| | `LOCAL_CACHE_MAX_SIZE` | `10000` | จำนวน item สูงสุดใน cache |
| | `LOCAL_CACHE_TTL` | `5m` | Time-To-Live ของ cache item |
| **Kafka** | `KAFKA_ENABLED` | `true` | เปิด/ปิด Kafka messaging |
| | `KAFKA_BROKERS` | `kafka.anc-portal.svc.cluster.local:9092` | Kafka broker address |
| | `KAFKA_TOPIC` | `anc-portal-events` | ชื่อ topic ส่ง/รับ message |
| | `KAFKA_GROUP_ID` | `anc-portal-worker` | Consumer group ID |
| **OTel** | `OTEL_ENABLED` | `true` | เปิด/ปิด OpenTelemetry tracing |
| | `OTEL_SERVICE_NAME` | `anc-portal-be` | ชื่อ service ใน traces |
| | `OTEL_ENV` | `staging` | environment label |
| | `OTEL_SAMPLE_RATIO` | `0.1` | สัดส่วนการเก็บ trace (10%) |
| | `OTEL_EXPORTER_URL` | `http://otel-collector....:4318` | OTel Collector endpoint |

**คำสั่ง:**
```powershell
# ดู ConfigMap
kubectl get configmap -n anc-portal

# ดูค่า ConfigMap
kubectl describe configmap anc-portal-config -n anc-portal

# ดูค่าในรูปแบบ YAML
kubectl get configmap anc-portal-config -n anc-portal -o yaml
```

---

### 7.3 Secret

**ไฟล์:** `base/secret.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: anc-portal-secret
type: Opaque
stringData:                      # ← ใส่ plain text ได้ (K8s จะ encode base64 ให้)
  DB_USER: "anc_app"
  DB_PASSWORD: "CHANGE_ME"       # ⚠️ ต้องเปลี่ยนเป็นรหัสจริง!
  REDIS_PASSWORD: ""
  JWT_SECRET_KEY: "CHANGE_ME"    # ⚠️ ต้องเปลี่ยนเป็น key จริง!
```

> ⚠️ **ห้าม commit ค่าจริงลง Git!** ไฟล์นี้เป็น placeholder เท่านั้น
> สำหรับ Production ใช้ **External Secrets Operator** หรือ **Sealed Secrets** แทน

**คำสั่ง:**
```powershell
# สร้าง Secret จาก command line (แนะนำ — ไม่ต้องแก้ไฟล์)
kubectl create secret generic anc-portal-secret `
  --namespace=anc-portal `
  --from-literal=DB_USER=anc_app `
  --from-literal=DB_PASSWORD='รหัสจริง' `
  --from-literal=REDIS_PASSWORD='รหัสจริง' `
  --from-literal=JWT_SECRET_KEY='secret-key-จริง'

# ดู Secrets
kubectl get secrets -n anc-portal

# ดูค่า (base64 encoded)
kubectl get secret anc-portal-secret -n anc-portal -o yaml

# decode ค่า
kubectl get secret anc-portal-secret -n anc-portal -o jsonpath='{.data.DB_PASSWORD}' | base64 -d
```

---

### 7.4 Deployment — API Server

**ไฟล์:** `base/api-deployment.yaml`

Deployment คือ resource หลักที่สร้าง **Pod** — ถ้า Pod ตาย K8s จะสร้างใหม่ให้อัตโนมัติ

#### Spec สรุป

| ค่า | ความหมาย | ค่าที่ตั้ง |
|---|---|---|
| `replicas` | จำนวน Pod สำเนา | `2` |
| `revisionHistoryLimit` | เก็บ revision เก่าไว้กี่ชุด (สำหรับ rollback) | `3` |
| `strategy.type` | วิธี deploy | `RollingUpdate` |
| `strategy.rollingUpdate.maxSurge` | จำนวน Pod ใหม่ที่สร้างก่อนลบเก่า | `1` |
| `strategy.rollingUpdate.maxUnavailable` | จำนวน Pod ที่อนุญาตให้ unavailable | `0` (zero-downtime) |
| `image` | Docker image | `ghcr.io/onizukazaza/anc-portal-be-fake:latest` |
| `containerPort` | Port ที่ app listen | `20000` |
| `terminationGracePeriodSeconds` | เวลาให้ pod shutdown gracefully | `15` วินาที |

#### Resources (CPU & Memory)

| Resource | Request (ขั้นต่ำ) | Limit (สูงสุด) |
|---|---|---|
| **CPU** | `100m` (0.1 core) | `500m` (0.5 core) |
| **Memory** | `128Mi` (128 MB) | `256Mi` (256 MB) |

> **Request** = K8s จอง resource ขั้นต่ำให้
> **Limit** = ถ้าใช้เกิน K8s จะ throttle (CPU) หรือ kill (Memory/OOMKilled)

#### Health Probes

| Probe | Endpoint | Initial Delay | Period | Timeout | Fail Threshold |
|---|---|---|---|---|---|
| **startupProbe** | `GET /healthz` | 2s | 5s | — | 10 ครั้ง → restart |
| **livenessProbe** | `GET /healthz` | 5s | 15s | 3s | 3 ครั้ง → restart |
| **readinessProbe** | `GET /ready` | 3s | 10s | 3s | 2 ครั้ง → ถอดจาก Service |

#### Security Context

```yaml
securityContext:
  allowPrivilegeEscalation: false   # ← ห้าม escalation สิทธิ์
  readOnlyRootFilesystem: true      # ← filesystem เป็น read-only
  runAsNonRoot: true                # ← ห้ามรันเป็น root
```

> Security context ป้องกัน container จากการถูกโจมตี (best practice)

---

### 7.5 Service

**ไฟล์:** `base/api-service.yaml`

```yaml
spec:
  type: ClusterIP              # ← เข้าถึงได้ภายใน cluster เท่านั้น
  selector:
    app.kubernetes.io/name: anc-portal-api   # ← เลือก Pod ตาม label
  ports:
    - name: http
      port: 80                 # ← Port ที่ Service เปิด
      targetPort: http         # ← ส่ง traffic ไปยัง Pod port 20000 (ชื่อ "http")
```

**อธิบาย:**
- **Service** เปรียบเหมือน "ชื่อถาวร" ที่ชี้ไปยังกลุ่ม Pod
- Pod มี IP เปลี่ยนตลอด (สร้าง/ลบ) แต่ Service DNS ไม่เปลี่ยน
- DNS ภายใน cluster: `anc-portal-api.anc-portal.svc.cluster.local`
- ส่ง traffic จาก port 80 → Pod port 20000

**คำสั่ง:**
```powershell
# ดู Services
kubectl get svc -n anc-portal

# Port-forward ทดสอบ API ตรง
kubectl port-forward -n anc-portal svc/anc-portal-api 8080:80
# จากนั้น: curl http://localhost:8080/healthz
```

---

### 7.6 HorizontalPodAutoscaler (HPA)

**ไฟล์:** `base/api-hpa.yaml`

HPA **เพิ่ม/ลด Pod อัตโนมัติ**ตาม CPU/Memory usage

#### Spec สรุป

| ค่า | ความหมาย | ค่าที่ตั้ง |
|---|---|---|
| `minReplicas` | Pod ขั้นต่ำ | `2` |
| `maxReplicas` | Pod สูงสุด | `6` |
| CPU target | เพิ่ม Pod ถ้า avg CPU > 70% | `70%` |
| Memory target | เพิ่ม Pod ถ้า avg Memory > 80% | `80%` |

#### Scaling Behavior

| ทิศทาง | Stabilization Window | Policy |
|---|---|---|
| **Scale Up** (เพิ่ม Pod) | 60 วินาที (รอสักครู่ก่อนเพิ่ม) | +2 pods ทุก 60 วินาที |
| **Scale Down** (ลด Pod) | 300 วินาที (5 นาที — รอนานกว่า) | -1 pod ทุก period |

> **Stabilization Window** = ช่วงเวลารอก่อนตัดสินใจ scale — ป้องกัน "กระพริบ" (เพิ่มๆ ลดๆ ซ้ำ)
> Scale down ตั้ง window นานกว่า scale up เพราะเราอยากเพิ่มเร็ว แต่ลดช้า (ป้องกัน traffic พุ่งกลับมา)

**คำสั่ง:**
```powershell
# ดู HPA status
kubectl get hpa -n anc-portal

# ดู metrics จริง (ต้องมี Metrics Server)
kubectl describe hpa anc-portal-api -n anc-portal

# ดู resource usage ของ Pod
kubectl top pods -n anc-portal
```

---

### 7.7 PodDisruptionBudget (PDB)

**ไฟล์:** `base/api-pdb.yaml`

```yaml
spec:
  minAvailable: 1              # ← ต้องมี Pod พร้อมใช้งานอย่างน้อย 1 ตัวเสมอ
  selector:
    matchLabels:
      app.kubernetes.io/name: anc-portal-api
```

**อธิบาย:**
- PDB ป้องกันในกรณี **Voluntary Disruption** เช่น:
  - `kubectl drain node` (ย้าย Pod ออกจาก node)
  - Cluster upgrade (อัพเดต K8s version)
  - Node auto-scaling (ลด node เพราะ load น้อย)
- กำหนดว่า K8s ต้องเหลือ Pod ขั้นต่ำ **1 ตัว** — ไม่ปิดหมดทุกตัวพร้อมกัน
- ไม่ป้องกัน **Involuntary Disruption** (hardware crash, OOMKilled)

> **Production** ตั้ง `minAvailable: 2` เพื่อความปลอดภัยมากขึ้น

---

### 7.8 Ingress

**ไฟล์:** `base/api-ingress.yaml`

**Ingress** = ประตูรับ traffic จากอินเทอร์เน็ตเข้า cluster

#### Spec สรุป

| ค่า | ความหมาย | ค่าที่ตั้ง |
|---|---|---|
| `ingressClassName` | Ingress Controller ที่ใช้ | `nginx` |
| `host` | Domain ที่รับ traffic | `api.anc-portal.example.com` ⚠️ ต้องเปลี่ยน |
| `path` | URL path | `/` (ทั้งหมด) |
| `backend.service` | ส่ง traffic ไปยัง Service | `anc-portal-api` port `http` |
| TLS | ใช้ HTTPS | ✅ ต้องสร้าง secret `anc-portal-tls` |

#### Annotations (ตั้งค่า NGINX)

| Annotation | ค่า | ความหมาย |
|---|---|---|
| `proxy-body-size` | `10m` | ขนาด request body สูงสุด 10 MB |
| `proxy-read-timeout` | `30` | timeout อ่าน response 30 วินาที |
| `proxy-send-timeout` | `30` | timeout ส่ง request 30 วินาที |
| `limit-rps` | `50` | Rate limit: 50 requests/second per IP |
| `ssl-redirect` | `true` | บังคับ redirect HTTP → HTTPS |

> ⚠️ **ก่อนใช้งานต้อง:**
> 1. เปลี่ยน `host` เป็น domain จริง
> 2. สร้าง TLS secret (หรือใช้ cert-manager ออก certificate อัตโนมัติ)
> 3. ติดตั้ง NGINX Ingress Controller บน cluster

---

### 7.9 Deployment — Worker

**ไฟล์:** `base/worker-deployment.yaml`

Worker คือ **Kafka consumer** แยก container — ไม่รับ HTTP request แต่รับ message จาก Kafka

| ค่า | API Deployment | Worker Deployment |
|---|---|---|
| replicas | 2 | 1 |
| image | `anc-portal-be-fake` | `anc-portal-worker` |
| CPU request/limit | 100m / 500m | 50m / 250m |
| Memory request/limit | 128Mi / 256Mi | 64Mi / 128Mi |
| Health port | 20000 | 20001 |
| gracePeriod | 15s | 30s (รอ consume message ให้เสร็จ) |

---

### 7.10 Job — Migration

**ไฟล์:** `base/migrate-job.yaml`

**Job** = ทำงานครั้งเดียวแล้วจบ (ไม่เหมือน Deployment ที่รันตลอด)

```yaml
spec:
  backoffLimit: 3              # ← ล้มเหลวได้สูงสุด 3 ครั้ง (retry)
  ttlSecondsAfterFinished: 300 # ← ลบ Job อัตโนมัติหลังเสร็จ 5 นาที
  template:
    spec:
      restartPolicy: OnFailure # ← ถ้า fail ให้ restart container (ไม่ใช่สร้าง Pod ใหม่)
      containers:
        - name: migrate
          command: ["./migrate"]
          args: ["--env", "staging", "--action", "up"]  # ← migration up
```

**คำสั่ง:**
```powershell
# รัน migration
kubectl apply -f deployments/k8s/base/migrate-job.yaml

# รอให้เสร็จ (timeout 120 วินาที)
kubectl wait --for=condition=complete job/anc-portal-migrate -n anc-portal --timeout=120s

# ดู log
kubectl logs -n anc-portal job/anc-portal-migrate

# ดู status
kubectl get jobs -n anc-portal

# ลบ job เก่า (ก่อนรัน migration ใหม่)
kubectl delete job anc-portal-migrate -n anc-portal
```

> **กฎสำคัญ:** รัน migration **ก่อน** deploy API เสมอ
> Job มี `ttlSecondsAfterFinished: 300` — ลบอัตโนมัติหลังสำเร็จ 5 นาที

---

### 7.11 CronJob — Sync

**ไฟล์:** `base/sync-cronjob.yaml`

**CronJob** = Job ที่รันตามตารางเวลา (เหมือน crontab ใน Linux)

```yaml
spec:
  schedule: "0 */6 * * *"       # ← ทุก 6 ชั่วโมง (00:00, 06:00, 12:00, 18:00)
  concurrencyPolicy: Forbid     # ← ห้ามรันซ้อนกัน (ถ้ายังไม่เสร็จ ข้ามรอบ)
  successfulJobsHistoryLimit: 3  # ← เก็บ job สำเร็จล่าสุด 3 รอบ
  failedJobsHistoryLimit: 3      # ← เก็บ job ล้มเหลวล่าสุด 3 รอบ
```

#### Cron Schedule อธิบาย

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12)
│ │ │ │ ┌───────────── day of week (0-6, 0=Sunday)
│ │ │ │ │
0 */6 * * *    ← นาทีที่ 0 ทุก 6 ชั่วโมง ทุกวัน
```

| Pattern | ความหมาย |
|---|---|
| `0 */6 * * *` | ทุก 6 ชั่วโมง |
| `*/30 * * * *` | ทุก 30 นาที |
| `0 2 * * *` | ทุกวัน ตี 2 |
| `0 0 * * 1` | ทุกวันจันทร์ เที่ยงคืน |

**Sync args:**
```yaml
args:
  - "--env"         # environment
  - "staging"
  - "--table"       # sync ตาราง
  - "all"           # ทุกตาราง
  - "--mode"        # วิธี sync
  - "incremental"   # เฉพาะข้อมูลใหม่ (ไม่ sync ทั้งหมด)
  - "--since"       # ช่วงเวลา
  - "7h"            # 7 ชั่วโมงย้อนหลัง
```

**คำสั่ง:**
```powershell
# ดู CronJobs
kubectl get cronjobs -n anc-portal

# ดู Jobs ที่ CronJob สร้าง
kubectl get jobs -n anc-portal

# สั่งรันทันที (ไม่ต้องรอ schedule)
kubectl create job --from=cronjob/anc-portal-sync anc-portal-sync-manual -n anc-portal

# ดู log ของ sync ล่าสุด
kubectl logs -n anc-portal -l app.kubernetes.io/name=anc-portal-sync --tail=100
```

---

## 8. Overlays — การตั้งค่าตาม Environment

### 8.1 Staging

**ไฟล์:** `overlays/staging/kustomization.yaml`
**namePrefix:** `stg-`

| ค่าที่ override | Base | Staging |
|---|---|---|
| API replicas | 2 | 2 |
| Worker replicas | 1 | 1 |
| HPA min/max | 2/6 | 2/4 |
| STAGE_STATUS | staging | staging |
| OTEL_SAMPLE_RATIO | 0.1 | 0.5 (50%) |
| SWAGGER_ENABLED | — | true |
| Ingress host | example.com | `api-staging.anc-portal.example.com` |
| Image tag | latest | `staging` |

### 8.2 UAT

**ไฟล์:** `overlays/uat/kustomization.yaml`
**namePrefix:** `uat-`

| ค่าที่ override | Base | UAT |
|---|---|---|
| API replicas | 2 | 2 |
| Worker replicas | 1 | 1 |
| HPA min/max | 2/6 | 2/4 |
| OTEL_ENV | staging | uat |
| OTEL_SAMPLE_RATIO | 0.1 | 0.5 (50%) |
| SWAGGER_ENABLED | — | true |
| SWAGGER_HOST | — | `uat-portal-api.anc.co.th` |
| REDIS_KEY_PREFIX | `anc:` | `anc-uat:` (แยก Redis data) |
| KAFKA_TOPIC | `anc-portal-events` | `anc-portal-events-uat` (แยก Kafka topic) |
| SERVER_ALLOW_ORIGINS | `*` | `https://uat-portal.anc.co.th` |

> UAT แยก Redis prefix + Kafka topic เพื่อไม่ให้ data ปนกับ staging

### 8.3 Production

**ไฟล์:** `overlays/production/kustomization.yaml`
**namePrefix:** `prod-`

| ค่าที่ override | Base | Production |
|---|---|---|
| API replicas | 2 | **3** |
| Worker replicas | 1 | **2** |
| HPA min/max | 2/6 | **3/8** |
| CPU request/limit | 100m/500m | **200m/1000m** |
| Memory request/limit | 128Mi/256Mi | **256Mi/512Mi** |
| STAGE_STATUS | staging | **production** |
| SERVER_ALLOW_ORIGINS | `*` | **`https://portal.anc.co.th`** |
| DB_MAX_CONNS | 20 | **30** |
| DB_MIN_CONNS | 5 | **10** |
| OTEL_SAMPLE_RATIO | 0.1 | **0.05** (5%) |
| SWAGGER_ENABLED | — | **false** |
| PDB minAvailable | 1 | **2** |
| Ingress host | example.com | **`api.portal.anc.co.th`** |
| TLS secretName | `anc-portal-tls` | **`anc-portal-tls-prod`** |
| Image tag | latest | **`v1.0.0`** (semantic version) |

---

## 9. Docker Image — การ Build

### Multi-stage Dockerfile

โปรเจคใช้ **multi-stage build** เพื่อให้ image มีขนาดเล็กและปลอดภัย:

```
Stage 1: builder (golang:1.25-alpine)
  └── go build → binary files (api, worker, migrate, seed, sync)

Stage 2: runtime-base (alpine:3.21)
  └── ติดตั้ง ca-certificates, tzdata, curl
  └── สร้าง user: appuser (ไม่ใช่ root)

Stage 3: api (default target)
  └── copy ทุก binary + migrations + config
  └── HEALTHCHECK ตรวจ /healthz
  └── EXPOSE 20000

Stage 4: worker
  └── copy เฉพาะ worker binary + config
```

### Build Commands

```powershell
# Build API image (default)
docker build -f deployments/docker/Dockerfile `
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) `
  --build-arg BUILD_TIME=$(Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ") `
  -t ghcr.io/onizukazaza/anc-portal-be-fake:latest .

# Build Worker image (target เฉพาะ stage "worker")
docker build -f deployments/docker/Dockerfile `
  --target worker `
  -t ghcr.io/anc-portal/anc-portal-worker:latest .

# Push ขึ้น registry
docker push ghcr.io/onizukazaza/anc-portal-be-fake:latest
docker push ghcr.io/anc-portal/anc-portal-worker:latest
```

### Build Arguments

| Arg | ความหมาย | ใช้ใน |
|---|---|---|
| `GIT_COMMIT` | Git commit hash (short) | แสดงใน startup banner |
| `BUILD_TIME` | เวลา build | แสดงใน startup banner |

> `-ldflags` inject ค่าเข้า binary ผ่าน `pkg/buildinfo` — ทำให้ banner แสดงเวอร์ชัน:
> `Build ··· a1b2c3d (2026-03-28T10:30:00Z)`

---

## 10. คำสั่ง kubectl ที่ใช้บ่อย

### 10.1 ดูข้อมูล (Get / Describe)

```powershell
# ดู resources ทั้งหมดใน namespace
kubectl get all -n anc-portal

# ดู Pods
kubectl get pods -n anc-portal

# ดู Pods แบบ watch (real-time)
kubectl get pods -n anc-portal -w

# ดู Pods พร้อม node ที่รัน
kubectl get pods -n anc-portal -o wide

# ดูรายละเอียด Pod (events, conditions, etc.)
kubectl describe pod <pod-name> -n anc-portal

# ดู Deployments
kubectl get deployments -n anc-portal

# ดู Services
kubectl get svc -n anc-portal

# ดู ConfigMaps
kubectl get configmap -n anc-portal

# ดู Secrets
kubectl get secrets -n anc-portal

# ดู Jobs
kubectl get jobs -n anc-portal

# ดู CronJobs
kubectl get cronjobs -n anc-portal

# ดู Ingress
kubectl get ingress -n anc-portal

# ดู HPA
kubectl get hpa -n anc-portal

# ดู Events (เรียงตามเวลา)
kubectl get events -n anc-portal --sort-by='.lastTimestamp'
```

### 10.2 ดู Logs

```powershell
# ดู logs ของ Pod เฉพาะตัว
kubectl logs <pod-name> -n anc-portal

# ดู logs แบบ follow (real-time)
kubectl logs <pod-name> -n anc-portal -f

# ดู logs ของ container ที่ crash ก่อนหน้า
kubectl logs <pod-name> -n anc-portal --previous

# ดู logs ตาม label (ทุก API pods)
kubectl logs -n anc-portal -l app.kubernetes.io/name=anc-portal-api -f

# ดู logs ล่าสุด 100 บรรทัด
kubectl logs <pod-name> -n anc-portal --tail=100

# ดู logs ของ Job
kubectl logs -n anc-portal job/anc-portal-migrate
```

### 10.3 Deploy & Apply

```powershell
# Apply ด้วย Kustomize
kubectl apply -k deployments/k8s/overlays/staging

# Apply ไฟล์เดียว
kubectl apply -f deployments/k8s/base/namespace.yaml

# Preview ก่อน deploy (dry-run)
kubectl kustomize deployments/k8s/overlays/staging

# ดูความต่าง (diff)
kubectl diff -k deployments/k8s/overlays/staging

# ลบ resources ที่ deploy ไว้
kubectl delete -k deployments/k8s/overlays/staging
```

### 10.4 Scaling

```powershell
# Scale Deployment (manual)
kubectl scale deployment anc-portal-api -n anc-portal --replicas=4

# Scale Worker
kubectl scale deployment anc-portal-worker -n anc-portal --replicas=3

# ดู HPA metrics
kubectl describe hpa anc-portal-api -n anc-portal

# ดู resource usage
kubectl top pods -n anc-portal
kubectl top nodes
```

### 10.5 Rollback

```powershell
# ดู revision history
kubectl rollout history deployment/anc-portal-api -n anc-portal

# ดู rollout status (หลัง deploy ใหม่)
kubectl rollout status deployment/anc-portal-api -n anc-portal

# Rollback ไป revision ก่อนหน้า
kubectl rollout undo deployment/anc-portal-api -n anc-portal

# Rollback ไป revision เฉพาะ
kubectl rollout undo deployment/anc-portal-api -n anc-portal --to-revision=3

# Restart pods ทั้งหมด (rolling restart)
kubectl rollout restart deployment/anc-portal-api -n anc-portal
```

### 10.6 Debug

```powershell
# เข้า shell ใน Pod
kubectl exec -it <pod-name> -n anc-portal -- /bin/sh

# Port-forward ทดสอบ API
kubectl port-forward -n anc-portal svc/anc-portal-api 8080:80
# จากนั้น: curl http://localhost:8080/healthz

# ดู environment variables ใน Pod
kubectl exec <pod-name> -n anc-portal -- env | Sort-Object

# ดู ConfigMap ที่ inject เข้า Pod
kubectl describe configmap anc-portal-config -n anc-portal

# ดู events ของ Pod
kubectl describe pod <pod-name> -n anc-portal

# คัดลอกไฟล์จาก Pod
kubectl cp anc-portal/<pod-name>:/app/logs/error.log ./error.log
```

### 10.7 Namespace & Context

```powershell
# ดู namespaces
kubectl get namespaces

# ตั้ง default namespace (ไม่ต้องพิมพ์ -n ทุกครั้ง)
kubectl config set-context --current --namespace=anc-portal

# ดู context ปัจจุบัน
kubectl config current-context

# เปลี่ยน cluster
kubectl config use-context <context-name>

# ดูทุก contexts
kubectl config get-contexts
```

---

## 11. ขั้นตอนการ Deploy ตั้งแต่เริ่มต้น (Step-by-Step)

### Step 1: เตรียม Image

```powershell
# Build API image
docker build -f deployments/docker/Dockerfile `
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) `
  --build-arg BUILD_TIME=$(Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ") `
  -t ghcr.io/onizukazaza/anc-portal-be-fake:staging .

# Build Worker image (target stage "worker")
docker build -f deployments/docker/Dockerfile `
  --target worker `
  -t ghcr.io/anc-portal/anc-portal-worker:staging .

# Push ขึ้น registry
docker push ghcr.io/onizukazaza/anc-portal-be-fake:staging
docker push ghcr.io/anc-portal/anc-portal-worker:staging
```

### Step 2: สร้าง Namespace

```powershell
kubectl apply -f deployments/k8s/base/namespace.yaml
kubectl get namespaces | Select-String "anc-portal"
```

### Step 3: สร้าง Secret (ใส่ค่าจริง)

```powershell
# สร้าง secret จาก command line (แนะนำ — ไม่ commit ค่าจริง)
kubectl create secret generic anc-portal-secret `
  --namespace=anc-portal `
  --from-literal=DB_USER=anc_app `
  --from-literal=DB_PASSWORD='รหัสจริง' `
  --from-literal=REDIS_PASSWORD='รหัสจริง' `
  --from-literal=JWT_SECRET_KEY='secret-key-จริง'

# ตรวจสอบ
kubectl get secrets -n anc-portal
```

### Step 4: รัน Migration

```powershell
# Run migration job
kubectl apply -f deployments/k8s/base/migrate-job.yaml

# รอให้เสร็จ
kubectl wait --for=condition=complete job/anc-portal-migrate -n anc-portal --timeout=120s

# ดู log
kubectl logs -n anc-portal job/anc-portal-migrate

# ตรวจ status
kubectl get jobs -n anc-portal
```

### Step 5: Deploy ทั้งหมดด้วย Kustomize

```powershell
# Preview (dry-run) — ดูก่อนว่า YAML จะเป็นอย่างไร
kubectl kustomize deployments/k8s/overlays/staging

# Deploy จริง
kubectl apply -k deployments/k8s/overlays/staging
```

### Step 6: ตรวจสอบ

```powershell
# ดู Pods (watch mode)
kubectl get pods -n anc-portal -w

# รอ rollout สำเร็จ
kubectl rollout status deployment/anc-portal-api -n anc-portal

# ดู logs
kubectl logs -n anc-portal -l app.kubernetes.io/name=anc-portal-api -f

# ทดสอบ health
kubectl port-forward -n anc-portal svc/anc-portal-api 8080:80
# Terminal ใหม่:
curl http://localhost:8080/healthz
curl http://localhost:8080/ready
```

### Step 7: ตรวจสอบ HPA

```powershell
kubectl get hpa -n anc-portal
# ควรเห็น:
# NAME             REFERENCE                   TARGETS        MINPODS   MAXPODS
# anc-portal-api   Deployment/anc-portal-api   <cpu>/<mem>    2         6
```

---

## 12. Health Checks & Probes อธิบาย

### แนวคิด: ทำไมต้องมี Probe?

K8s ไม่รู้ว่า app ข้างใน container "พร้อม" หรือ "พัง" — probe เป็นวิธีให้ K8s ตรวจสอบ

### ลำดับการทำงาน

```
Pod สร้าง → startupProbe ตรวจ → (ผ่าน) → livenessProbe + readinessProbe เริ่มทำงาน
                                          │                    │
                                          │                    └▶ ผ่าน → รับ traffic
                                          │                       ไม่ผ่าน → ถอดออกจาก Service
                                          │
                                          └▶ ผ่าน → ปกติ
                                             ไม่ผ่าน → restart Pod
```

### API Server Probes

| Probe | Endpoint | ทำอะไร | ทุกกี่วินาที | ล้มเหลวกี่ครั้ง = action |
|---|---|---|---|---|
| **startupProbe** | `GET /healthz` | ตรวจว่า boot เสร็จ (DB connected) | 5s | 10 ครั้ง → restart |
| **livenessProbe** | `GET /healthz` | ตรวจว่า process alive | 15s | 3 ครั้ง → restart |
| **readinessProbe** | `GET /ready` | ตรวจว่าพร้อมรับ request | 10s | 2 ครั้ง → ถอดจาก Service |

> `/healthz` ตรวจ: DB + Redis connectivity
> `/ready` ตรวจ: DB + Redis + return timestamp

### Worker Probes

| Probe | Endpoint | Port | ทำอะไร |
|---|---|---|---|
| **livenessProbe** | `GET /healthz` | 20001 | ตรวจ Kafka consumer alive |
| **readinessProbe** | `GET /healthz` | 20001 | ตรวจ consumer พร้อมรับ messages |

> Worker health ตรวจจาก `consumer.IsHealthy()` (atomic.Bool — true หลัง first successful fetch)

---

## 13. Resource Management — CPU & Memory

### หน่วยวัด

| หน่วย | ความหมาย | ตัวอย่าง |
|---|---|---|
| `m` (millicore) | 1/1000 ของ CPU core | `100m` = 0.1 core, `1000m` = 1 core |
| `Mi` (Mebibyte) | 1,048,576 bytes (~1 MB) | `128Mi` ≈ 128 MB |
| `Gi` (Gibibyte) | 1,073,741,824 bytes (~1 GB) | `1Gi` ≈ 1 GB |

### Request vs Limit

| | **Request** | **Limit** |
|---|---|---|
| ความหมาย | ขั้นต่ำที่ K8s จองให้ | สูงสุดที่ใช้ได้ |
| K8s ใช้ทำอะไร | เลือก node ที่มี resource ว่างพอ | ป้องกัน container กิน resource เกิน |
| CPU เกิน | — | **Throttle** (ช้าลง แต่ไม่ kill) |
| Memory เกิน | — | **OOMKilled** (kill ทันที) |

### Spec ของโปรเจค

| Component | CPU Request | CPU Limit | Mem Request | Mem Limit |
|---|---|---|---|---|
| **API (base)** | 100m | 500m | 128Mi | 256Mi |
| **API (production)** | 200m | 1000m | 256Mi | 512Mi |
| **Worker** | 50m | 250m | 64Mi | 128Mi |
| **Migration Job** | 50m | 200m | 64Mi | 128Mi |
| **Sync Job** | 100m | 500m | 128Mi | 256Mi |

---

## 14. Scaling — การขยายระบบ

### Auto-scaling (HPA)

HPA ตั้งค่าใน `api-hpa.yaml` — เพิ่ม/ลด Pod ตาม metric

```
Traffic 📈 → CPU usage สูง → HPA เพิ่ม Pod → traffic กระจาย → CPU ลดลง
Traffic 📉 → CPU usage ต่ำ → (รอ 5 นาที) → HPA ลด Pod → ประหยัด resource
```

| Environment | Min Pods | Max Pods | CPU Target | Memory Target |
|---|---|---|---|---|
| Staging | 2 | 4 | 70% | 80% |
| UAT | 2 | 4 | 70% | 80% |
| Production | 3 | 8 | 70% | 80% |

### Manual Scaling

```powershell
# เพิ่ม API replicas เป็น 5
kubectl scale deployment anc-portal-api -n anc-portal --replicas=5

# เพิ่ม Worker replicas
kubectl scale deployment anc-portal-worker -n anc-portal --replicas=3

# ⚠️ ถ้ามี HPA ค่า replicas อาจถูก override กลับ
# ให้แก้ HPA แทน:
kubectl patch hpa anc-portal-api -n anc-portal -p '{"spec":{"minReplicas":5}}'
```

---

## 15. Secrets Management — จัดการข้อมูลลับ

### วิธีที่ 1: kubectl (Dev/Staging)

```powershell
kubectl create secret generic anc-portal-secret `
  --namespace=anc-portal `
  --from-literal=DB_USER=anc_app `
  --from-literal=DB_PASSWORD='my-password' `
  --from-literal=JWT_SECRET_KEY='my-jwt-secret'
```

### วิธีที่ 2: External Secrets Operator (Production — แนะนำ)

ดึง secret จาก cloud provider (AWS Secrets Manager, GCP Secret Manager, Azure Key Vault):

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: anc-portal-secret
  namespace: anc-portal
spec:
  refreshInterval: 1h                    # ← ดึงค่าใหม่ทุกชั่วโมง
  secretStoreRef:
    name: aws-secrets-manager            # ← หรือ gcp-secret-manager
    kind: ClusterSecretStore
  target:
    name: anc-portal-secret              # ← ชื่อ K8s Secret ที่จะสร้าง
  data:
    - secretKey: DB_PASSWORD
      remoteRef:
        key: anc-portal/db-password      # ← key ใน cloud secret store
    - secretKey: JWT_SECRET_KEY
      remoteRef:
        key: anc-portal/jwt-secret
```

### วิธีที่ 3: Sealed Secrets (GitOps)

encrypt secret แล้ว commit ลง Git ได้ (decrypt ได้เฉพาะ cluster):

```powershell
kubeseal --format=yaml < deployments/k8s/base/secret.yaml > sealed-secret.yaml
kubectl apply -f sealed-secret.yaml
```

---

## 16. Monitoring & Observability

### Built-in Endpoints

| Endpoint | หน้าที่ |
|---|---|
| `GET /healthz` | Health check — DB + Redis |
| `GET /ready` | Readiness check + timestamp |
| `GET /metrics` | Prometheus metrics |

### แนะนำ: Prometheus + Grafana

```yaml
# ServiceMonitor (ถ้าใช้ Prometheus Operator)
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

### เครื่องมือ Observability

| เครื่องมือ | หน้าที่ |
|---|---|
| **Prometheus** | เก็บ metrics (CPU, memory, request count, latency) |
| **Grafana** | Dashboard visualize metrics |
| **Tempo** | Distributed tracing (OTel → Tempo) |
| **Loki** | Log aggregation (stdout → Loki) |

---

## 17. Rollback — ย้อนกลับเวอร์ชัน

### วิธี Rolling Update ทำงาน

```
Deployment v1 (2 pods)
  │
  ├── สร้าง Pod v2 ตัวที่ 1 (maxSurge=1)
  ├── รอ Readiness Probe ผ่าน
  ├── ลบ Pod v1 ตัวที่ 1 (maxUnavailable=0 = zero-downtime)
  ├── สร้าง Pod v2 ตัวที่ 2
  ├── รอ Readiness Probe ผ่าน
  └── ลบ Pod v1 ตัวที่ 2
  │
  ✅ Deployment v2 (2 pods) สำเร็จ
```

### Rollback Commands

```powershell
# ดูประวัติ revision
kubectl rollout history deployment/anc-portal-api -n anc-portal

# Rollback ไปเวอร์ชันก่อนหน้า
kubectl rollout undo deployment/anc-portal-api -n anc-portal

# Rollback ไป revision เฉพาะ (เช่น revision 3)
kubectl rollout undo deployment/anc-portal-api -n anc-portal --to-revision=3

# ดู status ระหว่าง rollout
kubectl rollout status deployment/anc-portal-api -n anc-portal
```

### Database Migration Rollback

```powershell
# ⚠️ ระวัง — migration down อาจลบ data!
# ต้องแก้ migrate-job.yaml เปลี่ยน args จาก "up" เป็น "down"
# แล้ว apply ใหม่
```

> **Best Practice:** เขียน migration ให้ backward-compatible เสมอ — code เก่ายังทำงานได้กับ schema ใหม่

---

## 18. Troubleshooting — แก้ปัญหา

### 18.1 Pod ไม่ขึ้น

```powershell
# 1. ดู Pod status
kubectl get pods -n anc-portal

# 2. ดู events ของ Pod
kubectl describe pod <pod-name> -n anc-portal

# 3. ดู logs
kubectl logs <pod-name> -n anc-portal

# 4. ดู logs ของ container ที่ crash ก่อนหน้า
kubectl logs <pod-name> -n anc-portal --previous

# 5. ดู events ทั้ง namespace
kubectl get events -n anc-portal --sort-by='.lastTimestamp'
```

### 18.2 ปัญหาที่พบบ่อย

| อาการ | สาเหตุ | วิธีแก้ |
|---|---|---|
| **CrashLoopBackOff** | App crash ซ้ำ (DB connect ไม่ได้, config ผิด) | ตรวจ logs + ConfigMap + Secret |
| **ImagePullBackOff** | ดึง Docker image ไม่ได้ | ตรวจ image name + registry access + imagePullSecrets |
| **Pending** | ไม่มี node ว่างพอ (resource ไม่พอ) | ตรวจ `kubectl describe pod` ดู Events |
| **Readiness fail** | App boot ไม่เสร็จ / dependency ไม่พร้อม | ตรวจ Redis, DB connectivity |
| **OOMKilled** | Memory ไม่พอ | เพิ่ม `resources.limits.memory` |
| **HPA ไม่ scale** | ไม่มี Metrics Server | รัน `kubectl top pods` — ถ้า error = ติดตั้ง metrics-server |
| **Ingress ไม่ทำงาน** | ไม่มี Ingress Controller | ตรวจว่าติดตั้ง NGINX Ingress Controller |

### 18.3 Debug Commands

```powershell
# เข้า shell ใน Pod
kubectl exec -it <pod-name> -n anc-portal -- /bin/sh

# Port-forward ทดสอบ API ตรง
kubectl port-forward -n anc-portal svc/anc-portal-api 8080:80

# ดู config ที่ inject เข้า Pod
kubectl exec <pod-name> -n anc-portal -- env | Sort-Object

# ดู resource usage
kubectl top pods -n anc-portal
kubectl top nodes

# ดู DNS resolution ใน cluster
kubectl exec <pod-name> -n anc-portal -- nslookup postgresql.anc-portal.svc.cluster.local

# ดู endpoint ของ Service (IP ของ Pod ที่อยู่หลัง Service)
kubectl get endpoints -n anc-portal
```

---

## 19. ตารางเปรียบเทียบ Spec ทุก Environment

| ค่า | Base (default) | Staging | UAT | Production |
|---|---|---|---|---|
| **namePrefix** | — | `stg-` | `uat-` | `prod-` |
| **API replicas** | 2 | 2 | 2 | 3 |
| **Worker replicas** | 1 | 1 | 1 | 2 |
| **HPA min/max** | 2/6 | 2/4 | 2/4 | 3/8 |
| **CPU req/limit (API)** | 100m/500m | 100m/500m | 100m/500m | 200m/1000m |
| **Mem req/limit (API)** | 128Mi/256Mi | 128Mi/256Mi | 128Mi/256Mi | 256Mi/512Mi |
| **PDB minAvailable** | 1 | 1 | 1 | 2 |
| **DB_MAX_CONNS** | 20 | 20 | 20 | 30 |
| **DB_MIN_CONNS** | 5 | 5 | 5 | 10 |
| **OTEL_SAMPLE_RATIO** | 0.1 | 0.5 | 0.5 | 0.05 |
| **SWAGGER_ENABLED** | — | true | true | false |
| **CORS origins** | `*` | `*` | `https://uat-portal.anc.co.th` | `https://portal.anc.co.th` |
| **REDIS_KEY_PREFIX** | `anc:` | `anc:` | `anc-uat:` | `anc:` |
| **KAFKA_TOPIC** | `anc-portal-events` | `anc-portal-events` | `anc-portal-events-uat` | `anc-portal-events` |
| **Image tag** | `latest` | `staging` | `staging` | `v1.0.0` |
| **Ingress host** | `api.anc-portal.example.com` | `api-staging.anc-portal.example.com` | — | `api.portal.anc.co.th` |

---

## 20. Checklist ก่อน Deploy

### Staging Checklist

- [ ] Unit tests ผ่านทั้งหมด (`go test ./...`)
- [ ] Docker build สำเร็จ
- [ ] Docker image push สำเร็จ
- [ ] Migration tested locally
- [ ] Secret สร้างแล้ว (DB_PASSWORD, JWT_SECRET_KEY)
- [ ] ConfigMap ค่าถูกต้อง (DB_HOST, REDIS_HOST)
- [ ] `kubectl kustomize` preview ไม่มี error
- [ ] Deploy สำเร็จ: `kubectl get pods -n anc-portal`
- [ ] Health check ผ่าน: `/healthz` return 200
- [ ] Readiness ผ่าน: `/ready` return 200

### Production Checklist

- [ ] ✅ ผ่าน staging testing แล้ว
- [ ] Image tag เป็น **semantic version** (`v1.0.0`) — ไม่ใช่ `latest`
- [ ] Migration **backward-compatible** (code เก่ายังทำงานได้)
- [ ] Secrets ใช้ **External Secrets Operator** (ไม่ใช่ plain kubectl)
- [ ] `OTEL_SAMPLE_RATIO` ลดเหลือ 5% (`0.05`)
- [ ] `SWAGGER_ENABLED=false` — ปิด Swagger UI
- [ ] `SERVER_ALLOW_ORIGINS` เฉพาะ domain จริง (ไม่ใช่ `*`)
- [ ] `PDB minAvailable` ≥ 2
- [ ] HPA configured + **Metrics Server ready** (`kubectl top pods` ไม่ error)
- [ ] Ingress host + TLS ตั้งค่าถูกต้อง
- [ ] Rollback plan พร้อม (เวอร์ชันก่อนหน้ายังอยู่)
- [ ] ทดสอบ rollback flow: `kubectl rollout undo`

---

## สิ่งที่ต้องเปลี่ยนก่อน Deploy จริง

| # | ไฟล์ | สิ่งที่ต้องเปลี่ยน |
|---|---|---|
| 1 | `base/secret.yaml` | ใส่ DB_PASSWORD, JWT_SECRET_KEY จริง หรือใช้ External Secrets |
| 2 | `base/api-deployment.yaml` | เปลี่ยน `image:` เป็น registry จริง |
| 3 | `base/worker-deployment.yaml` | เปลี่ยน `image:` เป็น registry จริง |
| 4 | `base/configmap.yaml` | ใส่ DB_HOST, REDIS_HOST, KAFKA_BROKERS ที่ถูกต้องตาม cluster |
| 5 | `base/api-ingress.yaml` | เปลี่ยน host เป็น domain จริง + สร้าง TLS secret / cert-manager |
| 6 | `overlays/production/` | ใส่ CORS domain จริง, Ingress host จริง, image tag จริง |
| 7 | (เพิ่มเอง) | `imagePullSecret` ถ้าใช้ private registry |

---

> **v1.0** — March 2026 | ANC Portal Backend Team
