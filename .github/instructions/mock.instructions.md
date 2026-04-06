---
description: "Use when creating, editing, or reviewing mock data files for FE development. Covers file naming, JSON format, routes.json rules, middleware behavior, and security constraints."
applyTo: "mockdata/**,server/middleware/mock.go,server/middleware/mock_test.go"
---

# Mock Data — กฎและวิธีใช้งาน

## 1. เปิดใช้เมื่อไร

Mock มีไว้ให้ **FE ใช้ระหว่างพัฒนา** ก่อนต่อ API จริง — ห้ามเปิดใน production

```env
MOCK_ENABLED=true
MOCK_ROUTES_FILE=mockdata/routes.json   # default, ไม่ต้องตั้งถ้าใช้ค่านี้
```

## 2. วิธีทำงานของ Middleware

```
Request → Mock Middleware → match? → ตอบ JSON จากไฟล์
                          → ไม่ match? → c.Next() → handler จริง
```

- Middleware อยู่ **ก่อน** auth middleware → ไม่ต้อง login ก่อนเรียก mock
- Response มี header `X-Mock: true` ให้ FE ตรวจสอบใน DevTools
- Status code ดึงจาก `status_code` ใน JSON file → FE test error handling ได้ครบ

## 3. โครงสร้าง mockdata/

```
mockdata/
├── routes.json              ← จับคู่ route → file (แก้ที่นี่เพื่อเปลี่ยน scenario)
├── _shared/                 ← Error response ที่ใช้ร่วมทุก module
│   ├── unauthorized.json
│   └── internal_error.json
├── auth/
│   ├── login_success.json
│   └── login_invalid.json
├── cmi/
│   ├── get_policy_success.json
│   └── get_policy_not_found.json
├── quotation/
│   ├── get_quotation_success.json
│   ├── get_quotation_not_found.json
│   ├── list_quotations_success.json
│   └── list_quotations_empty.json
└── externaldb/
    ├── check_all_success.json
    └── check_by_name_success.json
```

### กฎ

| กฎ | ทำไม |
|----|------|
| โฟลเดอร์ตั้งชื่อตาม **module** | หาง่าย, ตรงกับ `internal/modules/{name}/` |
| ห้ามใส่ไฟล์ mock ปนใน `internal/` | แยก concern — mock ไม่ใช่ business logic |
| `_shared/` ขึ้นต้นด้วย `_` | แยกชัดว่าไม่ใช่ module แต่ใช้ร่วมกัน |

## 4. Naming Convention

```
{action}_{resource}_{scenario}.json
```

| ส่วน | คำอธิบาย | ตัวอย่าง |
|------|---------|---------|
| action | verb ที่ทำ | `get`, `list`, `login`, `create`, `check` |
| resource | สิ่งที่ดำเนินการ | `policy`, `quotation`, `quotations` |
| scenario | ผลลัพธ์ | `success`, `not_found`, `invalid`, `empty` |

### ตัวอย่าง

```
login_success.json           ← POST /v1/auth/login (200)
login_invalid.json           ← POST /v1/auth/login (401)
get_policy_success.json      ← GET  /v1/cmi/:job_id/... (200)
get_policy_not_found.json    ← GET  /v1/cmi/:job_id/... (404)
list_quotations_empty.json   ← GET  /v1/quotations (200, empty list)
```

### ห้ามทำ

```
❌ data.json              → ไม่รู้ว่า action อะไร, scenario อะไร
❌ cmi.json               → ไม่บอก scenario
❌ test1.json             → ชื่อไม่มีความหมาย
❌ get-policy-success.json → ใช้ underscore ไม่ใช่ hyphen
```

## 5. JSON Format — ห้ามเบี่ยงจาก ApiResponse

ทุกไฟล์ **ต้อง** ตาม `dto.ApiResponse` format เดียวกับ API จริง

### Success

```json
{
  "status": "OK",
  "status_code": 200,
  "message": "success",
  "result": {
    "data": { ... }
  }
}
```

### Success with pagination

```json
{
  "status": "OK",
  "status_code": 200,
  "message": "success",
  "result": {
    "data": {
      "items": [...],
      "total": 3,
      "page": 1,
      "limit": 20,
      "totalPages": 1,
      "hasNext": false,
      "hasPrev": false
    }
  }
}
```

