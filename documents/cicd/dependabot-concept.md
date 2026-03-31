# Dependabot Concept — Automated Dependency Updates

## Dependabot คืออะไร?

**Dependabot** คือบอทของ GitHub ที่คอยตรวจสอบ dependency (library, framework, base image)
ที่โปรเจกต์เราใช้อยู่ว่า **มีเวอร์ชันใหม่กว่าหรือยัง** — ถ้ามี มันจะ **สร้าง Pull Request** ให้อัตโนมัติ
เพื่อให้เราตรวจสอบแล้ว merge ได้เลย

ให้นึกภาพว่า Dependabot คือ **"พนักงานตรวจสต๊อก"**:
ทุกสัปดาห์มันจะเข้ามาดู dependency ทั้งหมด → ถ้ามีของรุ่นใหม่ก็เขียนใบเสนอ (PR) มาให้เราอนุมัติ

---

## ทำไมถึงต้องมี?

### ปัญหาที่เจอถ้าไม่มี Dependabot

| ปัญหา | ผลกระทบ |
|--------|---------|
| Library มีช่องโหว่ (CVE) แต่เราไม่รู้ | โดน exploit ได้ เสี่ยงข้อมูลรั่ว |
| ปล่อยไม่อัปเดตนาน ๆ แล้วอัปทีเดียว | Version gap ใหญ่ แก้ breaking changes ยาก |
| ต้องเข้าไปเช็คเว็บ / changelog เอง | เสียเวลา ลืมง่าย |
| Docker base image เก่า | มี CVE สะสม image ไม่ปลอดภัย |
| GitHub Actions version เก่า | อาจ deprecated หรือมี bug |

### Dependabot แก้ปัญหาเหล่านี้ได้ทั้งหมด โดยอัตโนมัติ

---

## Config ของโปรเจกต์นี้

ไฟล์ `.github/dependabot.yml` ตั้งค่าให้ตรวจ **3 ระบบ**:

### 1. Go Modules (`gomod`)

```yaml
- package-ecosystem: gomod
  directory: /
  schedule:
    interval: weekly
    day: monday
    time: "09:00"
    timezone: Asia/Bangkok
```

- ตรวจ dependency ใน `go.mod` ทุก **วันจันทร์ 09:00 น. (เวลาไทย)**
- เปิด PR ได้สูงสุด **5 ตัว** พร้อมกัน
- ติด label: `dependencies`, `go`
- Commit message: `deps(scope): ...`

**ตัวอย่าง PR ที่สร้าง:**
> `deps(go): bump github.com/go-playground/validator/v10 from 10.30.0 to 10.30.1`

### 2. GitHub Actions (`github-actions`)

```yaml
- package-ecosystem: github-actions
  directory: /
  schedule:
    interval: weekly
    day: monday
    time: "09:00"
    timezone: Asia/Bangkok
```

- ตรวจ version ของ Actions ที่ใช้ใน `.github/workflows/*.yml`
- เช่น `actions/checkout@v4` → `@v5`, `docker/build-push-action@v6` → `@v7`
- ติด label: `dependencies`, `ci`
- Commit message: `ci(scope): ...`

### 3. Docker Base Images (`docker`)

```yaml
- package-ecosystem: docker
  directory: /deployments/docker
  schedule:
    interval: weekly
    day: monday
    time: "09:00"
    timezone: Asia/Bangkok
```

- ตรวจ base image ใน `Dockerfile` เช่น `golang:1.25-alpine`, `alpine:3.21`
- เปิด PR ได้สูงสุด **3 ตัว**
- ติด label: `dependencies`, `docker`
- Commit message: `docker(scope): ...`

---

## Flow การทำงาน

```
ทุกวันจันทร์ 09:00 (Asia/Bangkok)
        │
        ▼
┌───────────────────┐
│  Dependabot scan  │
│  ตรวจ 3 ระบบ:     │
│  - go.mod         │
│  - workflows/     │
│  - Dockerfile     │
└───────┬───────────┘
        │
   มี version ใหม่?
        │
   ├── ไม่มี → จบ (ไม่ทำอะไร)
   │
   └── มี → สร้าง PR อัตโนมัติ
              │
              ▼
     ┌─────────────────┐
     │  Pull Request    │
     │  - title ชัดเจน  │
     │  - label ติดให้   │
     │  - changelog link│
     │  - compatibility │
     └────────┬────────┘
              │
              ▼
     ┌─────────────────┐
     │  CI Workflow รัน │  ← Lint, Test, Build, Vuln
     │  ตรวจว่า update  │     ทำงานร่วมกับ CI pipeline
     │  ไม่ทำอะไรพัง    │
     └────────┬────────┘
              │
         CI ผ่าน?
              │
    ├── ผ่าน → Review & Merge ได้เลย ✅
    │
    └── ไม่ผ่าน → ดู error → อาจต้อง fix / ปิด PR ❌
```

---

## ข้อดีของ Dependabot

### ด้าน Security

| ข้อดี | รายละเอียด |
|--------|-----------|
| **Security alerts** | GitHub จะแจ้งเตือนถ้า dependency มี CVE (ช่องโหว่) |
| **Auto-fix vulnerabilities** | สร้าง PR แก้ช่องโหว่โดยอัตโนมัติ |
| **ลดเวลา patch** | ไม่ต้องรอ dev เข้ามาเช็คเอง — มี PR พร้อม merge ทันที |
| **Audit trail** | ทุก dependency update มี PR + CI history เป็นหลักฐาน |

