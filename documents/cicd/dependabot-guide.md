# Dependabot — คู่มือฉบับเต็ม

> อ่านจบ → เข้าใจ + ใช้งานได้ทันที
>
> Config: [`.github/dependabot.yml`](../../.github/dependabot.yml)

---

## สารบัญ

| ส่วน | เนื้อหา |
|------|---------|
| [1. Dependabot คืออะไร](#1-dependabot-คืออะไร) | อธิบายภาพรวม |
| [2. ทำไมต้องมี](#2-ทำไมต้องมี) | ปัญหาที่แก้ได้ |
| [3. Config ของโปรเจกต์นี้](#3-config-ของโปรเจกต์นี้) | ตรวจอะไร เมื่อไหร่ |
| [4. Flow การทำงาน](#4-flow-การทำงาน) | ตั้งแต่ scan จนถึง merge |
| [5. เมื่อ Dependabot สร้าง PR — ทำอะไร?](#5-เมื่อ-dependabot-สร้าง-pr--ทำอะไร) | ขั้นตอน review + merge |
| [6. Checkout ทดสอบบนเครื่อง](#6-checkout-ทดสอบบนเครื่อง) | ดึงโค้ดมาลองรัน |
| [7. กรณี CI Failed](#7-กรณี-ci-failed) | แก้ปัญหา |
| [8. PR หลายตัว — Merge ยังไง](#8-pr-หลายตัว--merge-ยังไง) | ลำดับการ merge |
| [9. Dependabot Commands](#9-dependabot-commands) | สั่ง bot ผ่าน comment |
| [10. Config อธิบายทีละบรรทัด](#10-config-อธิบายทีละบรรทัด) | อ่าน YAML ได้ |
| [11. Cheatsheet](#11-cheatsheet) | สรุปทุกอย่างในหน้าเดียว |

---

## 1. Dependabot คืออะไร

**Dependabot** = บอทของ GitHub ที่ตรวจ dependency อัตโนมัติ

```
นึกภาพแบบนี้:

Dependabot = พนักงานตรวจสต๊อก

ทุกสัปดาห์จะเข้ามาเช็ค:
  "library ตัวไหนมี version ใหม่บ้าง?"
  "Docker image ตัวไหนเก่าแล้ว?"

ถ้าเจอ → เขียนใบเสนอ (Pull Request) ให้อนุมัติ
```

**สิ่งที่ Dependabot ทำให้:**
- ✅ ตรวจ dependency ทุกสัปดาห์ → ไม่ต้องเช็คเอง
- ✅ สร้าง PR อัตโนมัติ → พร้อม merge เลย
- ✅ CI รัน test ให้ทันที → รู้เลยว่า version ใหม่พังไหม
- ✅ แจ้งเตือนช่องโหว่ (CVE) → ปลอดภัยขึ้น

**สิ่งที่ Dependabot ทำไม่ได้:**
- ❌ merge เองไม่ได้ → ต้องมีคน approve + กด merge
- ❌ แก้โค้ดให้ไม่ได้ → ถ้า update พังต้องแก้เอง

---

## 2. ทำไมต้องมี

### ถ้าไม่มี Dependabot

| ปัญหา | ผลกระทบ |
|--------|---------|
| Library มีช่องโหว่ แต่ไม่รู้ | โดน hack / ข้อมูลรั่ว |
| ไม่อัปเดตนาน แล้วอัปทีเดียว | แก้ breaking changes ยากมาก |
| ต้องเข้า GitHub / changelog เช็คเอง | เสียเวลา ลืมง่าย |
| Docker image เก่า | มี CVE สะสม |

### ถ้ามี Dependabot

| ข้อดี | รายละเอียด |
|-------|-----------|
| **อัตโนมัติ** | ตรวจทุกสัปดาห์ ไม่ต้องเช็คเอง |
| **ทีละนิด** | PR ละ 1 dependency → ถ้าพังก็รู้ว่าตัวไหน |
| **ปลอดภัย** | แจ้ง CVE + สร้าง PR แก้ช่องโหว่ทันที |
| **ฟรี** | เป็น feature ของ GitHub ไม่มีค่าใช้จ่าย |

---

## 3. Config ของโปรเจกต์นี้

Dependabot ตรวจ **3 สิ่ง** ในโปรเจกต์นี้:

| # | ตรวจอะไร | ไฟล์ที่ดู | ตัวอย่าง PR |
|---|---------|----------|------------|
| 1 | **Go modules** | `go.mod` | bump zerolog from 1.34.0 to 1.35.0 |
| 2 | **GitHub Actions** | `.github/workflows/*.yml` | bump actions/checkout from v4 to v5 |
| 3 | **Docker images** | `Dockerfile`, `Dockerfile.worker` | bump golang from 1.25-alpine to 1.26-alpine |

### เมื่อไหร่

```
ทุกวันจันทร์ 09:00 น. (เวลาไทย)
```

### จำกัด PR

| ประเภท | เปิดพร้อมกันได้สูงสุด |
|--------|---------------------|
| Go modules | 5 PRs |
| GitHub Actions | 5 PRs |
| Docker images | 3 PRs |

### Target Branch

```
PR ทั้งหมดชี้ไปที่ branch: dev
```

---

## 4. Flow การทำงาน

```
ทุกวันจันทร์ 09:00 น.
        │
        ▼
┌───────────────────────────────┐
│    Dependabot scan            │
│                               │
│    📦 go.mod    → มี update?  │
│    ⚙️ workflows → มี update?  │
│    🐳 Dockerfile → มี update? │
└───────────┬───────────────────┘
            │
       มี version ใหม่?
            │
    ┌───────┴───────┐
    │               │
  ไม่มี           มี!
  (จบ)             │
                   ▼
          ┌─────────────────┐
          │  สร้าง PR       │
          │  - title ชัดเจน │
          │  - label ติดให้  │
          │  - changelog    │
          └────────┬────────┘
                   │
                   ▼
          ┌─────────────────┐
          │  CI รันอัตโนมัติ │
          │  Lint ✅         │
          │  Test ✅         │
          │  Build ✅        │
          │  Vuln Check ✅   │
          └────────┬────────┘
                   │
              CI ผ่าน?
                   │
           ┌───────┴───────┐
           │               │
          ผ่าน           ไม่ผ่าน
           │               │
           ▼               ▼
     รอคุณ Review    ดู error log
     + Approve       → แก้ / ปิด PR
     + Merge
           │
           ▼
     merge เข้า dev ✅
```

---

## 5. เมื่อ Dependabot สร้าง PR — ทำอะไร?

### 5 ขั้นตอน ง่าย ๆ

```
ขั้นที่ 1:  เปิดดู PR
ขั้นที่ 2:  ดู CI → ผ่าน ✅ ?
ขั้นที่ 3:  ดูไฟล์ที่เปลี่ยน (ถ้าอยากรู้)
ขั้นที่ 4:  กด Approve
ขั้นที่ 5:  กด Merge
```

---

### ขั้นที่ 1: เปิดดู PR

**ใน VS Code:**

```
sidebar ซ้าย → ไอคอน GitHub Pull Requests
  └── All Open
      └── deps(deps): bump zerolog from 1.34.0 to 1.35.0  ← คลิกที่นี่
```

**บน GitHub.com:**

```
เปิด repository → แท็บ Pull requests → จะเห็น PR มี label "dependencies"
```

---

### ขั้นที่ 2: ดู CI Checks

เลื่อนลงในหน้า PR → ดูส่วน Checks:

| สถานะ | ไอคอน | ความหมาย | ทำอะไรต่อ |
|-------|-------|---------|----------|
| ✅ Passed | เขียว | ผ่าน | ไป Approve ได้เลย |
| ❌ Failed | แดง | พัง | [ดูหัวข้อ CI Failed](#7-กรณี-ci-failed) |
| 🟡 Pending | เหลือง | กำลังรัน | รอ |
| ⏭ Skipped | เทา | ข้าม (ปกติ) | ไม่ต้องสนใจ |

**ตัวอย่างที่เห็น:**

```
✅ Lint          ผ่าน
✅ Test          ผ่าน
✅ Vuln Check    ผ่าน
✅ Build         ผ่าน
⏭ Docker        ข้าม (ปกติสำหรับ PR)
⏭ Image Scan    ข้าม
✅ Notify        ผ่าน

→ ทุกอย่างผ่าน ✅ → ไป Approve ได้เลย
```

---

### ขั้นที่ 3: ดู Files Changed (ไม่จำเป็น แต่แนะนำ)

**ใน VS Code:** คลิก expand PR → คลิกที่ไฟล์ → เห็น diff (เทียบก่อน/หลัง)

| ประเภท PR | ไฟล์ที่แก้ | สิ่งที่เห็น |
|-----------|-----------|-----------|
| Go module | `go.mod`, `go.sum` | version เปลี่ยน |
| Docker | `Dockerfile` | base image version เปลี่ยน |
| Actions | `*.yml` | action version เปลี่ยน |

**ตัวอย่าง diff ของ go.mod:**

```diff
- github.com/rs/zerolog v1.34.0
+ github.com/rs/zerolog v1.35.0
```

> แค่เลข version เปลี่ยน → ปลอดภัย

---

### ขั้นที่ 4: Approve

**ใน VS Code:**

```
เปิดหน้า PR → เลื่อนลง → ใต้กล่อง comment
→ เลือก "Approve" จาก dropdown
→ กด Submit
```

**บน GitHub.com:**

```
เปิด PR → แท็บ "Files changed"
→ กดปุ่มเขียว "Review changes" (มุมขวาบน)
→ เลือก "Approve"
→ กด "Submit review"
```

**3 ตัวเลือกที่เห็น:**

| ปุ่ม | ใช้เมื่อ |
|------|---------|
| **Comment** | อยากบอกอะไร แต่ไม่ตัดสินใจ |
| **Approve** ✅ | โค้ด OK → กดอันนี้ |
| **Request Changes** ❌ | ต้องแก้ก่อน (ไม่ค่อยใช้กับ Dependabot) |

---

### ขั้นที่ 5: Merge

**ใน VS Code:**

```
เปิดหน้า PR → เลื่อนลงล่างสุด
→ กดปุ่ม "Merge Pull Request"
→ (หรือกด ▼ เลือก "Squash and Merge")
→ กด Confirm
```

**บน GitHub.com:**

```
เลื่อนลงล่างสุด → กด "Squash and merge" → กด Confirm
```

**3 วิธี Merge:**

| วิธี | ผลลัพธ์ | แนะนำ |
|------|---------|-------|
| Merge commit | รวมทุก commit | |
| **Squash and merge** | รวมเป็น 1 commit | ✅ แนะนำ |
| Rebase and merge | วาง commit ต่อท้าย | |

---

### หลัง Merge เสร็จ

```powershell
# ดึงโค้ดใหม่มาเครื่อง
git checkout dev
git pull

# (ถ้าอยู่ branch อื่น)
git checkout dev
```

> Branch ของ Dependabot จะถูกลบอัตโนมัติหลัง merge

---

## 6. Checkout ทดสอบบนเครื่อง

> **ไม่จำเป็นเสมอ** — ใช้เมื่อไม่มั่นใจ หรือเป็น major version bump

### เมื่อไหร่ควร Checkout

| สถานการณ์ | ควร Checkout? |
|-----------|-------------|
| Patch bump (1.34.0 → 1.34.1) | ❌ ไม่จำเป็น |
| Minor bump (1.34.0 → 1.35.0) | ⚠️ ถ้าไม่มั่นใจ |
| Major bump (v1 → v2) | ✅ ควรทำ |
| Docker image bump | ⚠️ ถ้า Go version เปลี่ยน |
| CI ผ่านทุกอย่าง | ❌ ไม่จำเป็น |

### วิธี Checkout

**ใน VS Code:**

```
sidebar ซ้าย → GitHub Pull Requests → คลิก PR → กดปุ่ม Checkout (ลูกศร →)
→ VS Code สลับ branch ให้อัตโนมัติ
```

**ใน Terminal:**

```powershell
# ดึง branch ล่าสุด
git fetch

# สลับไป branch ของ PR
git checkout dependabot/go_modules/github.com/rs/zerolog-1.35.0

# ทดสอบ
go build ./...       # build ผ่านไหม
go test ./...        # test ผ่านไหม
.\run.ps1 dev        # รันจริงลองดู (optional)

# กลับ dev เมื่อเสร็จ
git checkout dev
```

---

## 7. กรณี CI Failed

### ดูว่าพังตรงไหน

```
ใน PR → กด "Details" / "Show" ข้าง check ที่ fail
→ อ่าน error log
```

### แก้ตามกรณี

```
CI Failed
    │
    ├── Test fail?
    │    → version ใหม่มี breaking change
    │    → checkout มาแก้โค้ดให้ compatible → push
    │
    ├── Lint fail?
    │    → API เปลี่ยน format
    │    → ส่วนใหญ่แก้ง่าย
    │
    ├── Build fail?
    │    → version ใหม่ไม่ compatible
    │    → อาจต้องรอ fix หรือปิด PR
    │
    └── ไม่แน่ใจ?
         → comment ใน PR ถามทีม
         → หรือปิด PR → Dependabot สร้างใหม่สัปดาห์หน้า
```

### ปิด PR (ถ้าไม่ต้องการ)

- VS Code: เปิด PR → กด **Close**
- GitHub: กด **Close pull request**
- Comment: `@dependabot close`

---

## 8. PR หลายตัว — Merge ยังไง

### กฎง่าย ๆ

```
✅ PR ที่แก้คนละไฟล์ → merge ลำดับไหนก็ได้
⚠️ PR ที่แก้ไฟล์เดียวกัน → merge ทีละตัว
```

### ตัวอย่าง

```
PR #1: bump zerolog      (แก้ go.mod)      ← merge ก่อน
PR #2: bump validator    (แก้ go.mod)      ← merge ทีหลัง (รอ rebase)
PR #3: bump golang image (แก้ Dockerfile)  ← merge เมื่อไหร่ก็ได้
PR #4: bump alpine image (แก้ Dockerfile)  ← merge หลัง #3

ลำดับแนะนำ: #1 → #3 → #2 → #4
(สลับไฟล์ที่แก้ ไม่ต้องรอ rebase)
```

### ถ้าเกิด Conflict

Dependabot จะ **rebase ให้อัตโนมัติ** หลัง PR อื่น merge

ถ้าไม่ rebase → comment ใน PR: `@dependabot rebase`

---

## 9. Dependabot Commands

comment ใน PR เพื่อสั่ง Dependabot:

| Command | ทำอะไร |
|---------|--------|
| `@dependabot rebase` | Rebase branch ให้ up-to-date |
| `@dependabot recreate` | ปิด PR แล้วสร้างใหม่ |
| `@dependabot merge` | Merge (ถ้า CI ผ่าน + approved) |
| `@dependabot squash and merge` | Squash แล้ว merge |
| `@dependabot cancel merge` | ยกเลิก auto-merge |
| `@dependabot close` | ปิด PR |
| `@dependabot ignore this dependency` | ไม่ track dependency นี้อีก |
| `@dependabot ignore this major version` | ข้าม major version นี้ |

### ตัวอย่างการใช้

```
สถานการณ์: PR #3 conflict กับ PR #1 ที่เพิ่ง merge

คุณ: comment ใน PR #3 ว่า "@dependabot rebase"
Bot: rebase branch ให้ → conflict หายไป → CI รันใหม่
```

---

## 10. Config อธิบายทีละบรรทัด

```yaml
version: 2                          # Dependabot config version

updates:
  - package-ecosystem: gomod        # ตรวจ Go modules (go.mod)
    directory: /                    # อยู่ root ของโปรเจกต์
    target-branch: dev              # สร้าง PR ชี้ไปที่ branch dev
    schedule:
      interval: weekly              # ตรวจทุกสัปดาห์
      day: monday                   # วันจันทร์
      time: "09:00"                 # 09:00 น.
      timezone: Asia/Bangkok        # เวลาไทย
    open-pull-requests-limit: 5     # เปิด PR พร้อมกันได้สูงสุด 5
    labels:                         # ติด label อัตโนมัติ
      - dependencies
      - go
    commit-message:                 # รูปแบบ commit message
      prefix: "deps"               # deps(scope): bump xxx
      include: scope
```

### ค่าที่ปรับได้

| ค่า | ตัวเลือก | แนะนำ |
|-----|---------|-------|
| `interval` | `daily` / `weekly` / `monthly` | `weekly` ✅ |
| `day` | `monday` - `sunday` | `monday` |
| `open-pull-requests-limit` | 1-99 | 3-5 |
| `target-branch` | ชื่อ branch | `dev` |

---

## 11. Cheatsheet

### เมื่อเห็น Dependabot PR → ทำ 3 สิ่ง

```
┌──────────────────────────────────────────┐
│                                          │
│   1. ดู CI  →  ผ่าน ✅ ?                  │
│   2. กด Approve                          │
│   3. กด Merge (Squash and merge)         │
│                                          │
│   เสร็จ! 🎉                              │
│                                          │
└──────────────────────────────────────────┘
```

### กรณีพิเศษ

```
CI fail?      → ดู log → แก้โค้ด หรือ ปิด PR
หลาย PR?      → merge ทีละตัว (สลับไฟล์ที่แก้)
Conflict?     → comment: @dependabot rebase
ไม่ต้องการ?   → comment: @dependabot close
Major bump?   → checkout มาทดสอบก่อน
```

### หลัง Merge

```powershell
git checkout dev
git pull
```

### Flow ปัจจุบันของโปรเจกต์

```
Dependabot ──── PR ──── merge เข้า dev ──── (review) ──── merge เข้า main
                                  ▲
                                  │
                          คุณอยู่ตรงนี้
```
