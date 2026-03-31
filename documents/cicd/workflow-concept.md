# Workflow Concept — CI/CD Pipeline

## Workflow คืออะไร?

**Workflow** คือชุดคำสั่งอัตโนมัติที่ GitHub Actions จะรันให้เมื่อเกิด **event** บางอย่าง เช่น push code, สร้าง PR
หรือ tag version — โดยเราไม่ต้องเข้าไปรันเองทีละคำสั่ง

ให้นึกภาพว่า Workflow คือ **"สายพานในโรงงาน"**:
เราโยนวัตถุดิบ (code) เข้าสายพาน → ผ่านด่านตรวจสอบทีละด่าน → ออกมาเป็นสินค้าพร้อมใช้ (deployed app)

---

## ภาพรวม Pipeline ทั้งหมด

```
                 push / PR ไป develop หรือ main
                              │
                    ┌─────────▼──────────┐
                    │     CI Workflow     │
                    │                    │
                    │  ┌──────────────┐  │
                    │  │  Lint (1)    │  │   ← ตรวจโค้ดสวย + best practices
                    │  │  Test (2)    │  │   ← รัน unit tests + วัด coverage
                    │  │  Vuln (3)    │  │   ← เช็ค dependency มีช่องโหว่ไหม
                    │  └──────┬───────┘  │
                    │         │          │
                    │  ┌──────▼───────┐  │
                    │  │  Build (4)   │  │   ← compile Go binaries ผ่านไหม
                    │  └──────┬───────┘  │
                    │         │          │
                    │  ┌──────▼───────┐  │
                    │  │  Docker (5)  │  │   ← build + push Docker image ไป GHCR
                    │  └──────┬───────┘  │
                    │         │          │
                    │  ┌──────▼───────┐  │
                    │  │  Scan (6)    │  │   ← Trivy scan หาช่องโหว่ใน image
                    │  └──────┬───────┘  │
                    │         │          │
                    │  ┌──────▼───────┐  │
                    │  │  Notify (7)  │  │   ← แจ้ง Discord ว่าผ่าน/ไม่ผ่าน
                    │  └──────────────┘  │
                    └────────────────────┘
                              │
           ┌──────────────────┼──────────────────┐
           │                                     │
    CI ผ่าน + branch develop                push tag v*
           │                                     │
  ┌────────▼─────────┐              ┌────────────▼────────────┐
  │  Deploy Staging   │              │    Release Workflow      │
  │                   │              │  สร้าง GitHub Release    │
  │  migrate → deploy │              │  + changelog อัตโนมัติ   │
  │  → rollout verify │              └────────────┬────────────┘
  │  → smoke test     │                           │
  │  → notify Discord │              ┌────────────▼────────────┐
  └───────────────────┘              │  Deploy Production      │
                                     │                         │
                                     │  ⚠️ ต้อง manual approve  │
                                     │  verify image → migrate │
                                     │  → deploy → rollout     │
                                     │  → verify pods          │
                                     │  → notify Discord       │
                                     └─────────────────────────┘
```

---

## แต่ละ Workflow ทำอะไร?

### 1. CI Workflow (`ci.yml`)

> **Trigger:** push หรือ PR ไป `develop` / `main`

Pipeline หลักที่รันทุกครั้งเมื่อ push code — ทำหน้าที่ **"ตรวจสอบ"** ว่าโค้ดมีคุณภาพดีพอ

| Job | ลำดับ | หน้าที่ | เครื่องมือ |
|-----|-------|---------|-----------|
| **Lint** | 1 (ขนานกับ Test, Vuln) | ตรวจ coding style, best practices, security patterns | golangci-lint v2 |
| **Test** | 2 (ขนานกับ Lint, Vuln) | รัน unit tests + วัด coverage (race detection เปิด) | `go test -race` |
| **Vuln Check** | 3 (ขนานกับ Lint, Test) | ตรวจ dependency มี known vulnerability ไหม | govulncheck |
| **Build** | 4 (รอ 1-3 ผ่าน) | compile Go binaries ทั้ง 5 ตัว (api, worker, migrate, seed, sync) | `go build` |
| **Docker** | 5 (รอ Build ผ่าน) | build Docker images + push ไป GHCR (เฉพาะ push event) | docker buildx |
| **Image Scan** | 6 (รอ Docker ผ่าน) | สแกนช่องโหว่ใน container image (CRITICAL + HIGH) | Trivy |
| **Notify** | 7 (รันเสมอ) | แจ้งผลรวมไป Discord webhook | curl |

