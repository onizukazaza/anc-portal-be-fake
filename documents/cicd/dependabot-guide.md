# Dependabot — คู่มือฉบับสมบูรณ์

> ทุกอย่างเกี่ยวกับ Dependabot ในโปรเจกต์นี้ — ตั้งแต่มันคืออะไร ทำงานยังไง ไปจนถึงวิธีจัดการ PR

---

## สารบัญ

- [Part 1: Dependabot คืออะไร](#part-1-dependabot-คืออะไร)
- [Part 2: Config ของโปรเจกต์นี้](#part-2-config-ของโปรเจกต์นี้)
- [Part 3: วิธีจัดการ PR ของ Dependabot](#part-3-วิธีจัดการ-pr-ของ-dependabot)
- [Part 4: สถานการณ์พิเศษ](#part-4-สถานการณ์พิเศษ)
- [Part 5: Dependabot Commands](#part-5-dependabot-commands)
- [Part 6: FAQ — คำถามที่พบบ่อย](#part-6-faq--คำถามที่พบบ่อย)
- [Cheatsheet](#cheatsheet)

---

## Part 1: Dependabot คืออะไร

### อธิบายง่าย ๆ

**Dependabot = พนักงานตรวจสต๊อกอัตโนมัติ**

```
ทุกสัปดาห์ Dependabot จะเข้ามาดูว่า:
  - Library ที่ใช้อยู่ มี version ใหม่ไหม?
  - Docker image ที่ใช้อยู่ เก่าไปหรือยัง?
  - GitHub Actions version ล้าสมัยไหม?

ถ้าเจอของใหม่ → สร้าง Pull Request ให้อัตโนมัติ
เราแค่ review แล้ว merge ได้เลย
```

### ทำไมต้องมี?

| ถ้าไม่มี Dependabot | ถ้ามี Dependabot |
|---------------------|------------------|
| ต้องเข้าไปเช็ค version เอง | ตรวจให้ทุกสัปดาห์อัตโนมัติ |
| มีช่องโหว่ (CVE) ก็ไม่รู้ | แจ้งเตือน + สร้าง PR แก้ให้ทันที |
| ปล่อยนาน อัพทีเดียวพังหมด | ค่อย ๆ อัพทีละตัว รู้ทันทีว่าตัวไหนพัง |
| ต้อง test เอง | CI รันให้อัตโนมัติใน PR |

### Flow ภาพรวมของ Dependabot

```
  ┌─────────────────────────────────────────────────────┐
  │           Dependabot ทำงานอัตโนมัติ                  │
  │                                                     │
  │   ทุกวันจันทร์ 09:00 (Asia/Bangkok)                  │
  │         │                                           │
  │         ▼                                           │
  │   ┌───────────────────────────────────┐             │
  │   │  สแกน 3 ระบบ:                     │             │
  │   │   📦 Go modules    (go.mod)       │             │
  │   │   ⚙️ GitHub Actions (workflows/)  │             │
  │   │   🐳 Docker images (Dockerfile)   │             │
  │   └──────────────┬────────────────────┘             │
  │                  │                                  │
  │            มี version ใหม่?                          │
  │            ├── ❌ ไม่มี → จบ (ไม่ทำอะไร)             │
  │            └── ✅ มี → สร้าง PR อัตโนมัติ            │
  │                        │                            │
  │                        ▼                            │
  │              ┌──────────────────┐                   │
  │              │    Pull Request  │                   │
  │              │  - title ชัดเจน  │                   │
  │              │  - label ติดให้   │                   │
  │              │  - changelog     │                   │
  │              └────────┬─────────┘                   │
  │                       │                             │
  │                       ▼                             │
  │              CI รัน (lint, test, build)              │
  │                       │                             │
  │                  ผ่านไหม?                            │
  │                  ├── ✅ → รอเรา Approve + Merge      │
  │                  └── ❌ → ต้องดู log แก้ไข           │
  └─────────────────────────────────────────────────────┘
```

**สิ่งที่ Dependabot ทำให้:** สร้าง PR + รัน CI

**สิ่งที่เราต้องทำเอง:** Review + Approve + Merge

---

## Part 2: Config ของโปรเจกต์นี้

ไฟล์: [`.github/dependabot.yml`](../../.github/dependabot.yml)

### สรุป Config

| ระบบ | ตรวจอะไร | ไฟล์ที่แก้ | PR สูงสุด | Labels |
|------|---------|-----------|----------|--------|
| **Go modules** | Library ใน `go.mod` | `go.mod`, `go.sum` | 5 ตัว/สัปดาห์ | `dependencies`, `go` |
| **GitHub Actions** | Actions ใน workflows | `.github/workflows/*.yml` | 5 ตัว/สัปดาห์ | `dependencies`, `ci` |
| **Docker images** | Base image ใน Dockerfile | `Dockerfile`, `Dockerfile.worker` | 3 ตัว/สัปดาห์ | `dependencies`, `docker` |

### ตารางเวลา

```
ทุกวันจันทร์ 09:00 น. (เวลาไทย)
```

### Target Branch

```
Dependabot → สร้าง PR → merge เข้า branch "develop"
```

> หมายเหตุ: ตั้งค่าเป็น `target-branch: develop` เพื่อให้ merge เข้า develop ก่อนแล้วค่อย merge develop เข้า main

### Commit Message Format

| ระบบ | ตัวอย่าง commit message |
|------|------------------------|
| Go | `deps(go.mod): bump github.com/rs/zerolog from 1.34.0 to 1.35.0` |
| Actions | `ci(actions): bump actions/checkout from v4 to v5` |
| Docker | `docker(deployments/docker): bump golang from 1.25-alpine to 1.26-alpine` |

---

## Part 3: วิธีจัดการ PR ของ Dependabot

### ขั้นตอนหลัก (3 ขั้นตอน)

```
  ┌──────────────────────────────────────────┐
  │  ขั้นตอนที่ 1:  ดู CI ผ่านไหม?           │
  │       ↓                                  │
  │  ขั้นตอนที่ 2:  Approve                   │
  │       ↓                                  │
  │  ขั้นตอนที่ 3:  Merge                     │
  │       ↓                                  │
  │  เสร็จ! ✅                                │
  └──────────────────────────────────────────┘
```

---

### ขั้นตอนที่ 1: ดู CI ผ่านไหม

#### เปิดดู PR ใน VS Code

1. Sidebar ซ้าย → ไอคอน **GitHub Pull Requests**
2. หมวด **All Open** → คลิก PR ของ Dependabot
3. หน้า PR เปิดทางขวา → เลื่อนลงดู **Checks**

#### อ่าน CI Results

```
✅ Lint          ผ่าน     → code style OK
✅ Test          ผ่าน     → unit tests ผ่าน
✅ Vuln Check    ผ่าน     → ไม่มีช่องโหว่
✅ Build         ผ่าน     → compile ได้
⏭ Docker        ข้าม     → ปกติสำหรับ PR
⏭ Image Scan    ข้าม     → ปกติสำหรับ PR
✅ Notify        ผ่าน     → แจ้ง Discord แล้ว
```

#### ตัดสินใจ

| สถานะ | ไอคอน | ทำอะไรต่อ |
|-------|-------|----------|
| ผ่านทั้งหมด | ✅ เขียว | → ไป Approve เลย |
| กำลังรัน | 🟡 เหลือง | → รอให้เสร็จก่อน |
| มี Failed | ❌ แดง | → [ดูหัวข้อ CI Failed](#กรณี-ci-failed) |

---

### ขั้นตอนที่ 2: Approve

> Approve = อนุมัติว่าโค้ดนี้ OK พร้อม merge

#### วิธี Approve ใน VS Code

1. เปิดหน้า PR → เลื่อนลง
2. ใต้กล่อง comment → เลือก **Approve** จาก dropdown
3. กด **Submit**

#### วิธี Approve บน GitHub.com

1. เปิด PR → แท็บ **Files changed**
2. มุมขวาบน → กดปุ่มเขียว **Review changes**
3. เลือก **Approve** → กด **Submit review**

---

### ขั้นตอนที่ 3: Merge

> Merge = รวมโค้ดเข้า branch dev

#### วิธี Merge ใน VS Code

1. เปิดหน้า PR → เลื่อนลงล่างสุด
2. กดปุ่ม **Merge Pull Request**
3. (แนะนำ) กด ▼ เลือก **Squash and Merge** → commit message สะอาด
4. กด Confirm

#### วิธี Merge บน GitHub.com

1. เลื่อนลงล่างสุดในหน้า PR
2. กด ▼ เลือก **Squash and merge**
3. แก้ commit message ถ้าต้องการ → กด **Confirm squash and merge**

#### หลัง Merge

```powershell
# ดึงโค้ดล่าสุดมาเครื่อง
git checkout develop
git pull
```

> Branch ของ Dependabot จะถูกลบอัตโนมัติหลัง merge

---

### Checkout ทดสอบ (ไม่จำเป็นเสมอ)

> ⚡ ถ้า CI ผ่านและเป็นแค่ patch/minor → ข้ามขั้นตอนนี้ Approve ได้เลย

#### ควร Checkout เมื่อ

- Major version bump (v1 → v2)
- Dependency หลัก ๆ (Go version, Fiber, pgx)
- CI ผ่านแต่ยังไม่มั่นใจ

#### วิธี Checkout

**VS Code:** กดปุ่ม **Checkout** (ลูกศร →) ข้าง PR ใน sidebar

**Terminal:**
```powershell
git fetch
git checkout dependabot/go_modules/github.com/rs/zerolog-1.35.0

# ทดสอบ
go build ./...
go test ./...

# กลับ develop
git checkout develop
```

---

## Part 4: สถานการณ์พิเศษ

### กรณี CI Failed

```
CI Failed
   │
   ├── กด "Details" / "Show" ดู log
   │
   ├── Test Failed?
   │    → version ใหม่มี breaking change
   │    → ต้อง checkout มาแก้โค้ดให้ compatible
   │    → push → CI รันใหม่อัตโนมัติ
   │
   ├── Lint Failed?
   │    → อาจมี API เปลี่ยน format
   │    → ส่วนใหญ่แก้ง่าย
   │
   ├── Build Failed?
   │    → version ใหม่ไม่ compatible กับ Go/Docker
   │    → ปิด PR ไปก่อน (Dependabot สร้างใหม่สัปดาห์ถัดไป)
   │
   └── ไม่แน่ใจ?
        → Comment ใน PR ถามทีม
        → หรือปิด PR ไว้ก่อน
```

### PR หลายตัวพร้อมกัน — merge ลำดับไหน

```
กฎง่าย ๆ:

1. แก้คนละไฟล์ → merge ลำดับไหนก็ได้
2. แก้ไฟล์เดียวกัน → merge ทีละตัว (Dependabot จะ rebase ให้)
```

#### ตัวอย่างลำดับ merge

```
PR #1: bump zerolog     (แก้ go.mod)        ← merge ก่อน
PR #2: bump validator   (แก้ go.mod)        ← merge ทีหลัง
PR #3: bump golang      (แก้ Dockerfile)    ← merge เมื่อไหร่ก็ได้
PR #4: bump alpine      (แก้ Dockerfile)    ← merge หลัง #3
```

#### ถ้าเกิด Conflict

Dependabot จะ rebase ให้อัตโนมัติหลัง PR อื่น merge แล้ว

ถ้าไม่ rebase เอง → comment ใน PR: `@dependabot rebase`

### Major Version Bump — ต้องระวัง

```
⚠️ สัญญาณว่าต้องระวัง:

  - Title มีคำว่า "v1 to v2" หรือ "1.x to 2.x"
  - CI Failed
  - Dependabot เขียนว่ามี "Breaking changes"

วิธีจัดการ:
  1. Checkout มาทดสอบจริงบนเครื่อง
  2. อ่าน changelog / release notes
  3. แก้โค้ดถ้ามี breaking change
  4. ถ้ายังไม่พร้อม → ปิด PR ไว้ก่อน
```

### ปิด PR ที่ไม่ต้องการ

- **VS Code:** เปิด PR → กด **Close** ด้านล่าง
- **GitHub:** กด **Close pull request**

> Dependabot จะสร้าง PR ใหม่ในรอบถัดไปถ้ายังมี version ใหม่กว่า

---

## Part 5: Dependabot Commands

comment ใน PR เพื่อสั่ง Dependabot:

| Command | ทำอะไร |
|---------|--------|
| `@dependabot rebase` | Rebase branch ให้ up-to-date กับ develop |
| `@dependabot recreate` | ปิด PR แล้วสร้างใหม่ทั้งหมด |
| `@dependabot merge` | Merge PR อัตโนมัติ (ถ้า CI ผ่าน + approved) |
| `@dependabot squash and merge` | Squash แล้ว merge |
| `@dependabot cancel merge` | ยกเลิก auto-merge |
| `@dependabot close` | ปิด PR |
| `@dependabot ignore this dependency` | ไม่ track dependency นี้อีกต่อไป |
| `@dependabot ignore this major version` | ข้าม major version นี้ |
| `@dependabot ignore this minor version` | ข้าม minor version นี้ |

#### ตัวอย่างการใช้

```
สมมติ Dependabot สร้าง PR bump library X จาก v2 เป็น v3
แต่เรายังไม่พร้อมอัพ v3:

→ Comment: @dependabot ignore this major version

Dependabot จะไม่สร้าง PR สำหรับ v3 อีก
แต่ยังอัพ v2.x.x ให้ตามปกติ
```

---

## Part 6: FAQ — คำถามที่พบบ่อย

### Q: Dependabot สร้าง PR ตอนไหน?
**A:** ทุกวันจันทร์ 09:00 น. (เวลาไทย) หรือเมื่อตรวจพบ security vulnerability

### Q: ถ้า merge แล้วพัง ทำไง?
**A:** `git revert <commit>` เพื่อย้อนกลับ → หรือปิด PR แล้วให้ Dependabot สร้างใหม่สัปดาห์ถัดไป

### Q: Approve ตัวเอง PR ตัวเองได้ไหม?
**A:** Dependabot PR ไม่ใช่ PR ของเรา — ใครมี access ก็ approve ได้ (ยกเว้น PR ของตัวเอง)

### Q: Dependabot PR merge เข้า branch ไหน?
**A:** เข้า branch `develop` (ตั้งค่าใน `target-branch: develop`)

### Q: ทำไม PR บางตัวขึ้น "Skipped"?
**A:** Docker + Image Scan steps ปกติจะ skip ใน PR — ไม่ใช่ปัญหา

### Q: Label มีไว้ทำอะไร?
**A:** ใช้ filter PR ตามประเภท เช่น ดูเฉพาะ `go` หรือเฉพาะ `docker`

### Q: Dependabot ฟรีไหม?
**A:** ฟรี — เป็นฟีเจอร์ในตัวของ GitHub ไม่มีค่าใช้จ่าย

---

## Cheatsheet

```
┌──────────────────────────────────────────────────────────────┐
│                  Dependabot — Quick Reference                │
│                                                              │
│  📅 รันเมื่อไหร่?     ทุกวันจันทร์ 09:00                      │
│  🎯 merge เข้าไหน?    branch develop                          │
│  📦 ตรวจอะไร?         Go modules / Actions / Docker images   │
│                                                              │
│  ✅ ขั้นตอนจัดการ PR:                                         │
│     1. ดู CI  → ผ่าน? ✅                                      │
│     2. Approve                                               │
│     3. Squash and Merge                                      │
│     4. git checkout develop && git pull                      │
│                                                              │
│  ❌ CI fail?                                                  │
│     → ดู log → แก้ / ปิด PR                                   │
│                                                              │
│  🔀 หลาย PR?                                                 │
│     → merge ทีละตัว → Dependabot rebase ให้เอง               │
│                                                              │
│  💬 สั่ง Dependabot:                                          │
│     @dependabot rebase          → rebase branch              │
│     @dependabot squash and merge → merge ให้                 │
│     @dependabot close           → ปิด PR                     │
│     @dependabot ignore this major version → ข้าม major       │
│                                                              │
│  📁 Config:  .github/dependabot.yml                          │
└──────────────────────────────────────────────────────────────┘
```

---

## ไฟล์ที่เกี่ยวข้อง

| ไฟล์ | หน้าที่ |
|------|---------|
| `.github/dependabot.yml` | Config ของ Dependabot |
| `.github/workflows/ci.yml` | CI pipeline ที่รันเมื่อ Dependabot เปิด PR |
| `go.mod` / `go.sum` | Go dependencies ที่ Dependabot ตรวจ |
| `deployments/docker/Dockerfile` | Docker base images ที่ Dependabot ตรวจ |
| `deployments/docker/Dockerfile.worker` | Docker base images (worker) |
