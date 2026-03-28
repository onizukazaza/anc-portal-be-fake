# Swagger (OpenAPI) — คู่มือการใช้งาน

> **v2.0** — Last updated: March 2026
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
// @Failure      400 {object} dto.ApiResponse "Job ID is required"
// @Failure      404 {object} dto.ApiResponse "Job not found"
// @Failure      500 {object} dto.ApiResponse "Internal error"
// @Router       /cmi/{job_id}/request-policy-single-cmi [get]
```

---

## Endpoints ที่มี

| Group | Endpoints | Description |
|---|---|---|
| Auth | `POST /auth/login` | Authentication |
| ExternalDB | `GET /external-db/health`, `GET /external-db/health/{name}` | Database health check |
| Quotation | `GET /quotations/{id}`, `GET /quotations` | ใบเสนอราคา |
| CMI | `GET /cmi/{job_id}/request-policy-single-cmi` | งาน พรบ. เดี่ยว |

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

> **v2.0** — March 2026 | ANC Portal Backend Team
