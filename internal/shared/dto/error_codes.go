package dto

// ===================================================================
// Error Code Catalog — Single Source of Truth
// ===================================================================
//
// ไฟล์นี้เป็น "แหล่งข้อมูลเดียว" ของ error trace_id ทั้งโปรเจกต์
// Swagger @description ใน cmd/api/main.go อ้างอิงตารางจากที่นี่
// Handler annotation อ้างอิง constant + คำอธิบายจากที่นี่
//
// ─────────────────────────────────────────────────────────────────
// Module Prefix Mapping (สำหรับ running number)
// ─────────────────────────────────────────────────────────────────
//
//   AUTH  = 10xxx    QT    = 11xxx    CMI   = 12xxx
//   EXTDB = 13xxx    WH    = 14xxx    PAY   = 15xxx (reserved)
//   POL   = 16xxx    DOC   = 17xxx    NOTIF = 18xxx
//   JOB   = 19xxx
//
// ─────────────────────────────────────────────────────────────────
// Error Code Table (ใช้อ้างอิงใน Swagger)
// ─────────────────────────────────────────────────────────────────
//
//  Code  | Trace ID               | HTTP | คำอธิบาย
//  ──────|────────────────────────|──────|──────────────────────────
//  10001 | auth-bind-failed       | 400  | request body ไม่ถูกต้อง
//  10002 | auth-invalid-creds     | 401  | username/password ไม่ถูกต้อง
//  10003 | auth-internal-error    | 500  | เกิดข้อผิดพลาดภายใน auth service
//  10004 | auth-token-missing     | 401  | ไม่มี Authorization header หรือ token ว่าง
//  10005 | auth-token-invalid     | 401  | token ไม่ถูกต้องหรือหมดอายุ
//  10006 | auth-apikey-missing    | 401  | ไม่มี X-API-Key header
//  10007 | auth-apikey-invalid    | 401  | API key ไม่ถูกต้อง
//  ──────|────────────────────────|──────|──────────────────────────
//  11001 | qt-id-required         | 400  | ไม่ได้ส่ง quotation id
//  11002 | qt-not-found           | 404  | ไม่พบ quotation
//  11003 | qt-internal-error      | 500  | เกิดข้อผิดพลาดภายใน quotation service
//  11004 | qt-customer-id-required| 400  | ไม่ได้ส่ง customerId
//  11005 | qt-list-internal-error | 500  | เกิดข้อผิดพลาดขณะดึงรายการ quotation
//  ──────|────────────────────────|──────|──────────────────────────
//  12001 | cmi-job-id-required    | 400  | ไม่ได้ส่ง job_id
//  12002 | cmi-job-not-found      | 404  | ไม่พบ job
//  12003 | cmi-internal-error     | 500  | เกิดข้อผิดพลาดภายใน CMI service
//  ──────|────────────────────────|──────|──────────────────────────
//  13001 | extdb-name-required    | 400  | ไม่ได้ส่ง database name
//  13002 | extdb-not-found        | 404  | ไม่พบ database ที่ระบุ
//  13003 | extdb-unhealthy        | 503  | database ไม่สามารถเชื่อมต่อได้
//  ──────|────────────────────────|──────|──────────────────────────
//  14001 | wh-invalid-signature   | 401  | GitHub signature ไม่ถูกต้อง
//  14002 | wh-process-failed      | 500  | ประมวลผล webhook ล้มเหลว
//
// ─────────────────────────────────────────────────────────────────
// วิธีเพิ่ม error code ใหม่:
//   1. เพิ่ม const ใน section ของ module (ใช้ running number ถัดไป)
//   2. อัปเดตตาราง Error Code Table ด้านบน
//   3. อัปเดต @description ใน cmd/api/main.go (Swagger catalog)
//   4. รัน: swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
// ─────────────────────────────────────────────────────────────────

// ─── Auth (10xxx) ────────────────────────────────────────────────

const (
	TraceAuthBindFailed    = "auth-bind-failed"    // 10001 | 400 | request body ไม่ถูกต้อง
	TraceAuthBadLogin      = "auth-invalid-creds"  // 10002 | 401 | username/password ไม่ถูกต้อง
	TraceAuthInternalError = "auth-internal-error" // 10003 | 500 | เกิดข้อผิดพลาดภายใน auth service
	TraceAuthNoHeader      = "auth-token-missing"  // 10004 | 401 | ไม่มี Authorization header หรือ token ว่าง
	TraceAuthVerifyFailed  = "auth-token-invalid"  // 10005 | 401 | token ไม่ถูกต้องหรือหมดอายุ
	TraceAuthAPIKeyMissing = "auth-apikey-missing" //nolint:gosec // G101: trace ID label, not a credential
	TraceAuthAPIKeyInvalid = "auth-apikey-invalid" //nolint:gosec // G101: trace ID label, not a credential
)

// ─── Quotation (11xxx) ───────────────────────────────────────────

const (
	TraceQTIdRequired        = "qt-id-required"          // 11001 | 400 | ไม่ได้ส่ง quotation id
	TraceQTNotFound          = "qt-not-found"            // 11002 | 404 | ไม่พบ quotation
	TraceQTInternalError     = "qt-internal-error"       // 11003 | 500 | เกิดข้อผิดพลาดภายใน quotation service
	TraceQTCustomerRequired  = "qt-customer-id-required" // 11004 | 400 | ไม่ได้ส่ง customerId
	TraceQTListInternalError = "qt-list-internal-error"  // 11005 | 500 | เกิดข้อผิดพลาดขณะดึงรายการ quotation
)

// ─── CMI (12xxx) ─────────────────────────────────────────────────

const (
	TraceCMIJobIdRequired = "cmi-job-id-required" // 12001 | 400 | ไม่ได้ส่ง job_id
	TraceCMIJobNotFound   = "cmi-job-not-found"   // 12002 | 404 | ไม่พบ job
	TraceCMIInternalError = "cmi-internal-error"  // 12003 | 500 | เกิดข้อผิดพลาดภายใน CMI service
)

// ─── ExternalDB (13xxx) ──────────────────────────────────────────

const (
	TraceExtDBNameRequired = "extdb-name-required" // 13001 | 400 | ไม่ได้ส่ง database name
	TraceExtDBNotFound     = "extdb-not-found"     // 13002 | 404 | ไม่พบ database ที่ระบุ
	TraceExtDBUnhealthy    = "extdb-unhealthy"     // 13003 | 503 | database ไม่สามารถเชื่อมต่อได้
)

// ─── Webhook (14xxx) ─────────────────────────────────────────────

const (
	TraceWHInvalidSignature = "wh-invalid-signature" // 14001 | 401 | GitHub signature ไม่ถูกต้อง
	TraceWHProcessFailed    = "wh-process-failed"    // 14002 | 500 | ประมวลผล webhook ล้มเหลว
)