**สิ่งสำคัญ:**
- Job 1-3 รัน **ขนาน** กัน (ประหยัดเวลา)
- Job 4+ รัน **ต่อเนื่อง** (ต้องผ่านด่านก่อนหน้า)
- `concurrency: cancel-in-progress: true` → push ใหม่จะยกเลิก CI เก่าที่กำลังรันอยู่

```
Lint  ──┐
Test  ──┼── Build ── Docker ── Scan ── Notify
Vuln  ──┘
```

---

### 2. Deploy Staging (`deploy-staging.yml`)

> **Trigger:** CI ผ่านบน branch `develop` / manual dispatch

Auto-deploy ไป staging environment เมื่อ CI สำเร็จ

| Step | หน้าที่ |
|------|---------|
| **Determine tag** | หา image tag จาก git short SHA |
| **Configure kubeconfig** | เซ็ตค่า K8s cluster credentials |
| **Database Migration** | รัน migration job บน K8s |
| **Deploy** | apply Kustomize overlay → rolling update |
| **Rollout Verify** | รอจนทุก pod เป็น Ready |
| **Smoke Test** | port-forward แล้วยิง `/healthz` ต้องได้ 200 |
| **Notify Discord** | แจ้งผลว่า deploy สำเร็จหรือพัง |

**Guard:** `cancel-in-progress: false` → ไม่ยกเลิก deploy ที่ทำอยู่ (ป้องกัน half-deployed state)

---

### 3. Deploy Production (`deploy-production.yml`)

> **Trigger:** push tag `v*` (เช่น `v1.0.0`) / manual dispatch

Deploy ขึ้น production **ต้องผ่าน manual approval** (ตั้งใน GitHub Environment protection rules)

| Step | หน้าที่ |
|------|---------|
| **Determine tag** | ใช้ tag name หรือ input จาก manual dispatch |
| **Verify image exists** | ตรวจว่า Docker image มีอยู่ใน registry จริง |
| **Database Migration** | รัน migration (timeout 180s) |
| **Deploy** | apply Kustomize overlay → rolling update |
| **Rollout Verify** | รอ API + Worker ready (timeout 300s) |
| **Verify Pods** | เช็คว่าทุก pod อยู่ในสถานะ Running |
| **Notify Discord** | แจ้ง deploy สำเร็จ/ล้มเหลว |

**ข้อแตกต่างจาก Staging:**
- ต้อง **manual approval** (ไม่ auto deploy)
- มี **pre-flight check** ว่า image มีจริง
- Timeout **นานกว่า** (300s vs 180s)
- ไม่มี smoke test (ใช้ verify pods แทน)

---

### 4. Release (`release.yml`)

> **Trigger:** push tag `v*`

สร้าง GitHub Release อัตโนมัติ พร้อม changelog

| Step | หน้าที่ |
|------|---------|
| **Extract version** | อ่าน tag → จำแนก stable vs pre-release |
| **Find previous tag** | หา tag ก่อนหน้าเพื่อสร้าง diff |
| **Generate changelog** | สรุปจำนวน commits, compare link, build info |
| **Create Release** | สร้าง GitHub Release (ไม่ draft, auto-generated notes) |
| **Notify Discord** | แจ้งไป Discord ว่ามี release ใหม่ |

**Pre-release detection:** tag ที่มี suffix เช่น `-rc.1`, `-beta.2` จะถูกตั้งเป็น pre-release อัตโนมัติ

---

## Flow ตาม Scenario

### Scenario 1: Developer push code ปกติ

