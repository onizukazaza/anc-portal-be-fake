# Kubernetes Resource & Spec Guide — ANC Portal Backend

> **v1.0** — Last updated: March 2026
>
> คู่มือการตั้งค่า CPU, Memory, Replicas, Probe
> พร้อมแนวทาง capacity planning สำหรับ Go application

---

## สารบัญ

1. [ทำไมต้องตั้ง Resource?](#1-ทำไมต้องตั้ง-resource)
2. [หน่วยวัด CPU & Memory](#2-หน่วยวัด-cpu--memory)
3. [Request vs Limit — คำอธิบายเชิงลึก](#3-request-vs-limit--คำอธิบายเชิงลึก)
4. [Spec ปัจจุบันของโปรเจค](#4-spec-ปัจจุบันของโปรเจค)
5. [วิธีคำนวณ Resource สำหรับ Go App](#5-วิธีคำนวณ-resource-สำหรับ-go-app)
6. [ResourceQuota — จำกัด Resource ระดับ Namespace](#6-resourcequota--จำกัด-resource-ระดับ-namespace)
7. [LimitRange — Default Resource ให้ Pod](#7-limitrange--default-resource-ให้-pod)
8. [Quality of Service (QoS) Class](#8-quality-of-service-qos-class)
9. [Probe Tuning — ตั้งค่า Health Check](#9-probe-tuning--ตั้งค่า-health-check)
10. [HPA Tuning — ตั้งค่า Auto-scaling](#10-hpa-tuning--ตั้งค่า-auto-scaling)
11. [ตาราง Spec แนะนำตาม Workload](#11-ตาราง-spec-แนะนำตาม-workload)
12. [Capacity Planning — วางแผนทรัพยากร](#12-capacity-planning--วางแผนทรัพยากร)
13. [Troubleshooting — ปัญหา Resource](#13-troubleshooting--ปัญหา-resource)
14. [Checklist ตั้งค่า Resource](#14-checklist-ตั้งค่า-resource)

---

## 1. ทำไมต้องตั้ง Resource?

| ไม่ตั้ง Resource | ผลลัพธ์ |
|---|---|
| ไม่มี `requests` | K8s ไม่รู้ว่า Pod ต้องการ resource เท่าไหร่ → อาจ schedule ไปที่ node ที่เต็มแล้ว |
| ไม่มี `limits` | Container กิน CPU/Memory ไม่จำกัด → กระทบ Pod อื่นบน node เดียวกัน |
| Request สูงเกินไป | Pod หา node ว่างไม่เจอ → **Pending** ตลอด |
| Limit ต่ำเกินไป | CPU ถูก **throttle** (ช้า) หรือ Memory **OOMKilled** (ตาย) |

> **กฎทอง:** ทุก container ต้องมีทั้ง `requests` และ `limits`

---

## 2. หน่วยวัด CPU & Memory

### CPU

| หน่วย | ความหมาย | ตัวอย่าง |
|---|---|---|
| `1` | 1 CPU core เต็ม | container ใช้ได้ 1 core |
| `500m` | 0.5 core (500 millicore) | ครึ่ง core |
| `100m` | 0.1 core | 1/10 ของ core |
| `250m` | 0.25 core | 1/4 ของ core |

> **เทียบง่ายๆ:** `1000m = 1 core = 1 vCPU`

### Memory

| หน่วย | ขนาดจริง | หมายเหตุ |
|---|---|---|
| `128Mi` | 134,217,728 bytes (~128 MB) | Mebibyte (base 2) |
| `256Mi` | ~256 MB | |
| `512Mi` | ~512 MB | |
| `1Gi` | ~1 GB | Gibibyte (base 2) |

> **ข้อแตกต่าง:** `Mi` (Mebibyte = 2²⁰) vs `M` (Megabyte = 10⁶) — K8s แนะนำใช้ `Mi`

### ตัวอย่าง YAML

```yaml
resources:
  requests:
    cpu: "100m"        # ← จอง 0.1 core
    memory: "128Mi"    # ← จอง 128 MB
  limits:
    cpu: "500m"        # ← ใช้ได้สูงสุด 0.5 core
    memory: "256Mi"    # ← ใช้ได้สูงสุด 256 MB (เกิน = OOMKilled)
```

---

## 3. Request vs Limit — คำอธิบายเชิงลึก

### Request — "ขั้นต่ำที่ขอจอง"

```
Pod ต้องการ CPU request = 100m

Node A: ว่าง 200m → ✅ schedule ได้
Node B: ว่าง  50m → ❌ ไม่พอ ข้ามไป
```

- K8s ใช้ **request** เลือก node (scheduling)
- Resource ถูก "จอง" แม้ container ใช้จริงน้อยกว่า
- **ตั้งตามที่ app ใช้ปกติ** (average usage)

### Limit — "เพดานสูงสุด"

| Resource | เกิน Limit | ผลลัพธ์ |
|---|---|---|
| **CPU** | ใช้เกิน limit | **Throttle** — ช้าลง แต่ไม่ตาย |
| **Memory** | ใช้เกิน limit | **OOMKilled** — container ถูก kill ทันที |

> **CPU Throttle** ไม่อันตราย แต่ทำให้ latency สูงขึ้น
> **Memory OOM** อันตราย — Pod restart loop ถ้าแก้ไม่ได้

### อัตราส่วนแนะนำ

| สถานการณ์ | Request : Limit | เหตุผล |
|---|---|---|
| **API (latency-sensitive)** | 1:5 (CPU), 1:2 (Mem) | burst ได้เวลา traffic พุ่ง |
| **Worker (throughput-oriented)** | 1:5 (CPU), 1:2 (Mem) | ใช้ CPU เท่ากัน per message |
| **Job/CronJob (batch)** | 1:4 (CPU), 1:2 (Mem) | ทำเสร็จแล้วปิด ใช้เต็มที่ได้ |
| **ถ้าไม่แน่ใจ** | 1:3 (ทั้งคู่) | ปลอดภัย ไม่เปลืองมาก |

---

## 4. Spec ปัจจุบันของโปรเจค

### ตาราง Resource ทุก Component

| Component | CPU Request | CPU Limit | Mem Request | Mem Limit | อัตราส่วน CPU |
|---|---|---|---|---|---|
| **API (base)** | 100m | 500m | 128Mi | 256Mi | 1:5 |
| **API (production)** | 200m | 1000m | 256Mi | 512Mi | 1:5 |
| **Worker** | 50m | 250m | 64Mi | 128Mi | 1:5 |
| **Migration Job** | 50m | 200m | 64Mi | 128Mi | 1:4 |
| **Sync CronJob** | 100m | 500m | 128Mi | 256Mi | 1:5 |

### ตาราง Replica & Scaling

| Component | Base | Staging | UAT | Production |
|---|---|---|---|---|
| **API replicas** | 2 | 2 | 2 | 3 |
| **Worker replicas** | 1 | 1 | 1 | 2 |
| **HPA min/max** | 2/6 | 2/4 | 2/4 | 3/8 |
| **PDB minAvailable** | 1 | 1 | 1 | 2 |

### Resource รวม (ทั้ง namespace)

| Environment | Total CPU Request | Total Mem Request | หมายเหตุ |
|---|---|---|---|
| **Staging** | ~350m | ~384Mi | 2 API + 1 Worker + CronJob เป็นระยะ |
| **Production (min)** | ~900m | ~896Mi | 3 API + 2 Worker |
| **Production (max HPA)** | ~2200m | ~2.3Gi | 8 API + 2 Worker |

---

## 5. วิธีคำนวณ Resource สำหรับ Go App

### ขั้นตอนที่ 1: วัด Baseline ใน Local

```powershell
# รัน app แล้ววัด memory baseline
# Go app ปกติใช้ ~20-50 MB ตอนเริ่มต้น

# Watch memory ด้วย Go pprof
curl http://localhost:20000/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

### ขั้นตอนที่ 2: Load Test

```powershell
# ใช้ hey หรือ k6 ยิง load test
hey -n 1000 -c 50 http://localhost:20000/v1/quotations

# ดู peak CPU/Memory ระหว่าง test
# → ใช้เป็น basis สำหรับ limit
```

### ขั้นตอนที่ 3: คำนวณ

```
Request = average usage ระหว่าง normal traffic
Limit  = peak usage × 1.5 (safety margin)
```

### สูตรง่าย สำหรับ Go HTTP API

| Metric | สูตรประมาณ | ตัวอย่าง |
|---|---|---|
| **Memory baseline** | ~30-50 MB (Go runtime + pool) | 40 MB |
| **Memory per connection** | ~2-5 KB × max_conns | 5KB × 100 = 500 KB |
| **Memory peak** | baseline × 2-3 (under load) | 40MB × 2.5 = 100 MB |
| **CPU idle** | ~5-20m | 10m |
| **CPU per request** | ~0.5-2m (simple CRUD) | 1m × 100 concurrent = 100m |
| **CPU peak** | per_request × max_concurrent | 2m × 200 = 400m |

### ตัวอย่างคำนวณ API Server

```
สมมติ: app ใช้ memory 40 MB idle, peak 100 MB ที่ 200 req/s

Memory Request = 100 MB → ปัดขึ้น → 128Mi
Memory Limit   = 100 MB × 2 = 200 MB → ปัดขึ้น → 256Mi

CPU idle = 15m, CPU peak = 150m ที่ 200 req/s

CPU Request = 100m (ค่ากลาง ให้ schedule ได้ง่าย)
CPU Limit   = 500m (เผื่อ burst)
```

### Go-specific Tips

| เรื่อง | คำแนะนำ |
|---|---|
| **GOMAXPROCS** | ตั้ง = CPU limit ÷ 1000 (เช่น 500m = 0.5 → GOMAXPROCS=1) หรือใช้ `automaxprocs` |
| **GOMEMLIMIT** | Go 1.19+ — ตั้ง ~80% ของ memory limit เพื่อให้ GC ทำงานก่อน OOM |
| **Connection Pool** | `DB_MAX_CONNS` ต้องไม่ใหญ่เกิน memory limit (แต่ละ conn ~2-5 KB) |
| **Goroutine** | goroutine เริ่มที่ 2-8 KB stack — มี 10k goroutine ≈ 80 MB |

```yaml
# ตัวอย่าง: ตั้ง GOMEMLIMIT ใน container env
env:
  - name: GOMEMLIMIT
    value: "200MiB"          # 80% ของ memory limit 256Mi
  - name: GOMAXPROCS
    value: "1"               # CPU limit 500m ≈ 0.5 core → ใช้ 1 thread
```

---

## 6. ResourceQuota — จำกัด Resource ระดับ Namespace

**ResourceQuota** = กำหนดเพดาน resource ทั้ง namespace — ป้องกัน team ใช้ resource เกิน

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: anc-portal-quota
  namespace: anc-portal
spec:
  hard:
    # จำกัดจำนวน resources
    pods: "20"                        # Pod สูงสุด 20 ตัว
    services: "10"                    # Service สูงสุด 10 ตัว
    configmaps: "15"                  # ConfigMap สูงสุด 15 ตัว
    secrets: "10"                     # Secret สูงสุด 10 ตัว
    persistentvolumeclaims: "5"       # PVC สูงสุด 5 ตัว

    # จำกัด compute resources (รวมทุก Pod)
    requests.cpu: "4"                 # CPU request รวม ≤ 4 core
    requests.memory: "4Gi"            # Memory request รวม ≤ 4 GB
    limits.cpu: "8"                   # CPU limit รวม ≤ 8 core
    limits.memory: "8Gi"              # Memory limit รวม ≤ 8 GB
```

### คำสั่ง

```powershell
# Apply
kubectl apply -f resource-quota.yaml

# ดู quota usage
kubectl describe resourcequota anc-portal-quota -n anc-portal

# ตัวอย่าง output:
# Name:            anc-portal-quota
# Resource         Used    Hard
# --------         ----    ----
# pods             5       20
# requests.cpu     450m    4
# requests.memory  448Mi   4Gi
```

### แนะนำสำหรับโปรเจคนี้

| Environment | CPU Request Quota | Memory Request Quota | Max Pods |
|---|---|---|---|
| **Staging** | 2 cores | 2Gi | 15 |
| **Production** | 8 cores | 8Gi | 30 |

---

## 7. LimitRange — Default Resource ให้ Pod

**LimitRange** = ตั้งค่า default resource ให้ container ที่ไม่ได้ระบุ + กำหนด min/max

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: anc-portal-limits
  namespace: anc-portal
spec:
  limits:
    # Default สำหรับ Container
    - type: Container
      default:                         # ← ค่า limit default (ถ้าไม่ระบุ)
        cpu: "500m"
        memory: "256Mi"
      defaultRequest:                  # ← ค่า request default (ถ้าไม่ระบุ)
        cpu: "100m"
        memory: "128Mi"
      min:                             # ← ค่าต่ำสุดที่อนุญาต
        cpu: "10m"
        memory: "32Mi"
      max:                             # ← ค่าสูงสุดที่อนุญาต
        cpu: "2"
        memory: "2Gi"

    # Default สำหรับ Pod (รวมทุก container ใน pod)
    - type: Pod
      max:
        cpu: "4"
        memory: "4Gi"
```

### ผลลัพธ์

| สถานการณ์ | พฤติกรรม |
|---|---|
| Container ไม่ระบุ resource | ได้ default: CPU 100m/500m, Mem 128Mi/256Mi |
| Container ระบุ limit > max | **Reject** — ไม่สร้าง Pod |
| Container ระบุ request < min | **Reject** — ไม่สร้าง Pod |

### คำสั่ง

```powershell
# Apply
kubectl apply -f limit-range.yaml

# ดู LimitRange
kubectl describe limitrange anc-portal-limits -n anc-portal
```

---

## 8. Quality of Service (QoS) Class

K8s จัดลำดับความสำคัญของ Pod เป็น 3 class — ส่งผลเมื่อ node resource ไม่พอ

### 3 QoS Classes

| QoS Class | เงื่อนไข | ลำดับ Kill (node เต็ม) |
|---|---|---|
| **Guaranteed** | request = limit (ทุก container) | ❸ Kill สุดท้าย (ปลอดภัยสุด) |
| **Burstable** | request < limit (อย่างน้อย 1 container) | ❷ Kill ก่อน Guaranteed |
| **BestEffort** | ไม่มี request/limit เลย | ❶ Kill แรกสุด |

### Pod ของเราเป็น QoS อะไร?

| Component | Request | Limit | QoS Class |
|---|---|---|---|
| **API** | CPU 100m, Mem 128Mi | CPU 500m, Mem 256Mi | **Burstable** ← request ≠ limit |
| **Worker** | CPU 50m, Mem 64Mi | CPU 250m, Mem 128Mi | **Burstable** |
| **API (Guaranteed ถ้าต้องการ)** | CPU 500m, Mem 256Mi | CPU 500m, Mem 256Mi | **Guaranteed** |

### เมื่อไหร่ควรใช้ Guaranteed?

| สถานการณ์ | แนะนำ |
|---|---|
| **Production API หลัก** | ✅ Guaranteed — กัน OOM ตอน node เต็ม |
| **Staging / Dev** | Burstable ก็พอ — ประหยัด resource |
| **Worker / Job** | Burstable — burst ได้เวลา process หนัก |

### ตัวอย่าง Guaranteed

```yaml
# request = limit → QoS = Guaranteed
resources:
  requests:
    cpu: "500m"
    memory: "256Mi"
  limits:
    cpu: "500m"         # ← เท่ากับ request
    memory: "256Mi"     # ← เท่ากับ request
```

> ⚠️ **ข้อเสีย Guaranteed:** ใช้ resource เต็มที่ตลอด ไม่ได้ burst — แพงกว่า

---

## 9. Probe Tuning — ตั้งค่า Health Check

### Probe Spec ปัจจุบัน

#### API Server

```yaml
# Startup: ให้เวลา boot (connect DB, warm up)
startupProbe:
  httpGet:
    path: /healthz
    port: http               # 20000
  periodSeconds: 5           # ตรวจทุก 5 วินาที
  failureThreshold: 10       # fail 10 ครั้ง → restart (รวม 50 วินาที)
  initialDelaySeconds: 2     # รอ 2 วินาทีก่อนเริ่มตรวจ

# Liveness: ตรวจว่า process alive
livenessProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 5
  periodSeconds: 15          # ตรวจทุก 15 วินาที
  timeoutSeconds: 3          # timeout 3 วินาที
  failureThreshold: 3        # fail 3 ครั้ง → restart

# Readiness: ตรวจว่าพร้อมรับ traffic
readinessProbe:
  httpGet:
    path: /ready
    port: http
  initialDelaySeconds: 3
  periodSeconds: 10          # ตรวจทุก 10 วินาที
  timeoutSeconds: 3
  failureThreshold: 2        # fail 2 ครั้ง → ถอดจาก Service
```

#### Worker

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 20001              # Worker health port
  initialDelaySeconds: 10    # Kafka consumer ต้องใช้เวลา connect
  periodSeconds: 30
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /healthz
    port: 20001
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

### แนวทางตั้งค่า Probe

| Parameter | คำแนะนำ | เหตุผล |
|---|---|---|
| **initialDelaySeconds** | = เวลา boot ของ app (ดูจาก log) | ไม่ fail ก่อน boot เสร็จ |
| **periodSeconds** | Liveness: 10-30s, Readiness: 5-10s | Readiness ถี่กว่า เพราะต้องรับ traffic เร็ว |
| **timeoutSeconds** | 2-5s | ถ้า health endpoint ช้ากว่า 5s = มีปัญหา |
| **failureThreshold** | Liveness: 3, Readiness: 2-3 | Readiness ตอบเร็วกว่า liveness restart |
| **successThreshold** | Readiness: 1 (default) | กลับมา 1 ครั้งก็พอ |

### Timeline ตัวอย่าง — Boot → Ready

```
0s    Pod สร้าง
2s    startupProbe เริ่มตรวจ
      → /healthz → fail (DB ยังไม่ connect)
7s    → /healthz → fail
12s   → /healthz → pass ✅ (DB connected)
      → startupProbe หยุด → liveness + readiness เริ่ม
15s   livenessProbe: /healthz → pass ✅
15s   readinessProbe: /ready → pass ✅ → Pod เข้า Service → รับ traffic
```

### Anti-patterns

| ❌ ผิด | ✅ ถูก |
|---|---|
| Liveness ตรวจ external dependency (DB) | Liveness ตรวจแค่ process alive |
| Readiness fail = restart | Readiness fail = ถอดออกจาก Service |
| initialDelay = 0 | initialDelay ≥ เวลา boot |
| periodSeconds = 1 (ถี่เกิน) | periodSeconds ≥ 5 |
| failureThreshold = 1 (ไว้เกิน) | failureThreshold ≥ 2 |

> **หมายเหตุ:** โปรเจคนี้ liveness ตรวจ DB ด้วย (`/healthz` เช็ค DB ping)
> ซึ่ง **ยอมรับได้** เพราะ Go app ตัวเดียวที่มี DB เป็น core dependency
> แต่ถ้าอนาคต DB ล่มบ่อย ควรแยก liveness เป็น `/livez` ที่ตรวจแค่ process

---

## 10. HPA Tuning — ตั้งค่า Auto-scaling

### HPA Spec ปัจจุบัน

```yaml
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: anc-portal-api
  minReplicas: 2
  maxReplicas: 6
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70     # เพิ่ม Pod ถ้า avg CPU > 70%
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80     # เพิ่ม Pod ถ้า avg Memory > 80%
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60   # รอ 60 วินาทีก่อนเพิ่ม
      policies:
        - type: Pods
          value: 2                     # เพิ่มครั้งละ 2 pods
          periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300  # รอ 5 นาทีก่อนลด
      policies:
        - type: Pods
          value: 1                     # ลดครั้งละ 1 pod
          periodSeconds: 60
```

### กฎการ Scale

```
                scale up
average CPU > 70% ──────────▶ เพิ่ม Pod (max 2 ต่อรอบ)
                ◄──────────── (stabilization 60s)
                 scale down
average CPU < 70% ──────────▶ ลด Pod (max 1 ต่อรอบ)
                ◄──────────── (stabilization 300s)
```

### แนวทางเลือก Target

| Target | ค่าแนะนำ | เหตุผล |
|---|---|---|
| **CPU 50-60%** | latency-sensitive API | เผื่อ headroom เยอะ ตอบเร็ว |
| **CPU 70%** ← ใช้อยู่ | balanced | ดีสำหรับ Go app ทั่วไป |
| **CPU 80-90%** | throughput-oriented | เหมาะกับ batch/worker |
| **Memory 80%** ← ใช้อยู่ | ปกติ | Memory ไม่ค่อยขึ้นลงแรงเท่า CPU |

### Custom Metrics (อนาคต)

```yaml
# Scale ตาม request rate (ต้องมี Prometheus Adapter)
- type: Pods
  pods:
    metric:
      name: http_requests_per_second
    target:
      type: AverageValue
      averageValue: "100"      # เพิ่ม Pod ถ้า avg > 100 req/s ต่อ Pod
```

---

## 11. ตาราง Spec แนะนำตาม Workload

### Go HTTP API

| ขนาด Traffic | CPU Req/Limit | Mem Req/Limit | Replicas | HPA |
|---|---|---|---|---|
| **Small** (< 100 req/s) | 50m / 250m | 64Mi / 128Mi | 2 | 2-4 |
| **Medium** (100-500 req/s) | 100m / 500m | 128Mi / 256Mi | 2-3 | 2-6 |
| **Large** (500-2000 req/s) | 200m / 1000m | 256Mi / 512Mi | 3-5 | 3-10 |
| **XL** (> 2000 req/s) | 500m / 2000m | 512Mi / 1Gi | 5+ | 5-20 |

### Kafka Consumer (Worker)

| ขนาด Throughput | CPU Req/Limit | Mem Req/Limit | Replicas |
|---|---|---|---|
| **Low** (< 100 msg/s) | 50m / 250m | 64Mi / 128Mi | 1 |
| **Medium** (100-500 msg/s) | 100m / 500m | 128Mi / 256Mi | 2 |
| **High** (> 500 msg/s) | 200m / 1000m | 256Mi / 512Mi | 3+ |

### Database Migration Job

| ขนาด Migration | CPU Req/Limit | Mem Req/Limit |
|---|---|---|
| **ปกติ** (schema change) | 50m / 200m | 64Mi / 128Mi |
| **Data migration** (ย้ายข้อมูลเยอะ) | 200m / 1000m | 256Mi / 512Mi |

### Sync CronJob

| ขนาดข้อมูล | CPU Req/Limit | Mem Req/Limit |
|---|---|---|
| **Small** (< 10k rows) | 50m / 250m | 64Mi / 128Mi |
| **Medium** (10k-100k rows) | 100m / 500m | 128Mi / 256Mi |
| **Large** (> 100k rows) | 200m / 1000m | 256Mi / 512Mi |

---

## 12. Capacity Planning — วางแผนทรัพยากร

### สูตรคำนวณ Node Capacity

```
จำนวน Node ขั้นต่ำ = (total CPU request ÷ Node CPU × 0.7) + 1 spare

ตัวอย่าง Production:
  total CPU request = 900m (3 API × 200m + 2 Worker × 50m + buffer)
  Node = 4 vCPU
  จำนวน Node = (0.9 ÷ 4 × 0.7) + 1 ≈ 2 nodes (but recommend 3 for HA)
```

### Production Capacity Planning

```
┌───────────────────────────────────────────────────┐
│ Component        │ Pods │ CPU Req │ Mem Req │ Total│
├──────────────────┼──────┼─────────┼─────────┼──────┤
│ API (HPA max)    │   8  │  200m   │  256Mi  │ 1.6c │ 2Gi  │
│ Worker           │   2  │   50m   │   64Mi  │ 100m │ 128Mi│
│ Sync (peak)      │   1  │  100m   │  128Mi  │ 100m │ 128Mi│
│ System overhead  │   -  │    -    │    -    │ 500m │ 512Mi│
├──────────────────┼──────┼─────────┼─────────┼──────┤
│ TOTAL            │  11  │    -    │    -    │ 2.3c │ 2.8Gi│
└───────────────────────────────────────────────────┘

แนะนำ: 3 nodes × 2 vCPU × 4Gi = 6 vCPU, 12Gi
  → CPU headroom: ~60% (รองรับ burst + system pods)
  → Memory headroom: ~75%
```

### Cost Estimation (Cloud)

| Provider | Node Type | vCPU | Memory | $/month (approx) |
|---|---|---|---|---|
| **GKE** | e2-medium | 2 | 4Gi | ~$35 |
| **EKS** | t3.medium | 2 | 4Gi | ~$30 |
| **AKS** | Standard_B2s | 2 | 4Gi | ~$30 |

> 3 nodes × ~$33/month ≈ **$100/month** สำหรับ production cluster ขนาดเล็ก

---

## 13. Troubleshooting — ปัญหา Resource

### OOMKilled (Memory ไม่พอ)

```powershell
# ตรวจว่า Pod ถูก OOMKilled
kubectl describe pod <pod-name> -n anc-portal | Select-String -Pattern "OOMKilled|Reason|Last State"

# ดู memory usage ปัจจุบัน
kubectl top pods -n anc-portal

# แก้: เพิ่ม memory limit
# เช่น 256Mi → 512Mi
```

| อาการ | สาเหตุ | วิธีแก้ |
|---|---|---|
| Pod restart + OOMKilled | Memory limit ต่ำเกินไป | เพิ่ม memory limit |
| Memory usage ค่อยๆ เพิ่ม (leak) | Memory leak ใน app | ตรวจ pprof, fix goroutine/connection leak |
| Memory พุ่งเวลา query ใหญ่ | ดึงข้อมูลเยอะไม่ paginate | ใช้ pagination, streaming |

### CPU Throttling (CPU ไม่พอ)

```powershell
# ดู CPU usage
kubectl top pods -n anc-portal

# ดู throttling (ต้องมี cAdvisor metrics)
# ถ้า CPU usage ≈ CPU limit → กำลังถูก throttle
```

| อาการ | สาเหตุ | วิธีแก้ |
|---|---|---|
| Response ช้า | CPU limit ต่ำ + traffic สูง | เพิ่ม CPU limit หรือเพิ่ม Pod (HPA) |
| CPU เต็ม limit ตลอด | App ใช้ CPU เยอะ (heavy computation) | เพิ่ม limit + optimize code |
| HPA ไม่ scale แม้ช้า | Metrics Server ไม่ทำงาน | ติดตั้ง/restart metrics-server |

### Pod Pending (Resource ไม่พอ)

```powershell
# ดู events ของ Pod
kubectl describe pod <pod-name> -n anc-portal

# มองหา: "Insufficient cpu" หรือ "Insufficient memory"
# แปลว่า: ไม่มี node ที่รองรับ request ได้
```

| วิธีแก้ | อธิบาย |
|---|---|
| ลด resource request | ถ้าตั้งสูงเกินจริง |
| เพิ่ม node | Cluster Autoscaler จะทำอัตโนมัติ (ถ้าเปิดไว้) |
| ลบ Pod ที่ไม่ใช้ | เช่น Job เก่าที่ค้างอยู่ |

---

## 14. Checklist ตั้งค่า Resource

### ก่อน Deploy ครั้งแรก

- [ ] ทุก container มี `requests` และ `limits`
- [ ] Memory limit ≥ 2× baseline usage ของ app
- [ ] CPU limit ≥ 3× idle CPU usage
- [ ] Health probe `initialDelaySeconds` ≥ เวลา boot จริง
- [ ] HPA `minReplicas` ≥ 2 (high availability)
- [ ] PDB `minAvailable` ≥ 1

### หลัง Deploy

- [ ] `kubectl top pods` — ดู actual usage
- [ ] CPU usage ไม่เกิน 80% ของ limit ปกติ
- [ ] Memory usage ไม่เกิน 70% ของ limit ปกติ
- [ ] HPA scale up/down ทำงานถูกต้อง
- [ ] ไม่มี OOMKilled ใน events

### ทุก Sprint / การ Review

- [ ] Review resource usage trend (Grafana dashboard)
- [ ] ปรับ request/limit ถ้า usage เปลี่ยนมาก (± 50%)
- [ ] ตรวจ ResourceQuota usage vs limit
- [ ] ตรวจ node capacity headroom ≥ 30%

---

> **v1.0** — March 2026 | ANC Portal Backend Team
>
> **เอกสารหลัก:** [Kubernetes Guide](kubernetes-guide.md) — คู่มือ K8s ฉบับสมบูรณ์
