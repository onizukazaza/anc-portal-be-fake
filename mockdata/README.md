# Mock Data — FE Development

ข้อมูล mock สำหรับ Frontend ใช้ระหว่างพัฒนา ก่อนต่อ API จริง

## วิธีเปิดใช้

```yaml
# config.yaml
mock:
  enabled: true
  routesFile: "mockdata/routes.json"
```

```env
# หรือใช้ env
MOCK_ENABLED=true
MOCK_ROUTES_FILE=mockdata/routes.json
```

## วิธีทำงาน

1. Middleware อ่าน `routes.json` ตอน server startup
2. เมื่อ request เข้ามา → จับคู่ method + path กับ route ที่กำหนด
3. ถ้า match → ตอบ JSON จาก mock file พร้อม status code ที่ถูกต้อง
4. ถ้าไม่ match → ส่งต่อไป handler จริง (fall-through)
5. Response จะมี header `X-Mock: true` ให้ FE ตรวจสอบได้

## โครงสร้าง

```
mockdata/
├── routes.json                          ← Index: จับคู่ route → file
├── _shared/                             ← Error response ที่ใช้ร่วมทุก module
│   ├── unauthorized.json
│   └── internal_error.json
├── auth/
│   ├── login_success.json               ← POST /v1/auth/login (200)
│   └── login_invalid.json               ← POST /v1/auth/login (401)
├── cmi/
│   ├── get_policy_success.json          ← GET  /v1/cmi/:job_id/... (200)
│   └── get_policy_not_found.json        ← GET  /v1/cmi/:job_id/... (404)
├── quotation/
│   ├── get_quotation_success.json       ← GET  /v1/quotations/:id (200)
│   ├── get_quotation_not_found.json     ← GET  /v1/quotations/:id (404)
│   ├── list_quotations_success.json     ← GET  /v1/quotations (200)
│   └── list_quotations_empty.json       ← GET  /v1/quotations (200, empty)
└── externaldb/
    ├── check_all_success.json           ← GET  /v1/external-db/health (200)
    └── check_by_name_success.json       ← GET  /v1/external-db/health/:name (200)
```

## Naming Convention

```
{action}_{resource}_{scenario}.json
```

| ตัวอย่าง | คำอธิบาย |
|----------|---------|
| `login_success.json` | Login สำเร็จ |
| `login_invalid.json` | Login ผิด password |
| `get_policy_success.json` | ดึง policy สำเร็จ |
| `get_policy_not_found.json` | ไม่พบ policy |
| `list_quotations_empty.json` | List แต่ไม่มีข้อมูล |

## JSON Format

ทุกไฟล์ใช้ `ApiResponse` format เดียวกับ API จริง:

**Success:**
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

**Error:**
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

## วิธีเปลี่ยน Scenario

แก้ `file` ใน `routes.json` ให้ชี้ไปไฟล์ที่ต้องการ:

```diff
  {
    "method": "GET",
    "path": "/v1/cmi/:job_id/request-policy-single-cmi",
-   "file": "cmi/get_policy_success.json"
+   "file": "cmi/get_policy_not_found.json"
  }
```

## วิธีเพิ่ม Mock ใหม่

1. สร้าง JSON file ใน `mockdata/{module}/` ตาม format ด้านบน
2. เพิ่ม entry ใน `routes.json`
3. Restart server (mock routes โหลดตอน startup)

## หมายเหตุ

- Mock middleware จะทำงาน **ก่อน** auth middleware → ไม่ต้อง login ก่อนเรียก
- Route ที่ **ไม่มี** ใน `routes.json` จะ fall-through ไปใช้ handler จริง
- ใช้งานเฉพาะ **local/dev** เท่านั้น — production ต้อง `MOCK_ENABLED=false`