```
git push origin develop
  └── CI Workflow รัน (Lint → Test → Vuln → Build → Docker → Scan → Notify)
        └── ถ้า CI ผ่าน → Deploy Staging อัตโนมัติ
```

### Scenario 2: สร้าง Pull Request

```
git push origin feature/xxx → สร้าง PR ไป develop
  └── CI Workflow รัน (แค่ตรวจสอบ ไม่ build Docker, ไม่ deploy)
```

### Scenario 3: Release version ใหม่

```
git tag v1.0.0
git push origin v1.0.0
  └── Release Workflow → สร้าง GitHub Release + changelog
  └── Deploy Production → (รอ manual approve) → migrate → deploy → verify
```

### Scenario 4: Hotfix ฉุกเฉิน

```
git tag v1.0.1
git push origin v1.0.1
  └── เหมือน Scenario 3 แต่ approve เร็วกว่า
```

### Scenario 5: Manual deploy (emergency)

```
GitHub Actions → Deploy Production → Run workflow
  └── ใส่ image_tag ที่ต้องการ → deploy โดยไม่ต้องสร้าง tag ใหม่
```

---

## คำศัพท์ที่ควรรู้

| คำศัพท์ | ความหมาย |
|---------|----------|
| **Workflow** | ไฟล์ YAML ที่กำหนดชุดคำสั่งอัตโนมัติ (อยู่ใน `.github/workflows/`) |
| **Job** | กลุ่มของ steps ที่รันบนเครื่องเดียวกัน (1 workflow มีหลาย jobs ได้) |
| **Step** | คำสั่งเดียวใน job (เช่น checkout, build, test) |
| **Trigger / Event** | เหตุการณ์ที่ทำให้ workflow รัน (push, PR, tag, manual) |
| **Runner** | เครื่อง virtual ที่ GitHub จัดให้รัน job (ubuntu-latest) |
| **Artifact** | ไฟล์ผลลัพธ์ที่เก็บไว้หลังรัน (เช่น coverage.out) |
| **Concurrency** | ควบคุมไม่ให้ workflow เดียวกันรันซ้อน |
| **Environment** | กลุ่มตั้งค่า (staging/production) ที่กำหนด secrets + protection rules |
| **GHCR** | GitHub Container Registry — ที่เก็บ Docker images |
| **Kustomize** | เครื่องมือจัดการ K8s config แบบ overlay (base + per-env patches) |

---

## ไฟล์ที่เกี่ยวข้อง

| ไฟล์ | หน้าที่ |
|------|---------|
| `.github/workflows/ci.yml` | CI pipeline หลัก |
| `.github/workflows/deploy-staging.yml` | CD ไป staging |
| `.github/workflows/deploy-production.yml` | CD ไป production |
| `.github/workflows/release.yml` | สร้าง GitHub Release |
| `.golangci.yml` | config สำหรับ golangci-lint (CI อ่านอัตโนมัติ) |
| `deployments/docker/Dockerfile` | Multi-stage Dockerfile (builder → api / worker) |
| `deployments/k8s/` | Kustomize base + overlays (staging / production) |

---

## Security ที่ออกแบบไว้

1. **Least privilege permissions** — แต่ละ workflow ขอ permission เท่าที่จำเป็น
2. **Environment protection** — production ต้อง manual approve
3. **Pre-flight image check** — production ตรวจ image มีจริงก่อน deploy
4. **Trivy image scan** — สแกน CRITICAL + HIGH vulnerability
5. **SARIF upload** — ผลสแกนขึ้น GitHub Security tab
6. **Concurrency control** — deploy ไม่ cancel กัน ป้องกัน half-deployed
7. **Secrets management** — credentials เก็บใน GitHub Secrets เท่านั้น

---

## สรุปสั้น ๆ

> **Push code → CI ตรวจ → Docker build → Staging auto-deploy**
>
> **Tag version → Release สร้าง → Production deploy (manual approve)**
>
> ทุกขั้นตอนมี Discord notification + GitHub Summary แจ้งผลอัตโนมัติ