### Error

```json
{
  "status": "ERROR",
  "status_code": 404,
  "message": "job not found",
  "result": {
    "trace_id": "cmi-job-not-found"
  }
}
```

### กฎ JSON

| กฎ | ทำไม |
|----|------|
| `status_code` ต้องตรงกับ HTTP status ที่ต้องการ | middleware ดึงค่านี้เป็น HTTP status |
| `trace_id` ต้องตรงกับ `error_codes.go` | FE จะใช้ trace_id เดียวกับ API จริง |
| ห้ามใส่ field ที่ domain struct ไม่มี | FE จะพัง เมื่อสลับไปใช้ API จริง |
| ห้ามใส่ข้อมูลจริง (PII) | mock data ต้องเป็นข้อมูลสมมติเสมอ |
| ห้ามใส่ comment ใน JSON | JSON ไม่รองรับ comment |

## 6. routes.json — วิธีจัดการ

### Format

```json
[
  {
    "method": "GET",
    "path": "/v1/cmi/:job_id/request-policy-single-cmi",
    "file": "cmi/get_policy_success.json",
    "enabled": true
  }
]
```

| Field | คำอธิบาย |
|-------|---------|
| `method` | HTTP method: `GET`, `POST`, `PUT`, `DELETE` |
| `path` | Fiber-style path — `:param` คือ wildcard |
| `file` | path สัมพัทธ์จาก `mockdata/` |
| `enabled` | `true`/`false` — เปิด/ปิดรายตัว (ไม่ระบุ = `true`) |

### เปลี่ยน Scenario

แก้ `file` ให้ชี้ไปไฟล์อื่น:

```diff
  {
    "method": "GET",
    "path": "/v1/cmi/:job_id/request-policy-single-cmi",
-   "file": "cmi/get_policy_success.json",
+   "file": "cmi/get_policy_not_found.json",
    "enabled": true
  }
```

### ปิด Mock เฉพาะ Endpoint

ตั้ง `enabled` เป็น `false` → route นั้นจะ fall-through ไป handler จริง:

```diff
  {
    "method": "GET",
    "path": "/v1/quotations",
    "file": "quotation/list_quotations_success.json",
-   "enabled": true
+   "enabled": false
  }
```

ไม่ต้องลบ route ออก — แค่ `false` พอ เปิดกลับแค่เปลี่ยนเป็น `true`

### กฎ

| กฎ | ทำไม |
|----|------|
| `file` ห้ามมี `..` | ป้องกัน path traversal (middleware จะ block) |
| `enabled` ไม่ระบุ → default `true` | backward compatible กับ routes เดิม |
| route ซ้ำกัน → ใช้ตัวแรกที่ match | middleware อ่าน sequential, match แรกชนะ |
| route ไม่อยู่ใน `routes.json` → fall-through | ไปใช้ handler จริง |

## 7. วิธีเพิ่ม Mock ใหม่

1. สร้าง JSON file ใน `mockdata/{module}/` ตาม naming convention
2. เนื้อหาตาม `ApiResponse` format + `status_code` ที่ถูกต้อง
3. เพิ่ม entry ใน `routes.json`
4. Restart server (routes โหลดตอน startup)

## 8. Security

| กฎ | ทำไม |
|----|------|
| Production ห้ามเปิด `MOCK_ENABLED=true` | mock bypass auth ทั้งหมด |
| ห้ามใส่ข้อมูลจริงในไฟล์ mock | mock data อยู่ใน git |
| `file` path ห้ามมี `..` | middleware block path traversal |
| ไม่ serve static file ตรง | ป้องกัน directory listing |

## 9. ข้อจำกัดที่ยอมรับได้

| จุด | เหตุผล |
|-----|--------|
| โหลด routes.json ตอน startup เท่านั้น | เปลี่ยน route ต้อง restart — ยอมรับได้สำหรับ dev |
| ไม่รองรับ query string matching | เปลี่ยน scenario จาก `routes.json` แทน |
| Cache mock file ใน memory | ลด I/O — เปลี่ยนไฟล์ต้อง restart |
