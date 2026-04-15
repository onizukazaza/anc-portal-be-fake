# Pull Request (PR) Guide — คู่มือการใช้งาน

> **v1.0** — วันที่: 1 เมษายน 2026
>
> คู่มือสำหรับทีมพัฒนา — อธิบายการ Review, Approve, และ Merge PR ทีละขั้นตอน

---

## สารบัญ

- [Pull Request คืออะไร](#pull-request-คืออะไร)
- [Flow ภาพรวม](#flow-ภาพรวม)
- [ขั้นตอนที่ 1: สร้าง Branch ใหม่](#ขั้นตอนที่-1-สร้าง-branch-ใหม่)
- [ขั้นตอนที่ 2: เขียนโค้ดแล้ว Push](#ขั้นตอนที่-2-เขียนโค้ดแล้ว-push)
- [ขั้นตอนที่ 3: สร้าง Pull Request](#ขั้นตอนที่-3-สร้าง-pull-request)
- [ขั้นตอนที่ 4: CI Checks ทำงานอัตโนมัติ](#ขั้นตอนที่-4-ci-checks-ทำงานอัตโนมัติ)
- [ขั้นตอนที่ 5: Code Review](#ขั้นตอนที่-5-code-review)
- [ขั้นตอนที่ 6: Merge PR](#ขั้นตอนที่-6-merge-pr)
- [อธิบายปุ่มแต่ละตัว](#อธิบายปุ่มแต่ละตัว)
- [Checkout คืออะไร](#checkout-คืออะไร)
- [สรุปคำศัพท์](#สรุปคำศัพท์)
- [Tips & Best Practices](#tips--best-practices)

---

## Pull Request คืออะไร

Pull Request (PR) คือ **การขออนุญาตรวมโค้ดของเราเข้า branch หลัก**

เปรียบเทียบง่าย ๆ:

```
PR = "เฮ้ทีม ผมเขียนโค้ดเสร็จแล้ว ช่วยดูหน่อยว่า OK ไหม ถ้า OK ก็รวมเข้า main ได้เลย"
```

ทำไมต้องมี PR:
- ✅ มีคนอื่นช่วยตรวจโค้ดก่อน merge (ลด bug)
- ✅ CI รัน test อัตโนมัติ (ลดโอกาส build พัง)
- ✅ มีประวัติทุกการเปลี่ยนแปลง (ย้อนดูได้)

---

## Flow ภาพรวม

```
1. สร้าง Branch          คุณ: git checkout -b feature/xxx
       ↓
2. เขียนโค้ด + Push      คุณ: git push
       ↓
3. สร้าง PR              คุณ: กดสร้าง PR บน GitHub / VS Code
       ↓
4. CI Checks             อัตโนมัติ: lint, test, build
       ↓
5. Code Review           ทีม: ดูโค้ด → Comment / Approve / Request Changes
       ↓
6. Merge                 คุณ: กด Merge เมื่อ Approved + Checks ผ่าน
       ↓
7. Branch ถูกลบ          อัตโนมัติ (หรือลบเอง)
```

---

## ขั้นตอนที่ 1: สร้าง Branch ใหม่

**ห้าม** เขียนโค้ดตรงบน `main` → ต้องสร้าง branch แยกเสมอ

### ใน Terminal

```powershell
# ดึงโค้ดล่าสุดจาก main ก่อน
git checkout main
git pull

# สร้าง branch ใหม่
git checkout -b feature/add-login-api
```

### ใน VS Code

1. กดมุมล่างซ้าย (ชื่อ branch) → **Create new branch**
2. ตั้งชื่อ เช่น `feature/add-login-api`

### หลักการตั้งชื่อ Branch

| ประเภท | รูปแบบ | ตัวอย่าง |
|--------|--------|---------|
| Feature ใหม่ | `feature/ชื่อ-feature` | `feature/add-login-api` |
| แก้ Bug | `fix/ชื่อ-bug` | `fix/healthz-nil-panic` |
| Refactor | `refactor/ชื่อ` | `refactor/database-layer` |
| Docs | `docs/ชื่อ` | `docs/api-guide` |

---

## ขั้นตอนที่ 2: เขียนโค้ดแล้ว Push

```powershell
# เขียนโค้ดเสร็จแล้ว...

# ดูว่าแก้ไขไฟล์อะไรบ้าง
git status

# เพิ่มไฟล์ที่ต้องการ
git add .

# Commit (ข้อความอธิบายสั้น ๆ)
git commit -m "feat: add login API endpoint"

# Push ขึ้น GitHub
git push -u origin feature/add-login-api
```

### Commit Message Format

```
<type>: <คำอธิบายสั้น ๆ>

ตัวอย่าง:
feat: add login API endpoint
fix: resolve healthz nil pointer panic
docs: update API guide
refactor: simplify database layer
```

---

## ขั้นตอนที่ 3: สร้าง Pull Request

### วิธีที่ 1: บน GitHub.com

1. เปิด repository บน GitHub
2. จะเห็นแถบสีเหลือง **"Compare & pull request"** → กด
3. กรอกข้อมูล:
   - **Title**: อธิบายสั้น ๆ ว่าทำอะไร
   - **Description**: รายละเอียดเพิ่มเติม
   - **Reviewers**: เลือกคนที่ต้องการให้ review
4. กด **Create pull request**

### วิธีที่ 2: ใน VS Code

1. เมนูซ้าย → ไอคอน **GitHub Pull Requests**
2. กด **Create Pull Request**
3. กรอก Title + Description
4. กด **Create**

---

## ขั้นตอนที่ 4: CI Checks ทำงานอัตโนมัติ

เมื่อสร้าง PR แล้ว GitHub Actions จะ **รันอัตโนมัติ**:

```
✅ Lint        → ตรวจ code style
✅ Test        → รัน unit tests
✅ Vuln Check  → ตรวจ security vulnerabilities
✅ Build       → ทดสอบว่า build ผ่าน
```

### อ่านผลลัพธ์

| สถานะ | ไอคอน | ความหมาย |
|-------|-------|---------|
| ✅ Passed | เขียว | ผ่านทั้งหมด → พร้อม review |
| ❌ Failed | แดง | มีอะไรพัง → ต้องแก้ก่อน |
| 🟡 Pending | เหลือง | กำลังรันอยู่ → รอ |
| ⏭ Skipped | เทา | ข้ามไป (ไม่เกี่ยวกับ PR นี้) |

### ถ้า Check Failed

1. กด **Show** หรือ **Details** ดู log
2. อ่าน error message
3. แก้โค้ด → commit → push → CI จะรันใหม่อัตโนมัติ

---

## ขั้นตอนที่ 5: Code Review

Reviewer เปิด PR แล้วทำอย่างใดอย่างหนึ่ง:

### 5 สิ่งที่ Reviewer ทำได้

| ปุ่ม | ใช้เมื่อ | ผลลัพธ์ |
|------|---------|---------|
| **Comment** | อยากบอกอะไรแต่ยังไม่ตัดสินใจ | แค่ comment ไม่ approve/reject |
| **Approve** ✅ | โค้ด OK พร้อม merge | PR ได้ status "Approved" |
| **Request Changes** ❌ | ต้องแก้ก่อน merge | PR ถูก block จนกว่าจะแก้ |

### Comment แบบต่าง ๆ

```
1. Comment ทั่วไป (General)
   → พิมพ์ใน "Leave a comment" แล้วกด Comment
   → ใช้เมื่ออยากถามหรือแนะนำภาพรวม

2. Comment บนโค้ด (Inline)
   → กดที่บรรทัดโค้ดใน "Files changed"
   → ใช้เมื่ออยากชี้จุดเฉพาะในโค้ด

3. Suggestion
   → Comment บนโค้ด + กดปุ่ม "Suggest"
   → เสนอโค้ดแก้ไข ที่เจ้าของ PR กด Accept ได้เลย
```

### ตัวอย่างการ Review

```
สมมติ PR เพิ่ม Login API:

Reviewer ดูโค้ดแล้วเห็นว่า:
  1. Logic ถูกต้อง → ✅
  2. มี typo ตัวแปร → Comment บนบรรทัดนั้น
  3. ขาด error handling → Request Changes

เจ้าของ PR แก้ตาม comment → push ใหม่
Reviewer ดูอีกรอบ → Approve ✅
→ พร้อม Merge
```

---

## ขั้นตอนที่ 6: Merge PR

เมื่อ **Checks ผ่าน** + **Approved** → กด Merge ได้

### ประเภท Merge

| วิธี | ผลลัพธ์ | ใช้เมื่อ |
|------|---------|---------|
| **Merge commit** | รวมทุก commit + สร้าง merge commit | default ทั่วไป |
| **Squash and merge** | รวมทุก commit เป็น 1 commit | PR มีหลาย commit เล็ก ๆ |
| **Rebase and merge** | เอา commits ไปต่อบน main | ต้องการ history เป็นเส้นตรง |

> **แนะนำ:** ใช้ **Squash and merge** สำหรับ feature PR → history สะอาด

---

## อธิบายปุ่มแต่ละตัว

### ปุ่ม Approve ✅

```
"โค้ดนี้ OK แล้ว เอาไป merge ได้เลย"
```
- กดแล้ว PR ได้ status **Approved**
- ถ้ามีกฎว่าต้อง approve ≥ 1 คน → ต้องมีคน approve ก่อน merge ได้

### ปุ่ม Comment 💬

```
"มีอะไรอยากบอก แต่ยังไม่ approve/reject"
```
- แค่ส่ง comment
- ไม่ block PR, ไม่ approve

### ปุ่ม Request Changes ❌

```
"ต้องแก้ก่อนนะ ยังไม่ให้ merge"
```
- PR ถูก **block** จนกว่า reviewer คนเดิมจะเปลี่ยนเป็น Approve
- เจ้าของ PR ต้องแก้โค้ดตาม feedback แล้ว push ใหม่

### Checks (CI) ✅❌

```
"ระบบตรวจอัตโนมัติ (lint, test, build)"
```
- ไม่ใช่คน → เป็น GitHub Actions ที่รันเอง
- ถ้า Failed → ต้องแก้โค้ดให้ผ่านก่อน merge

---

## Checkout คืออะไร

**Checkout = สลับไปยัง branch อื่น**

### ใช้ทำอะไร

```
สมมติทีมสร้าง PR → คุณอยากลอง/ทดสอบโค้ดของเขาบนเครื่องคุณ
→ ใช้ Checkout ดึง branch ของเขามารันดู
```

### วิธี Checkout

#### ใน Terminal

```powershell
# ดึง branch ล่าสุดจาก GitHub
git fetch

# สลับไปยัง branch ของ PR
git checkout feature/add-login-api

# ทดสอบ
.\run.ps1 dev

# กลับไป main เมื่อเสร็จ
git checkout main
```

#### ใน VS Code (GitHub PR Extension)

1. เปิด PR ใน sidebar ซ้าย (GitHub Pull Requests)
2. กดปุ่ม **Checkout** (ไอคอนลูกศร →)
3. VS Code จะสลับ branch ให้อัตโนมัติ
4. ทดสอบโค้ดได้เลย

### Checkout กับ PR เกี่ยวกันยังไง

```
PR #2: "bump golang from 1.25 to 1.26"
   → Branch: dependabot/docker/golang-1.26
   → คุณอยากลองว่า Go 1.26 build ผ่านไหม?

ขั้นตอน:
1. กด Checkout → สลับไป branch ของ PR
2. รัน: .\run.ps1 ci    → ดูว่า lint/test/build ผ่าน
3. OK → กลับ main → กด Approve + Merge บน PR
```

---

## สรุปคำศัพท์

| คำ | ภาษาง่าย ๆ |
|----|-----------|
| **Pull Request (PR)** | ขออนุญาตรวมโค้ดเข้า branch หลัก |
| **Branch** | สำเนาของโค้ดที่แยกออกมาแก้ไข |
| **Checkout** | สลับไป branch อื่น |
| **Commit** | บันทึกการเปลี่ยนแปลงโค้ด |
| **Push** | ส่ง commit ขึ้น GitHub |
| **Merge** | รวม branch เข้า main |
| **Approve** | อนุมัติ PR (โค้ด OK) |
| **Request Changes** | ขอให้แก้ก่อน (ยัง merge ไม่ได้) |
| **Comment** | แสดงความเห็น (ไม่ approve/reject) |
| **CI Checks** | ระบบตรวจอัตโนมัติ (lint, test, build) |
| **Reviewer** | คนที่ review โค้ดของเรา |
| **Squash** | รวมหลาย commit เป็น 1 |
| **Dependabot** | Bot ที่สร้าง PR อัพเดท dependency อัตโนมัติ |

---

## Tips & Best Practices

### สำหรับคนสร้าง PR
1. **PR เล็ก ๆ** → ง่ายต่อการ review (ไม่เกิน 400 บรรทัด)
2. **เขียน description** → บอกว่าทำอะไร ทำไม
3. **CI ต้องผ่านก่อน** → อย่าขอ review ถ้า checks ยัง fail
4. **ตอบ comment** → แก้ตามหรืออธิบายเหตุผล

### สำหรับ Reviewer
1. **ดูภาพรวมก่อน** → อ่าน description แล้วค่อยดูโค้ด
2. **Comment สร้างสรรค์** → บอกว่า "ควรแก้ยังไง" ไม่ใช่แค่ "ผิด"
3. **อย่า block นาน** → review ภายใน 1 วัน
4. **Approve เมื่อ OK** → อย่าลืมกด Approve (ไม่ใช่แค่ comment)

### สำหรับ Dependabot PRs (ที่เห็นในหน้าจอ)

```
PR เหล่านี้สร้างโดย bot อัตโนมัติ:
  - deps(deps): bump golang from 1.25 to 1.26
  - deps(deps): bump alpine from 3.21 to 3.23

วิธีจัดการ:
  1. ดู CI Checks → ผ่านไหม?
  2. ถ้าผ่าน → กด Approve แล้ว Merge ได้เลย
  3. ถ้า fail → ดู log ว่าพังเพราะอะไร → แก้หรือปิด PR
```
