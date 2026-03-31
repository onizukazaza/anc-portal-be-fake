# CI Pipeline Stages — อธิบายแต่ละ Stage บน GitHub Actions

## ภาพรวม

เมื่อเรา push code หรือสร้าง PR ไป `develop` / `main` GitHub Actions จะรัน **CI Pipeline**
ซึ่งประกอบด้วย **7 stages** (jobs) เรียงตามลำดับ:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          CI Pipeline                                    │
│                                                                         │
│   ┌───────┐  ┌───────┐  ┌───────────┐                                  │
│   │ Lint  │  │ Test  │  │ Vuln Check│    ← ด่านตรวจสอบ (รันขนาน)       │
│   │  46s  │  │ 1m24s │  │    20s    │                                   │
│   └───┬───┘  └───┬───┘  └─────┬─────┘                                  │
│       │          │             │                                        │
│       └──────────┼─────────────┘                                        │
│                  │                                                      │
│            ┌─────▼─────┐                                                │
│            │  Build    │    ← compile ผ่านไหม? (รอ 3 ด่านแรกผ่าน)      │
│            │   43s     │                                                │
│            └─────┬─────┘                                                │
│                  │                                                      │
│            ┌─────▼─────┐                                                │
│            │  Docker   │    ← build image + push ไป GHCR               │
│            │  2m 5s    │                                                │
│            └─────┬─────┘                                                │
│                  │                                                      │
│          ┌───────▼────────┐                                             │
│          │  Image Scan    │    ← สแกนช่องโหว่ใน Docker image           │
│          │     22s        │                                             │
│          └───────┬────────┘                                             │
│                  │                                                      │
│            ┌─────▼─────┐                                                │
│            │  Notify   │    ← แจ้งผลไป Discord                         │
│            │    5s     │                                                │
│            └───────────┘                                                │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## สถานะของแต่ละ Stage

บน GitHub Actions แต่ละ stage จะแสดงสถานะเป็นไอคอน:

| ไอคอน | สถานะ | ความหมาย |
|--------|--------|----------|
| ✅ (วงกลมเขียว) | **Success** | ผ่าน! ไม่มีปัญหา |
| ❌ (วงกลมแดง) | **Failure** | พัง! มี error ต้องแก้ |
| 🟡 (วงกลมเหลือง) | **In Progress** | กำลังรันอยู่ |
| ⚪ (วงกลมขาว) | **Queued** | รอคิว (ยังไม่เริ่ม) |
| ⏭️ (วงกลมเทา) | **Skipped** | ข้าม (เพราะ stage ก่อนหน้า fail หรือ condition ไม่ตรง) |

---

## Stage 1: Lint

```
✅ Lint    46s
```

### คืออะไร?
ตรวจสอบ **คุณภาพโค้ด** — ว่าเขียนตาม best practices, มี bug pattern ไหม, มีช่องโหว่ security ไหม

### เครื่องมือ
**golangci-lint v2** — รัน linters หลายตัวพร้อมกัน:

| Linter | ตรวจอะไร |
|--------|----------|
| errcheck | ลืม handle error return ไหม |
| govet | ตรวจ suspicious code patterns |
| staticcheck | วิเคราะห์โค้ดเชิงลึก (bugs, performance, style) |
| gosec | ตรวจช่องโหว่ security (SQL injection, hardcoded secrets) |
| bodyclose | ลืมปิด HTTP response body ไหม |
| noctx | HTTP request ไม่ส่ง context.Context |
| ineffassign | assign ค่าแล้วไม่ได้ใช้ |
| unused | ประกาศ variable/function แล้วไม่ได้ใช้ |

### Config
อ่านจากไฟล์ `.golangci.yml` อัตโนมัติ (timeout 5 นาที)

### ถ้า Fail?
แสดงว่าโค้ดมีปัญหาด้าน style, bug pattern, หรือ security → **ต้องแก้โค้ดก่อน push ใหม่**

---

## Stage 2: Test

```
✅ Test    1m 24s
```

### คืออะไร?
รัน **unit tests** ทั้งหมดในโปรเจกต์ + วัด **code coverage**