### ด้าน Maintenance

| ข้อดี | รายละเอียด |
|--------|-----------|
| **ไม่ต้องเช็คเอง** | Dependabot ทำให้ทุกสัปดาห์อัตโนมัติ |
| **อัปเดตทีละนิด** | PR ละ 1 dependency → ถ้าพังก็รู้ทันทีว่าตัวไหน |
| **ลด version gap** | อัปเดตบ่อย ๆ → ไม่ต้องกระโดดข้ามหลาย major version |
| **Commit message สวย** | prefix + scope ชัดเจน → อ่าน git log ง่าย |
| **Label จัดหมวด** | filter PR ได้ตาม label (`go`, `ci`, `docker`) |

### ด้าน Developer Experience

| ข้อดี | รายละเอียด |
|--------|-----------|
| **Zero config effort** | ตั้งค่าครั้งเดียว ทำงานตลอด |
| **PR พร้อม context** | มี release notes, changelog, compatibility score |
| **ทำงานร่วมกับ CI** | PR เปิดมา → CI รันอัตโนมัติ → เห็นผลทันที |
| **ฟรี** | เป็น feature ในตัวของ GitHub ไม่มีค่าใช้จ่าย |

---

## ตัวอย่าง PR ที่ Dependabot สร้าง

โปรเจกต์นี้ Dependabot เคยสร้าง PR มาแล้ว เช่น:

| PR | ประเภท | อัปเดตอะไร |
|-----|--------|-----------|
| Bump `go-playground/validator` | Go module | security fix / new feature |
| Bump `rs/zerolog` | Go module | logger update |
| Bump `golang:alpine` | Docker | base image patch |
| Bump `alpine:3.21` → `3.22` | Docker | runtime image update |

---

## การตั้งค่าที่ควรรู้

### `open-pull-requests-limit`

```yaml
open-pull-requests-limit: 5
```

จำกัดจำนวน PR ที่เปิดพร้อมกันได้ — ป้องกัน PR ท่วม (โดยเฉพาะถ้า dependency เยอะ)

### `schedule.interval`

| ค่า | ความหมาย |
|-----|----------|
| `daily` | ตรวจทุกวัน |
| `weekly` | ตรวจทุกสัปดาห์ (แนะนำ ✅) |
| `monthly` | ตรวจทุกเดือน |

โปรเจกต์นี้ใช้ `weekly` วันจันทร์ — เหมาะกับจังหวะเริ่มสัปดาห์ review + merge ได้ช่วงเช้า

### `commit-message`

```yaml
commit-message:
  prefix: "deps"
  include: scope
```

ได้ commit message แบบ: `deps(go.mod): bump xxx from v1.0.0 to v1.1.0`
→ สอดคล้องกับ **Conventional Commits** ที่ใช้ในโปรเจกต์

### `labels`

```yaml
labels:
  - dependencies
  - go
```

ทุก PR จะถูกติด label อัตโนมัติ → สะดวกใน filter, search, dashboard

---

## เปรียบเทียบ: มี vs ไม่มี Dependabot

| ด้าน | ไม่มี Dependabot | มี Dependabot |
|------|-------------------|---------------|
| **เช็ค dependency** | ต้องทำเอง / ลืม | อัตโนมัติทุกสัปดาห์ |
| **ช่องโหว่** | รู้ช้า / ไม่รู้เลย | แจ้ง + สร้าง PR ทันที |
| **อัปเดต** | กระโดดข้ามหลาย version | ค่อย ๆ ทีละ version |
| **เวลา dev** | ต้องนั่งเช็ค changelog เอง | Dependabot สรุปมาให้ |
| **CI integration** | ต้อง test เอง | CI รันอัตโนมัติใน PR |
| **Audit** | ไม่มี trail | ทุก update มี PR history |

---

## สิ่งที่ควรระวัง

1. **อย่า merge โดยไม่ดู** — Dependabot ไม่รับประกันว่า update จะไม่ breaking
2. **ดู CI result ก่อน merge** — ถ้า CI fail แสดงว่า version ใหม่มีปัญหา
3. **Major version bump ต้องระวัง** — เช่น `v1` → `v2` อาจมี breaking changes
4. **Auto-merge ควรเปิดเฉพาะ patch** — minor/major ควร review ก่อน

---

## ไฟล์ที่เกี่ยวข้อง

| ไฟล์ | หน้าที่ |
|------|---------|
| `.github/dependabot.yml` | Config ของ Dependabot |
| `.github/workflows/ci.yml` | CI pipeline ที่รันเมื่อ Dependabot เปิด PR |
| `go.mod` / `go.sum` | Go dependency ที่ Dependabot ตรวจ |
| `deployments/docker/Dockerfile` | Docker base images ที่ Dependabot ตรวจ |
| `.github/workflows/*.yml` | GitHub Actions versions ที่ Dependabot ตรวจ |

---

## สรุปสั้น ๆ

> **Dependabot = บอทตรวจสต๊อก dependency อัตโนมัติ**
>
> ตรวจทุกจันทร์ 09:00 → เจอ version ใหม่ → สร้าง PR → CI รัน → Review & Merge
>
> ข้อดีหลัก: **ปลอดภัยกว่า, อัปเดตเร็วกว่า, ไม่ต้องเสียเวลาเช็คเอง, ฟรี**
