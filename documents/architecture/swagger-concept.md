# Swagger (OpenAPI) — คู่มือการใช้งาน

> **v2.2** — Last updated: March 2026
>
> Swagger สร้าง API documentation อัตโนมัติจาก code annotation
> พร้อม UI สำหรับทดสอบ API ได้ทันที

---

## สารบัญ

- [Swagger (OpenAPI) — คู่มือการใช้งาน](#swagger-openapi--คู่มือการใช้งาน)
  - [สารบัญ](#สารบัญ)
  - [Swagger คืออะไร](#swagger-คืออะไร)
  - [ข้อดีหลัก](#ข้อดีหลัก)
  - [เหมาะกับใคร](#เหมาะกับใคร)
    - [Developer](#developer)
    - [QA](#qa)
    - [Business / PM](#business--pm)
  - [การใช้งาน](#การใช้งาน)
    - [คำสั่ง](#คำสั่ง)
    - [เข้าใช้งาน](#เข้าใช้งาน)
    - [ตัวอย่าง Annotation](#ตัวอย่าง-annotation)
  - [Endpoints ที่มี](#endpoints-ที่มี)
  - [Error Response + TraceId](#error-response--traceid)
  - [เครื่องมือเปรียบเทียบ](#เครื่องมือเปรียบเทียบ)

---

## Swagger คืออะไร

Swagger (OpenAPI Specification) คือมาตรฐานในการอธิบาย REST API
โดยสร้าง documentation อัตโนมัติจาก code annotation
พร้อม UI สำหรับทดสอบ API ได้จาก browser ทันที

---

## ข้อดีหลัก

| ข้อดี | รายละเอียด |
|---|---|
| **Documentation อัตโนมัติ** | Generate จาก code comment — doc ไม่ out-of-date |
| **Try it out** | ทดสอบ API จาก browser เลย ไม่ต้องเปิด Postman |
| **Contract ชัดเจน** | Request/Response schema เห็นชัด ลด miscommunication |
| **Code Generation** | สร้าง client SDK จาก spec ได้ (TypeScript, Java, etc.) |
| **มาตรฐานสากล** | OpenAPI เป็นมาตรฐานที่ทุกคนในวงการรู้จัก |
| **ฟรี** | Open-source ไม่มีค่าใช้จ่าย |

---

## เหมาะกับใคร

### Developer

- เขียน `@Param`, `@Success`, `@Failure` ใน code — doc สร้างเอง
- Frontend dev ดู schema แล้วเริ่มทำงานคู่ขนานได้เลย ไม่ต้องรอ API เสร็จ
- ลด meeting "API นี้ส่งอะไรมา / ต้องส่งอะไรไป"
- ช่วย onboard dev ใหม่ได้เร็ว — ดู Swagger UI ก็เข้าใจ API ทั้งระบบ

### QA

- ทดสอบ API ได้จาก Swagger UI — กรอก parameter แล้ว Send
- เห็น schema ชัด — เขียน test case ได้ถูกต้อง
- Validate response ง่าย เพราะมี expected model อยู่แล้ว
- ไม่ต้อง setup tool เพิ่ม — ใช้ browser ได้เลย

### Business / PM

- เห็น API ทั้งหมดในระบบ แบบ non-technical friendly
- ใช้เป็น reference ตอนคุย requirement ใหม่
- ลดเวลา onboard คนใหม่ — เข้าใจ capability ของระบบได้เร็ว

---

## การใช้งาน

### คำสั่ง

```powershell
# Generate Swagger docs จาก code annotations
.\run.ps1 swagger

# รัน dev server แล้วเปิด Swagger UI
.\run.ps1 dev
```

### เข้าใช้งาน

```
http://localhost:20000/swagger/index.html
```

> **หมายเหตุ:** ต้องรัน server ก่อน (`.\run.ps1 dev`) แล้วค่อยเปิด Swagger UI

### ตัวอย่าง Annotation

```go
// GetPolicyByJobID godoc
// @Summary      Get CMI policy by job ID
// @Description  ดึงข้อมูลงาน พรบ. เดี่ยว ตาม job_id
// @Tags         CMI
// @Accept       json
// @Produce      json
// @Param        job_id path string true "Job ID"
// @Success      200 {object} dto.ApiResponse "CMI policy data"
// @Failure      400 {object} dto.ErrorResponse "trace_id: cmi-job-id-required"
// @Failure      404 {object} dto.ErrorResponse "trace_id: cmi-job-not-found"
// @Failure      500 {object} dto.ErrorResponse "trace_id: cmi-internal-error"
// @Router       /cmi/{job_id}/request-policy-single-cmi [get]
```

> **v2.1 เปลี่ยนแปลง:** `@Failure` เปลี่ยนจาก `dto.ApiResponse` → `dto.ErrorResponse` เพื่อแสดง `trace_id` ใน Swagger UI

---

## Endpoints ที่มี

| Group | Endpoints | Description |
|---|---|---|
| Auth | `POST /auth/login` | Authentication |
| ExternalDB | `GET /external-db/health`, `GET /external-db/health/{name}` | Database health check |
| Quotation | `GET /quotations/{id}`, `GET /quotations` | ใบเสนอราคา |
| CMI | `GET /cmi/{job_id}/request-policy-single-cmi` | งาน พรบ. เดี่ยว |
| Webhook | `POST /webhook/github` | GitHub webhook receiver |

---

## Error Response + TraceId

ตั้งแต่ v1.1.0 ทุก error response จะมี `trace_id` ใน `result` เพื่อระบุจุดที่เกิด error:

```json
{
  "status": "ERROR",
  "status_code": 404,
  "message": "quotation not found",
  "result": {
    "trace_id": "qt-not-found"
  }
}
```

### Response Structs

| Struct | ใช้เมื่อ | ไฟล์ |
|---|---|---|
| `dto.ApiResponse` | Success response | `internal/shared/dto/response.go` |
| `dto.ErrorResponse` | Error response (มี trace_id) | `internal/shared/dto/response.go` |
| `dto.ErrorResult` | trace_id object | `internal/shared/dto/response.go` |

### Error Code Catalog

Trace ID ทั้งหมดอยู่ใน `internal/shared/dto/error_codes.go`:

| Module | Trace ID | Code | คำอธิบาย |
|---|---|---|---|
| Auth | `auth-bind-failed` | 10001 | request body ไม่ถูกต้อง |
| Auth | `auth-invalid-creds` | 10002 | username/password ไม่ถูกต้อง |
| Auth | `auth-internal-error` | 10003 | เกิดข้อผิดพลาดภายใน |
| Auth | `auth-token-missing` | 10004 | ไม่มี Authorization header หรือ token ว่าง |
| Auth | `auth-token-invalid` | 10005 | token ไม่ถูกต้องหรือหมดอายุ |
| Auth | `auth-apikey-missing` | 10006 | ไม่มี X-API-Key header |
| Auth | `auth-apikey-invalid` | 10007 | API key ไม่ถูกต้อง |
| Quotation | `qt-id-required` | 11001 | ไม่ได้ส่ง quotation id |
| Quotation | `qt-not-found` | 11002 | ไม่พบ quotation |
| Quotation | `qt-internal-error` | 11003 | เกิดข้อผิดพลาดภายใน |
| Quotation | `qt-customer-id-required` | 11004 | ไม่ได้ส่ง customerId |
| Quotation | `qt-list-internal-error` | 11005 | เกิดข้อผิดพลาดขณะดึงรายการ |
| CMI | `cmi-job-id-required` | 12001 | ไม่ได้ส่ง job_id |
| CMI | `cmi-job-not-found` | 12002 | ไม่พบ job |
| CMI | `cmi-internal-error` | 12003 | เกิดข้อผิดพลาดภายใน |
| ExternalDB | `extdb-name-required` | 13001 | ไม่ได้ส่ง database name |
| ExternalDB | `extdb-not-found` | 13002 | ไม่พบ database |
| ExternalDB | `extdb-unhealthy` | 13003 | database ไม่สามารถเชื่อมต่อได้ |
| Webhook | `wh-invalid-signature` | 14001 | GitHub signature ไม่ถูกต้อง |
| Webhook | `wh-process-failed` | 14002 | ประมวลผล webhook ล้มเหลว |

---

## เครื่องมือเปรียบเทียบ

| Tool | ลักษณะ | จุดเด่น | เหมาะกับ |
|---|---|---|---|
| **Swagger / OpenAPI** | Annotation-based | ฟรี, มาตรฐาน, Try it out | โปรเจคนี้ |
| **Postman** | Collection-based | Collaboration, Testing | QA / Manual testing |
| **Redoc** | Render OpenAPI spec | UI สวยกว่า Swagger UI | Public API docs |
| **Stoplight** | Visual API designer | Design-first, mock ได้ | API Design phase |
| **Insomnia** | REST client | เบา, minimal UI | Dev ที่ชอบ simple tool |

---

> **v2.2** — March 2026 | ANC Portal Backend Team