### คำสั่งที่รัน
```bash
go test -race -coverprofile=coverage.out -covermode=atomic -count=1 ./...
```

| Flag | ทำอะไร |
|------|--------|
| `-race` | เปิด race condition detector (ตรวจ concurrent bugs) |
| `-coverprofile` | สร้างไฟล์ coverage report |
| `-covermode=atomic` | นับ coverage แบบ atomic (รองรับ concurrent) |
| `-count=1` | ไม่ cache ผลเก่า รันจริงทุกครั้ง |
| `./...` | รันทุก package |

### Output
- **Coverage summary** — แสดงเปอร์เซ็นต์ coverage ใน GitHub Step Summary
- **Coverage artifact** — เก็บไฟล์ `coverage.out` ไว้ 14 วัน (ดาวน์โหลดได้)

### ถ้า Fail?
แสดงว่า test case ไม่ผ่าน → **ดู error log ว่า test ไหนพัง แล้วแก้โค้ดหรือ test**

---

## Stage 3: Vuln Check

```
✅ Vuln Check    20s
```

### คืออะไร?
ตรวจว่า **dependency** (library ที่ import มา) มี **known vulnerability** (ช่องโหว่ที่รู้แล้ว) หรือไม่

### เครื่องมือ
**govulncheck** — เครื่องมือจาก Go official team ที่ตรวจ vulnerability database

### วิธีทำงาน
```
go.mod → ดู dependency ที่ใช้ → เทียบกับ Go Vulnerability Database → รายงาน
```

### ถ้า Fail?
แสดงว่า dependency มีช่องโหว่ → **อัปเดต dependency version** (เหมือนที่เราแก้ grpc v1.79.2 → v1.79.3)

---

## Stage 4: Build

```
✅ Build    43s
```

### คืออะไร?
**Compile โค้ด Go** ทั้ง 5 binaries เพื่อตรวจว่า:
1. โค้ด compile ผ่านไหม (ไม่มี syntax error, missing import)
2. ldflags (build info) ใส่ถูกต้องไหม

### Binaries ที่ build
| Binary | หน้าที่ |
|--------|---------|
| `cmd/api` | HTTP API server หลัก |
| `cmd/worker` | Background job worker (Kafka consumer) |
| `cmd/migrate` | Database migration runner |
| `cmd/seed` | Seed data loader |
| `cmd/sync` | Data synchronization tool |

### ทำไมรอ Stage 1-3 ก่อน?
เพราะ Build ใช้เวลา → ถ้า Lint/Test/Vuln ไม่ผ่าน ก็ไม่มีประโยชน์จะ build (ประหยัด CI minutes)

```
needs: [lint, test, vuln]   ← ต้องรอ 3 stage แรกผ่านก่อน
```

### ถ้า Fail?
แสดงว่าโค้ด compile ไม่ผ่าน → **ดู error ว่า file/line ไหน แล้วแก้ syntax**

---

## Stage 5: Docker

```
✅ Docker    2m 5s
```

### คืออะไร?
**Build Docker images** แล้ว **push ไป GHCR** (GitHub Container Registry) — เป็นขั้นตอน "แพ็คของ" เพื่อเตรียม deploy

### Images ที่ build
| Image | Dockerfile Target | เนื้อหา |
|-------|-------------------|---------|
| `ghcr.io/onizukazaza/anc-portal-be-fake` | `api` | API + Worker + Migrate + Seed + Sync (full image) |
| `ghcr.io/onizukazaza/anc-portal-be-fake-worker` | `worker` | Worker only (lightweight) |

### Image Tags
ทุกครั้งที่ build จะติด tag หลายแบบ:

| Tag | ตัวอย่าง | ใช้ทำอะไร |
|-----|----------|-----------|
| `branch name` | `main`, `develop` | อ้างอิงตาม branch |
| `short SHA` | `c32a59d` | อ้างอิง commit ที่แน่นอน |
| `latest` | `latest` | เฉพาะ main branch (version ล่าสุด) |

