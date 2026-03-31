# Code Coverage Concept — ANC Portal Backend

> **v1.0** — Last updated: March 2026
>
> แนวคิด Code Coverage สำหรับโปรเจกต์ ครอบคลุม: ทำไมต้องมี, วิธีใช้, เกณฑ์, การอ่านผล
> และการนำไปใช้ใน CI/CD Pipeline

---

## สารบัญ

1. [Code Coverage คืออะไร](#1-code-coverage-คืออะไร)
2. [ทำไมต้องมี Coverage](#2-ทำไมต้องมี-coverage)
3. [ประเภทของ Coverage](#3-ประเภทของ-coverage)
4. [วิธีรัน Coverage ในโปรเจกต์](#4-วิธีรัน-coverage-ในโปรเจกต์)
5. [การอ่านผล Coverage](#5-การอ่านผล-coverage)
6. [เกณฑ์ Coverage Threshold](#6-เกณฑ์-coverage-threshold)
7. [Coverage ใน CI/CD Pipeline](#7-coverage-ใน-cicd-pipeline)
8. [สิ่งที่ Coverage บอกไม่ได้](#8-สิ่งที่-coverage-บอกไม่ได้)
9. [Best Practices](#9-best-practices)
10. [ตัวอย่างจริงจากโปรเจกต์](#10-ตัวอย่างจริงจากโปรเจกต์)

---

## 1. Code Coverage คืออะไร

**Code Coverage** = ตัวเลขเปอร์เซ็นต์ที่บอกว่า test ของเราวิ่งผ่านโค้ดกี่บรรทัด

```
ถ้าโค้ดมี 100 บรรทัด
test วิ่งผ่าน 80 บรรทัด
→ Coverage = 80%
```

### อุปมา

นึกภาพว่าโค้ดคือ **ถนนในเมือง** — coverage บอกว่า test ของเราขับรถผ่านกี่เปอร์เซ็นต์ของถนนทั้งหมด

- 100% → ขับผ่านทุกถนน (ไม่ได้แปลว่าถนนไม่มีหลุม)
- 50% → ครึ่งเมืองยังไม่เคยขับผ่าน (อาจมีปัญหาที่ไม่รู้)
- 0% → ไม่เคยขับเลย (ไม่รู้อะไรเลย)

---

## 2. ทำไมต้องมี Coverage

| เหตุผล | รายละเอียด |
|--------|-----------|
| **รู้จุดบอด** | เห็นว่าโค้ดส่วนไหนยังไม่มี test ครอบคลุม |
| **ป้องกัน regression** | ยิ่ง coverage สูง ยิ่งจับ bug ได้เร็ว |
| **มั่นใจตอน refactor** | แก้โค้ดแล้วรัน test ถ้า coverage ไม่ลด = ปลอดภัย |
| **วัดคุณภาพได้** | เป็นตัวเลขที่ tracking ได้ใน CI/CD |
| **ISO 25010 / 9001** | เป็นหนึ่งในเกณฑ์คุณภาพซอฟต์แวร์ตามมาตรฐาน |

---

## 3. ประเภทของ Coverage

### 3.1 Line Coverage (โปรเจกต์นี้ใช้)

วัดว่า **กี่บรรทัด** ที่ test วิ่งผ่าน

```go
func Add(a, b int) int {     // ← บรรทัด 1
    return a + b              // ← บรรทัด 2
}
```

ถ้า test เรียก `Add(1, 2)` → coverage **100%** (วิ่งผ่านทั้ง 2 บรรทัด)

### 3.2 Branch Coverage

วัดว่าทุก **เงื่อนไข if/else** ถูก test ครบทุก case หรือยัง

```go
func Divide(a, b int) (int, error) {
    if b == 0 {                          // ← branch 1 (true)
        return 0, errors.New("zero")
    }
    return a / b, nil                    // ← branch 2 (false)
}
```

| Test | Branch ที่ผ่าน | Branch Coverage |
|------|---------------|----------------|
| `Divide(10, 2)` | false only | 50% |
| `Divide(10, 2)` + `Divide(10, 0)` | true + false | 100% |

### 3.3 Function Coverage

วัดว่า **function ไหน** ถูกเรียกบ้าง

```
ถ้ามี 10 functions
test เรียก 7 functions
→ Function Coverage = 70%
```

### 3.4 Statement Coverage (atomic — โปรเจกต์นี้ใช้)

Go ใช้ `-covermode=atomic` ซึ่งวัดระดับ **statement** + รองรับ concurrent test

```
atomic = นับจำนวนครั้งที่แต่ละ statement ถูก execute อย่างปลอดภัยแม้ test รัน parallel
```

---

## 4. วิธีรัน Coverage ในโปรเจกต์

### 4.1 PowerShell (Windows — แนะนำ)

```powershell
# รัน coverage พร้อม threshold check
.\run.ps1 test-cover

# ผลลัพธ์:
# [test-cover] Running tests with coverage (threshold: 70%)...
#
#   Coverage by package:
#   --------------------
#   ...internal/modules/auth/app    service.go      Login    85.7%
#   ...internal/shared/pagination   pagination.go   Parse    100.0%
#   total:                          (statements)             78.3%
#
#   PASS: coverage 78.3% meets threshold 70%
```

### 4.2 Makefile (Linux/macOS/CI)

```bash
# รัน coverage (default threshold 70%)
make test-cover

# กำหนด threshold เอง
make test-cover COVERAGE_THRESHOLD=80
```

### 4.3 Go Command ตรง ๆ

```bash
# สร้าง coverage profile
go test -coverprofile=coverage.out -covermode=atomic ./...

# ดูผลแบบ text
go tool cover -func=coverage.out

# ดูผลแบบ HTML (เปิดในเบราว์เซอร์ — เห็นโค้ดที่สีเขียว/แดง)
go tool cover -html=coverage.out
```

### 4.4 ไฟล์ที่สร้าง

| ไฟล์ | คำอธิบาย |
|------|---------|
| `coverage.out` | Coverage profile (raw data) |
| `coverage.html` | HTML report (ถ้าใช้ `-html` flag) |

> **หมายเหตุ:** `coverage.out` ควรอยู่ใน `.gitignore`

---

## 5. การอ่านผล Coverage

### 5.1 ผลแบบ Text (`go tool cover -func`)

```
github.com/.../auth/app/service.go:23:    Login           85.7%
github.com/.../auth/app/service.go:58:    Register        100.0%
github.com/.../cmi/app/service.go:19:     FindPolicy      76.5%
github.com/.../pagination/pagination.go:8: Parse           100.0%
total:                                    (statements)    78.3%
```

อ่านอย่างไร:
- **ซ้าย** = ไฟล์ + function
- **ขวา** = เปอร์เซ็นต์ coverage ของ function นั้น
- **total** = coverage รวมทั้งโปรเจกต์

### 5.2 ผลแบบ HTML

```powershell
go tool cover -html=coverage.out
```

เปิดเบราว์เซอร์แล้วจะเห็น:
- 🟩 **สีเขียว** = โค้ดที่ test วิ่งผ่าน
- 🟥 **สีแดง** = โค้ดที่ test ยังไม่ครอบคลุม
- สามารถเลือกดูทีละ package จาก dropdown

### 5.3 วิธีหา "จุดที่ควรเพิ่ม test"

1. เปิด HTML report
2. ดู function ที่สีแดงเยอะ
3. focus เพิ่ม test ที่ **business logic สำคัญ** ก่อน (เช่น Service layer)
4. ไม่ต้องไล่ทำ 100% ทุก function — เลือกจุดที่ impact สูง

---

## 6. เกณฑ์ Coverage Threshold

### เกณฑ์ของโปรเจกต์นี้

```
Threshold ปัจจุบัน: 70%
```

### มาตรฐานทั่วไป

| ระดับ | Coverage | ใช้เมื่อ |
|-------|----------|---------|
| ขั้นต่ำ | 50-60% | โปรเจกต์เริ่มต้น, legacy code |
| พอใช้ | 60-70% | โปรเจกต์ทั่วไป |
| ดี | **70-80%** | **← โปรเจกต์นี้ตั้งเป้าไว้** |
| ดีมาก | 80-90% | โปรเจกต์ที่ critical |
| ยอดเยี่ยม | 90%+ | Library, framework, financial system |

### ทำไมตั้ง 70% ไม่ใช่ 100%

| เหตุผล | รายละเอียด |
|--------|-----------|
| **100% ไม่คุ้ม** | ต้องเขียน test สำหรับ edge case ที่ไม่เกิดจริง เปลือง effort |
| **70-80% คุ้มค่าสุด** | ครอบคลุม business logic หลัก + error path สำคัญ |
| **Diminishing returns** | จาก 80% เป็น 90% ใช้ effort มากกว่า 0% เป็น 80% |

---

## 7. Coverage ใน CI/CD Pipeline

### Flow ปัจจุบัน

```
Developer                     CI Pipeline
─────────                    ────────────
push code ──────────────▶   [1] Lint
                             [2] Test ← ยังไม่มี coverage
                             [3] Vuln
                             [4] Build
```

### Flow ที่แนะนำ (เพิ่ม coverage)

```
Developer                     CI Pipeline
─────────                    ────────────
push code ──────────────▶   [1] Lint
                             [2] Test + Coverage ← เพิ่ม --coverprofile
                                 └─ check threshold ≥ 70%
                                 └─ upload coverage report
                             [3] Vuln
                             [4] Build
```

### GitHub Actions ตัวอย่าง

```yaml
# .github/workflows/ci.yml (เพิ่มใน test step)
- name: Test with Coverage
  run: |
    go test -coverprofile=coverage.out -covermode=atomic ./...
    TOTAL=$(go tool cover -func=coverage.out | grep total | awk '{print $NF}' | tr -d '%')
    echo "Coverage: ${TOTAL}%"
    if (( $(echo "$TOTAL < 70" | bc -l) )); then
      echo "::error::Coverage ${TOTAL}% is below 70% threshold"
      exit 1
    fi

- name: Upload Coverage
  uses: actions/upload-artifact@v4
  with:
    name: coverage-report
    path: coverage.out
```

---

## 8. สิ่งที่ Coverage บอกไม่ได้

| Coverage บอกได้ | Coverage บอกไม่ได้ |
|----------------|-------------------|
| โค้ดบรรทัดไหน test วิ่งผ่าน | Test มีคุณภาพดีไหม |
| Function ไหนยังไม่มี test | Business logic ถูกต้องไหม |
| ภาพรวมว่าครอบคลุมแค่ไหน | Edge case ครบไหม |
| Trend ว่า coverage ขึ้นหรือลง | Architecture ดีไหม |

### ตัวอย่าง coverage สูงแต่ test แย่

```go
// test ที่ coverage 100% แต่ไม่ได้ตรวจอะไรเลย
func TestLogin(t *testing.T) {
    svc := NewService(...)
    svc.Login(ctx, "admin", "password")  // ← ไม่มี assertion!
    // coverage 100% แต่ไม่รู้ว่าผลถูกหรือผิด
}
```

> **บทเรียน:** Coverage เป็นแค่ตัวชี้วัด **ปริมาณ** ไม่ใช่ **คุณภาพ** ของ test

---

## 9. Best Practices

### ควรทำ

- ✅ ตั้ง **threshold** ขั้นต่ำใน CI (โปรเจกต์นี้ = 70%)
- ✅ ดู **coverage trend** ว่าขึ้นหรือลงเมื่อเวลาผ่านไป
- ✅ Focus coverage ที่ **Service layer** (business logic) ก่อน
- ✅ ดู HTML report เพื่อหา **จุดแดง** ที่สำคัญ
- ✅ เขียน **assertion** ทุก test (ไม่ใช่แค่เรียก function)
- ✅ ใส่ `coverage.out` ใน `.gitignore`

### ไม่ควรทำ

- ❌ ไม่ไล่ทำ 100% ทุก function (diminishing returns)
- ❌ ไม่เขียน test แค่เพื่อเพิ่ม coverage โดยไม่มี assertion
- ❌ ไม่ skip test เมื่อ coverage ต่ำ (ต้องหาสาเหตุ)
- ❌ ไม่ลด threshold เพื่อให้ CI ผ่าน
- ❌ ไม่ test generated code (เช่น swagger docs, protobuf)

---

## 10. ตัวอย่างจริงจากโปรเจกต์

### Package ที่ควรมี Coverage สูง (Business Logic)

| Package | เหตุผล | เป้าหมาย |
|---------|--------|----------|
| `auth/app` | Login/Register logic | ≥ 80% |
| `cmi/app` | CMI policy search | ≥ 80% |
| `quotation/app` | CRUD quotation | ≥ 80% |
| `externaldb/app` | DB health check | ≥ 70% |
| `shared/pagination` | Page calculation | ≥ 90% |
| `shared/validator` | Input validation | ≥ 80% |

### Package ที่ Coverage ต่ำได้ (Infrastructure)

| Package | เหตุผล | เป้าหมาย |
|---------|--------|----------|
| `cmd/*` | Entrypoint — wire เฉย ๆ | ไม่บังคับ |
| `config` | Env loading | ≥ 50% |
| `database/*` | DB connection | ไม่บังคับ |
| `server` | HTTP server setup | ≥ 60% |

### คำสั่ง Quick Start

```powershell
# 1. รัน coverage
.\run.ps1 test-cover

# 2. ดู HTML report
go tool cover -html=coverage.out

# 3. ดูเฉพาะ package ที่ต่ำ
go tool cover -func=coverage.out | Where-Object { $_ -match '\d+\.\d+%' } |
    ForEach-Object { if ($_ -match '(\d+\.\d+)%' -and [double]$Matches[1] -lt 70) { $_ } }
```

---

> **สรุป:** Coverage เป็น **เครื่องมือวัด** ไม่ใช่เป้าหมาย — ใช้เพื่อหาจุดบอดและป้องกัน regression
> ตั้ง threshold ที่ **70%** แล้ว focus เพิ่ม test ที่ **business logic สำคัญ** ก่อน
