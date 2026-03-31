# GitHub Actions Workflow Guide — ANC Portal Backend

> **v1.0** — Last updated: March 2026
>
> วิธีเพิ่ม Workflow ขึ้น GitHub ตั้งแต่แก้ Token จนถึง Push สำเร็จ
> พร้อมอธิบายว่าแต่ละ Workflow ทำอะไร

---

## สารบัญ

1. [ปัญหาที่เจอ](#1-ปัญหาที่เจอ)
2. [วิธีแก้ — อัพเดท Token](#2-วิธีแก้--อัพเดท-token)
3. [Push Workflow ขึ้น GitHub](#3-push-workflow-ขึ้น-github)
4. [Workflow ทั้งหมดที่มี](#4-workflow-ทั้งหมดที่มี)
5. [CI Pipeline อธิบายทุก Job](#5-ci-pipeline-อธิบายทุก-job)
6. [Deploy Pipeline](#6-deploy-pipeline)
7. [Secrets & Variables ที่ต้องตั้ง](#7-secrets--variables-ที่ต้องตั้ง)
8. [Troubleshooting](#8-troubleshooting)

---

## 1. ปัญหาที่เจอ

เวลา `git push` แล้วมีไฟล์ใน `.github/workflows/` จะเจอ error นี้:

```
! [remote rejected] main -> main
(refusing to allow an OAuth App to create or update workflow
`.github/workflows/ci.yml` without `workflow` scope)
```

### สาเหตุ

GitHub ป้องกันไม่ให้ push ไฟล์ workflow (CI/CD) ด้วย token ที่ไม่มีสิทธิ์เพียงพอ
เพราะ workflow สามารถรัน code อะไรก็ได้บน GitHub — จึงต้องการ **`workflow` scope** เพิ่มเติม

---

## 2. วิธีแก้ — อัพเดท Token

### ขั้นตอน (ทำครั้งเดียว)

```
Step 1: เปิดเบราว์เซอร์ → github.com

Step 2: คลิกรูปโปรไฟล์ (มุมขวาบน) → Settings

Step 3: เลื่อนลงที่เมนูซ้าย → Developer settings (ล่างสุด)

Step 4: เลือก Personal access tokens → Tokens (classic)

Step 5: คลิกชื่อ token ที่ใช้อยู่ → กด Edit / Regenerate

Step 6: หาหัวข้อ "workflow" → ติ๊ก ☑ workflow

Step 7: กด "Update token" (หรือ "Regenerate token")

Step 8: (ถ้า regenerate) copy token ใหม่ → update ใน credential manager
```

### ตำแหน่งของ workflow scope

```
☑ repo           ← Full control of private repositories
☑ workflow       ← Update GitHub Action workflows  ← ✅ ติ๊กอันนี้
☐ write:packages
☐ delete:packages
...
```

### ถ้าใช้ Fine-Grained Token (แบบใหม่)

```
Repository permissions:
  Actions      → Read and write
  Contents     → Read and write
  Workflows    → Read and write    ← ✅ ต้องมี
```

### ถ้า Regenerate Token แล้ว — อัพเดท Credential

Windows จะจำ token เก่าไว้ ต้องอัพเดท:

```powershell
# วิธี 1: ให้ git ถามใหม่ตอน push
# Windows Credential Manager → ลบ entry ชื่อ "git:https://github.com"
# แล้ว git push จะถาม password ใหม่ → วาง token ใหม่

# วิธี 2: ใช้ command
cmdkey /delete:git:https://github.com
# แล้ว git push จะถาม username/password
# username = GitHub username
# password = token ที่ copy มา
```

---

## 3. Push Workflow ขึ้น GitHub

หลังอัพเดท token แล้ว:

```powershell
# ตรวจว่า workflow files อยู่ใน git
git status

# ถ้าเห็น workflow files ใน staged/committed แล้ว
git push

# ถ้ายังไม่ได้ add
git add .github/workflows/
git commit -m "ci: add GitHub Actions workflows"
git push
```

### ยืนยันว่าสำเร็จ

```
เปิด GitHub repo → tab "Actions" → จะเห็น workflow รันอัตโนมัติ
```

---

## 4. Workflow ทั้งหมดที่มี

```
.github/
├── dependabot.yml                          ← อัพเดท dependency อัตโนมัติ
├── release.yml                             ← สร้าง release notes
└── workflows/
    ├── ci.yml                              ← CI Pipeline (หลัก)
    ├── deploy-staging.yml                  ← Deploy ไป Staging
    ├── deploy-production.yml               ← Deploy ไป Production
    └── release.yml                         ← สร้าง Release + Tag
```

### สรุปแต่ละ Workflow

| ไฟล์ | ทำงานเมื่อ | ทำอะไร |
|------|-----------|--------|
| **ci.yml** | push/PR ไป `develop`, `main` | Lint → Test → Vuln → Build → Docker → Scan → Notify |
| **deploy-staging.yml** | push ไป `develop` (หลัง CI ผ่าน) | Deploy ไป K8s staging |
| **deploy-production.yml** | push ไป `main` (ต้อง manual approve) | Deploy ไป K8s production |
| **release.yml** | สร้าง tag `v*` | สร้าง GitHub Release + release notes |
| **dependabot.yml** | ทุกสัปดาห์ | เช็ค dependency update → สร้าง PR อัตโนมัติ |

---

## 5. CI Pipeline อธิบายทุก Job

### Flow

```
push/PR
  │
  ├──→ [1] Lint        (golangci-lint)
  ├──→ [2] Test        (go test + coverage)
  └──→ [3] Vuln        (govulncheck)
         │
         ▼
       [4] Build       (compile ทุก binary)
         │
         ▼
       [5] Docker      (build + push image → GHCR)
         │
         ▼
       [6] Scan        (Trivy security scan)
         │
         ▼
       [7] Notify      (Discord notification)
```

### Job 1 — Lint

```yaml
golangci-lint run ./...
```

| รายการ | รายละเอียด |
|--------|-----------|
| ใช้ | golangci-lint (17 linters จาก `.golangci.yml`) |
| ตรวจ | bugs, performance, security, style |
| ถ้า FAIL | Build job ไม่รัน |

### Job 2 — Test + Coverage

```yaml
go test -race -coverprofile=coverage.out -covermode=atomic -count=1 ./...
```

| รายการ | รายละเอียด |
|--------|-----------|
| `-race` | ตรวจ race condition |
| `-coverprofile` | สร้าง coverage report |
| `-count=1` | ไม่ใช้ cache |
| ผลลัพธ์ | Coverage % แสดงใน GitHub Summary |

### Job 3 — Vulnerability Check

```yaml
govulncheck ./...
```

| รายการ | รายละเอียด |
|--------|-----------|
| ตรวจ | dependency ที่มีช่องโหว่ |
| ฐานข้อมูล | Go Vulnerability Database |

### Job 4 — Build

```yaml
go build -ldflags="..." -o /dev/null ./cmd/api
go build -ldflags="..." -o /dev/null ./cmd/worker
go build -ldflags="..." -o /dev/null ./cmd/migrate
...
```

| รายการ | รายละเอียด |
|--------|-----------|
| ทำอะไร | ตรวจว่า compile ผ่าน + ldflags (GitCommit, BuildTime) ถูกต้อง |
| ต้องรอ | Lint + Test + Vuln ผ่านก่อน |

### Job 5 — Docker Build + Push

| รายการ | รายละเอียด |
|--------|-----------|
| Registry | GitHub Container Registry (ghcr.io) |
| Images | `ghcr.io/onizukazaza/anc-portal-be-fake` (API + Worker) |
| Tags | branch name, SHA short, `latest` (main only) |
| เมื่อไหร่ | push event เท่านั้น (PR ไม่ push image) |

### Job 6 — Security Scan (Trivy)

| รายการ | รายละเอียด |
|--------|-----------|
| ทำอะไร | Scan Docker image หาช่องโหว่ |
| Severity | CRITICAL, HIGH → fail ถ้าเจอ |
| ผลลัพธ์ | Upload SARIF → GitHub Security tab |

### Job 7 — Discord Notification

| รายการ | รายละเอียด |
|--------|-----------|
| ทำอะไร | ส่งผล CI ไป Discord ทุกครั้ง (pass/fail) |
| ข้อมูล | Branch, Commit, Author, Job results, Failed steps |
| ต้องการ | `DISCORD_WEBHOOK_URL` ใน repo variables |

---

## 6. Deploy Pipeline

### Staging (อัตโนมัติ)

```
push ไป develop
     │
     ▼
CI ผ่าน → Deploy ไป K8s staging อัตโนมัติ
```

### Production (ต้อง approve)

```
push ไป main
     │
     ▼
CI ผ่าน → รอ manual approval → Deploy ไป K8s production
```

---

## 7. Secrets & Variables ที่ต้องตั้ง

ไปที่ **GitHub → repo Settings → Secrets and variables → Actions**

### Secrets (ข้อมูลลับ)

| ชื่อ | ค่า | ใช้ใน |
|------|-----|------|
| `KUBE_CONFIG` | Base64 ของ kubeconfig | deploy-staging, deploy-production |

> `GITHUB_TOKEN` ไม่ต้องตั้ง — GitHub สร้างให้อัตโนมัติ

### Variables (ข้อมูลทั่วไป)

| ชื่อ | ค่า | ใช้ใน |
|------|-----|------|
| `DISCORD_WEBHOOK_URL` | Discord webhook URL | ci.yml (notify job) |

### วิธีตั้ง

```
1. GitHub repo → Settings → Secrets and variables → Actions
2. Tab "Secrets" → กด "New repository secret"
3. Tab "Variables" → กด "New repository variable"
```

---

## 8. Troubleshooting

### ❌ "refusing to allow an OAuth App to create or update workflow"

```
สาเหตุ: Token ไม่มี workflow scope
แก้:     GitHub → Settings → Developer settings → Tokens → ติ๊ก ☑ workflow
```

### ❌ CI ไม่รัน หลัง push

```
สาเหตุ: workflow file ยังไม่ได้ push ขึ้น GitHub
ตรวจ:   GitHub → repo → tab "Actions" → มี workflow หรือยัง
แก้:     git push (ต้องมี workflow scope)
```

### ❌ Docker push failed

```
สาเหตุ: GITHUB_TOKEN ไม่มีสิทธิ์ packages:write
แก้:     repo Settings → Actions → General → Workflow permissions
         → เลือก "Read and write permissions"
```

### ❌ Discord notification ไม่ส่ง

```
สาเหตุ: ยังไม่ได้ตั้ง DISCORD_WEBHOOK_URL
แก้:     repo Settings → Secrets and variables → Actions → Variables
         → เพิ่ม DISCORD_WEBHOOK_URL
```

### ❌ Deploy failed — no KUBE_CONFIG

```
สาเหตุ: ยังไม่ได้ตั้ง KUBE_CONFIG secret
แก้:     repo Settings → Secrets → เพิ่ม KUBE_CONFIG (base64 ของ kubeconfig)
```

---

> **สรุป:** ต้องการแค่ **อัพเดท token ให้มี `workflow` scope** แล้ว `git push` ได้เลย
> CI จะรันอัตโนมัติทุกครั้งที่ push/PR — ไม่ต้องทำอะไรเพิ่ม