### ทำเฉพาะ push event
```yaml
if: github.event_name == 'push'
```
ถ้าเป็น Pull Request จะ **ไม่ build Docker** (ไม่มีประโยชน์จะ push image จาก PR)

### ถ้า Fail?
สาเหตุที่พบบ่อย:
- `.dockerignore` exclude ไฟล์ที่จำเป็น (เคยเจอ `docs/` ถูก exclude)
- Base image ไม่มีบน Docker Hub
- `go build` ใน container fail (environment ต่างจาก local)

---

## Stage 6: Image Scan

```
✅ Image Scan    22s
```

### คืออะไร?
**สแกนช่องโหว่ใน Docker image** ที่เพิ่ง build — ตรวจทั้ง OS packages (Alpine) และ Go binaries

### เครื่องมือ
**Trivy** — container vulnerability scanner จาก Aqua Security

### วิธีทำงาน
```
Docker Image → Trivy สแกน
  ├── OS Layer (Alpine packages) → เทียบ Alpine CVE database
  └── App Layer (Go binary)     → เทียบ Go vulnerability database
```

### Severity ที่ตรวจ
| Severity | Exit Code 1 (fail)? | อัปโหลด SARIF? |
|----------|---------------------|----------------|
| **CRITICAL** | ✅ Yes | ✅ Yes |
| **HIGH** | ✅ Yes | ✅ Yes |
| **MEDIUM** | ❌ No (แค่รายงาน) | ✅ Yes |
| LOW | ❌ No | ❌ No |

### ขั้นตอน
1. สแกน **API image** → แสดงผลเป็นตาราง + สร้าง SARIF
2. สแกน **Worker image** → แสดงผลเป็นตาราง + สร้าง SARIF
3. อัปโหลด SARIF ไป **GitHub Security tab** → ดูรายละเอียดช่องโหว่ได้

### ตัวอย่างผลสแกน (ที่เคยเจอ)
```
┌────────────────────────┬────────────────┬──────────┬───────────────────┬───────────────┐
│        Library         │ Vulnerability  │ Severity │ Installed Version │ Fixed Version │
├────────────────────────┼────────────────┼──────────┼───────────────────┼───────────────┤
│ google.golang.org/grpc │ CVE-2026-33186 │ CRITICAL │ v1.79.2           │ 1.79.3        │
└────────────────────────┴────────────────┴──────────┴───────────────────┴───────────────┘
```

### ถ้า Fail?
แสดงว่า image มี **CRITICAL** หรือ **HIGH vulnerability** → **อัปเดต dependency/base image** ที่มีปัญหา

---

## Stage 7: Notify

```
✅ Notify    5s
```

### คืออะไร?
ส่ง **สรุปผล CI ทั้งหมด** ไปแจ้งใน **Discord** ผ่าน Webhook

### รันเสมอ (if: always())
ไม่ว่า pipeline จะ pass หรือ fail → Notify จะรันเสมอเพื่อแจ้งผล

### ข้อมูลที่แจ้ง
| ข้อมูล | ตัวอย่าง |
|--------|----------|
| สถานะรวม | ✅ CI Pipeline Passed / ❌ CI Pipeline Failed |
| ผลแต่ละ job | ✅ Lint  ✅ Test  ✅ Vuln  ✅ Build  ✅ Docker  ✅ Scan |
| Branch | `main` |
| Commit | `c32a59d` |
| ใครทำ | `onizukazaza` |
| Commit message | `fix: upgrade grpc v1.79.3...` |
| ลิงก์ไป Actions | คลิกดู log ได้ |
| Failed steps (ถ้า fail) | ⚠ Failed Steps: Scan |

### ถ้า Fail?
Notify ไม่ค่อย fail — ถ้า fail แสดงว่า Discord Webhook URL ไม่ถูกต้องหรือ Discord ล่ม

---

## ความสัมพันธ์ระหว่าง Stages

### Parallel (รันขนาน)
```
Lint  ─┐
Test  ─┤   ← 3 stages นี้รันพร้อมกัน ประหยัดเวลา
Vuln  ─┘
```

