# Dependabot PR — คู่มือการจัดการ

> คู่มือปฏิบัติ — เมื่อ Dependabot สร้าง PR มาแล้ว ต้องทำอะไร?
>
> อ่าน concept ก่อน → [dependabot-concept.md](dependabot-concept.md)

---

## สารบัญ

- [ภาพรวม](#ภาพรวม)
- [ขั้นตอนที่ 1: เปิดดู PR](#ขั้นตอนที่-1-เปิดดู-pr)
- [ขั้นตอนที่ 2: ดู CI Checks](#ขั้นตอนที่-2-ดู-ci-checks)
- [ขั้นตอนที่ 3: ดูไฟล์ที่เปลี่ยน](#ขั้นตอนที่-3-ดูไฟล์ที่เปลี่ยน)
- [ขั้นตอนที่ 4: Checkout ทดสอบ (ถ้าต้องการ)](#ขั้นตอนที่-4-checkout-ทดสอบ-ถ้าต้องการ)
- [ขั้นตอนที่ 5: Approve](#ขั้นตอนที่-5-approve)
- [ขั้นตอนที่ 6: Merge](#ขั้นตอนที่-6-merge)
- [กรณี CI Failed](#กรณี-ci-failed)
- [PR หลายตัว — Merge ลำดับไหน](#pr-หลายตัว--merge-ลำดับไหน)
- [Dependabot Commands](#dependabot-commands)
- [สรุป Cheatsheet](#สรุป-cheatsheet)

---

## ภาพรวม

เมื่อ Dependabot สร้าง PR มาแล้ว หน้าที่ของเราคือ:

```
Dependabot สร้าง PR ─────────────────────────────── อัตโนมัติ ✅
CI Checks รัน (lint, test, build) ───────────────── อัตโนมัติ ✅
ดู PR + ตรวจ CI result ──────────────────────────── ทำเอง 👈
Approve ─────────────────────────────────────────── ทำเอง 👈
Merge ───────────────────────────────────────────── ทำเอง 👈
```

---

## ขั้นตอนที่ 1: เปิดดู PR

### ใน VS Code

1. sidebar ซ้าย → ไอคอน **GitHub Pull Requests**
2. ดูหมวด **All Open** → จะเห็น PR ของ Dependabot มีไอคอน 🤖
3. คลิกที่ PR ที่ต้องการ → รายละเอียดเปิดทางขวา

### บน GitHub.com

1. เปิด repository → แท็บ **Pull requests**
2. จะเห็น PR ที่มี label `dependencies`

### อ่านอะไรบ้าง

| ส่วน | ดูอะไร |
|------|--------|
| **Title** | อัพเดตอะไร เช่น "bump zerolog from 1.34.0 to 1.35.0" |
| **Description** | Release notes, changelog, compatibility score |
| **Labels** | `dependencies` + `go` / `docker` / `ci` |
| **Commits** | ปกติมี 1 commit จาก Dependabot |

---

## ขั้นตอนที่ 2: ดู CI Checks

เลื่อนลงในหน้า PR → จะเห็นส่วน **Checks**

```
✅ Lint          ผ่าน
✅ Test          ผ่าน
✅ Vuln Check    ผ่าน
✅ Build         ผ่าน
⏭ Docker        ข้าม (ปกติสำหรับ PR)
⏭ Image Scan    ข้าม
✅ Notify        ผ่าน
```

### ตัดสินใจ

| CI ผลลัพธ์ | ทำอะไรต่อ |
|------------|----------|
| ✅ ผ่านทั้งหมด | → ไปขั้นตอน 3 (ดูไฟล์) หรือข้ามไป Approve เลย |
| ❌ มี Failed | → [ดูหัวข้อ CI Failed](#กรณี-ci-failed) |
| 🟡 Pending | → รอให้ CI รันเสร็จ |

---

## ขั้นตอนที่ 3: ดูไฟล์ที่เปลี่ยน

### ใน VS Code

คลิก expand PR → จะเห็นไฟล์ที่แก้ พร้อมสถานะ:

```
> deps(deps): bump zerolog from 1.34.0 to 1.35.0
    ✅ go.mod     M (Modified)
    ✅ go.sum     M (Modified)
```

คลิกที่ไฟล์ → จะเปิด **diff view** (เทียบก่อน/หลัง)

### ดูอะไร

| ประเภท PR | ไฟล์ที่แก้ | สิ่งที่ต้องดู |
|-----------|-----------|-------------|
| **Go module** | `go.mod`, `go.sum` | version เปลี่ยนจากอะไรเป็นอะไร |
| **Docker image** | `Dockerfile`, `Dockerfile.worker` | base image เปลี่ยนจากอะไร |
| **GitHub Actions** | `.github/workflows/*.yml` | action version เปลี่ยน |

> **สำหรับ Go module + Docker:** ส่วนใหญ่แค่ดู CI ผ่านก็พอ ไม่ต้องดูไฟล์ละเอียดมาก

---

## ขั้นตอนที่ 4: Checkout ทดสอบ (ถ้าต้องการ)

> ⚡ **ขั้นตอนนี้ไม่จำเป็นเสมอ** — ถ้า CI ผ่านและเป็นแค่ patch/minor ข้ามไป Approve ได้เลย

### ควร Checkout เมื่อ

- Major version bump (เช่น v1 → v2)
- Dependency สำคัญ ๆ (เช่น Go version, Fiber, pgx)
- CI ผ่านแต่ไม่มั่นใจ

### วิธี Checkout

**ใน VS Code:** กดปุ่ม **Checkout** (ลูกศร →) ข้าง PR

**ใน Terminal:**
```powershell
git fetch
git checkout dependabot/go_modules/github.com/rs/zerolog-1.35.0
```

### ทดสอบ

```powershell
# build ได้ไหม
go build ./...

# test ผ่านไหม
go test ./...

# (ถ้าอยากลอง) รันจริง
.\run.ps1 dev
```

### กลับ main

```powershell
git checkout main
```

---

## ขั้นตอนที่ 5: Approve

### ใน VS Code

1. เปิดหน้า PR → เลื่อนลง
2. ใต้กล่อง comment → เลือก **Approve** จาก dropdown
3. พิมพ์ comment (ไม่จำเป็น) → กด **Submit**

### บน GitHub.com

1. เปิด PR → แท็บ **Files changed**
2. กดปุ่มเขียว **Review changes** (มุมขวาบน)
3. เลือก **Approve** → กด **Submit review**

---

## ขั้นตอนที่ 6: Merge

### ใน VS Code

1. เปิดหน้า PR → เลื่อนลงล่างสุด
2. กดปุ่ม **Merge Pull Request** (หรือ ▼ เลือก Squash and Merge)
3. กด Confirm

### บน GitHub.com

1. เลื่อนลงล่างสุด → กด **Squash and merge**
2. แก้ commit message (ถ้าต้องการ) → กด **Confirm**

### หลัง Merge

```powershell
# ดึงโค้ดใหม่ที่ merge แล้วมาเครื่อง
git checkout main
git pull
```

> Branch ของ Dependabot จะถูกลบอัตโนมัติหลัง merge

---

## กรณี CI Failed

### ขั้นตอนจัดการ

```
CI Failed
   │
   ├── กด "Details" ดู log ว่าพังตรงไหน
   │
   ├── พัง Test?
   │    → version ใหม่มี breaking change
   │    → ต้อง checkout มาแก้โค้ดให้ compatible
   │
   ├── พัง Lint?
   │    → API เปลี่ยน format
   │    → ส่วนใหญ่แก้ง่าย
   │
   ├── พัง Build?
   │    → version ใหม่ไม่ compatible กับ Go version
   │    → อาจต้องรอ fix หรือปิด PR
   │
   └── ไม่แน่ใจ?
        → Comment ใน PR ถามทีม
        → หรือปิด PR (Dependabot จะสร้างใหม่สัปดาห์ถัดไป)
```

### วิธีปิด PR (ถ้าไม่ต้องการ)

- ใน VS Code: เปิด PR → กด **Close** ด้านล่าง
- บน GitHub: กด **Close pull request**

> Dependabot จะสร้าง PR ใหม่ในรอบถัดไป ถ้า version ยังใหม่กว่า

---

## PR หลายตัว — Merge ลำดับไหน

เมื่อ Dependabot สร้าง PR มาหลายตัวพร้อมกัน:

### กฎง่าย ๆ

```
1. PR ที่แก้คนละไฟล์ → merge ลำดับไหนก็ได้
2. PR ที่แก้ไฟล์เดียวกัน → merge ทีละตัว (รอ rebase)
```

### ตัวอย่าง

```
PR #1: bump zerolog (แก้ go.mod)        ← merge ก่อน
PR #2: bump validator (แก้ go.mod)      ← merge ทีหลัง (อาจต้อง rebase)
PR #3: bump golang image (แก้ Dockerfile) ← merge เมื่อไหร่ก็ได้
PR #4: bump alpine image (แก้ Dockerfile) ← merge หลัง #3
```

### ถ้าเกิด Conflict

Dependabot จะ **rebase ให้อัตโนมัติ** หลังจาก PR อื่น merge แล้ว

ถ้าไม่ rebase เอง → comment ใน PR ว่า `@dependabot rebase`

---

## Dependabot Commands

สามารถ comment ใน PR เพื่อสั่ง Dependabot ได้:

| Command | ทำอะไร |
|---------|--------|
| `@dependabot rebase` | Rebase branch ให้ up-to-date กับ main |
| `@dependabot recreate` | ปิด PR แล้วสร้างใหม่ |
| `@dependabot merge` | Merge PR (ถ้า CI ผ่าน + approved) |
| `@dependabot squash and merge` | Squash แล้ว merge |
| `@dependabot cancel merge` | ยกเลิก auto-merge |
| `@dependabot close` | ปิด PR (ไม่สร้างใหม่จนกว่าจะมี version ใหม่กว่า) |
| `@dependabot ignore this dependency` | ไม่ track dependency นี้อีก |
| `@dependabot ignore this major version` | ข้าม major version นี้ |

---

## สรุป Cheatsheet

```
┌────────────────────────────────────────────────────┐
│           Dependabot PR — ทำ 3 สิ่ง                │
│                                                    │
│  1. ดู CI  →  ผ่าน ✅ ?                             │
│  2. Approve                                        │
│  3. Merge (Squash and merge)                       │
│                                                    │
│  CI fail?  → ดู log → แก้ หรือ ปิด PR              │
│  หลาย PR?  → merge ทีละตัว                         │
│  Conflict? → comment: @dependabot rebase           │
│  กลับ main → git checkout main && git pull         │
└────────────────────────────────────────────────────┘
```