### Sequential (รันต่อเนื่อง)
```
[Lint+Test+Vuln] → Build → Docker → Image Scan → Notify
                   needs:   needs:    needs:       needs:
                   [1,2,3]  [build]   [docker]     [all]
```

### เมื่อ Stage พัง จะเกิดอะไร?

| Stage ที่พัง | ผลกระทบ |
|-------------|---------|
| Lint fail | Build ไม่รัน → Docker ไม่รัน → Scan ไม่รัน → Notify แจ้ง fail |
| Test fail | เหมือน Lint fail |
| Vuln fail | เหมือน Lint fail |
| Build fail | Docker ไม่รัน → Scan ไม่รัน → Notify แจ้ง fail |
| Docker fail | Scan ไม่รัน → Notify แจ้ง fail |
| Scan fail | Notify แจ้ง fail (แต่ image ถูก push ไปแล้ว) |
| Notify fail | ไม่มีผลกระทบอื่น (แค่ไม่ได้แจ้ง Discord) |

---

## เวลาโดยรวม

```
ด่านขนาน:      ~1m 30s  (Lint + Test + Vuln รันพร้อมกัน ใช้เวลาของตัวที่นานสุด)
Build:          ~43s
Docker:         ~2m 5s
Image Scan:     ~22s
Notify:         ~5s
────────────────────────
รวม:            ~5-6 นาที
```

### ทำไมใช้เวลานาน?
- **Docker** ใช้เวลามากสุดเพราะต้อง build Go binary + push image 2 ตัว
- **Test** ใช้เวลาเพราะรัน race detection + coverage
- ถ้ามี **cache hit** (go modules, docker layers) จะเร็วขึ้นมาก

---

## PR vs Push — ทำงานต่างกันยังไง?

| Behavior | Push Event | Pull Request Event |
|----------|------------|---------------------|
| Lint | ✅ รัน | ✅ รัน |
| Test | ✅ รัน | ✅ รัน |
| Vuln Check | ✅ รัน | ✅ รัน |
| Build | ✅ รัน | ✅ รัน |
| Docker | ✅ build + **push** image | ⏭️ **skip** (ไม่ push image จาก PR) |
| Image Scan | ✅ สแกน image ที่ push | ⏭️ **skip** (ไม่มี image ให้สแกน) |
| Notify | ✅ แจ้ง Discord | ✅ แจ้ง Discord |

---

## Concurrency Control

```yaml
concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true
```

### หมายความว่า?
ถ้า push commit ใหม่ขณะที่ CI เก่ายังรันอยู่ → **CI เก่าจะถูกยกเลิก** → รันเฉพาะ CI ใหม่

### ทำไมทำแบบนี้?
- ประหยัด CI minutes (ไม่ต้องรัน commit เก่าที่ outdated แล้ว)
- ผลลัพธ์จาก commit เก่าไม่มีประโยชน์แล้ว

---

## สรุป

| Stage | ทำอะไร | เปรียบเทียบ |
|-------|--------|------------|
| **Lint** | ตรวจคุณภาพโค้ด | เหมือน "ตรวจสอบคุณภาพวัตถุดิบ" |
| **Test** | รัน unit tests | เหมือน "ทดสอบสินค้าในห้อง lab" |
| **Vuln Check** | ตรวจ dependency ปลอดภัย | เหมือน "เช็คว่าวัตถุดิบไม่มีสารปนเปื้อน" |
| **Build** | compile โค้ด | เหมือน "ประกอบสินค้า" |
| **Docker** | แพ็คเป็น container image | เหมือน "บรรจุกล่อง" |
| **Image Scan** | สแกนช่องโหว่ใน image | เหมือน "ตรวจสอบคุณภาพสินค้าสำเร็จรูป" |
| **Notify** | แจ้งผลไป Discord | เหมือน "รายงานผลให้ทีม" |

> ทุก stage มีหน้าที่ชัดเจน ถ้า stage ใดพัง → stage ถัดไปจะไม่รัน (ยกเว้น Notify)
> ดังนั้นปัญหาจะถูกจับได้เร็ว ไม่ปล่อยโค้ดแย่ ๆ ไปถึงขั้น deploy
