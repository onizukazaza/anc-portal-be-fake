package dto

// ===================================================================
// Error Code Catalog — รวม error code + trace id ทั้งโปรเจค
// ===================================================================
//
// รูปแบบ: {module_prefix}{running_number}
//
//   Module Prefix:
//     AUTH  = 10xxx — Authentication
//     QT    = 11xxx — Quotation
//     CMI   = 12xxx — CMI (Compulsory Motor Insurance)
//     EXTDB = 13xxx — External Database Health
//     WH    = 14xxx — Webhook
//     PAY   = 15xxx — Payment
//     POL   = 16xxx — Policy
//     DOC   = 17xxx — Document
//     NOTIF = 18xxx — Notification
//     JOB   = 19xxx — Job / Worker

// ─── Auth (10xxx) ────────────────────────────────────────────────

const (
	TraceAuthBindFailed    = "auth-bind-failed"    // 10001 — request body ไม่ถูกต้อง
	TraceAuthBadLogin      = "auth-invalid-creds"  // 10002 — username/password ไม่ถูกต้อง
	TraceAuthInternalError = "auth-internal-error" // 10003 — เกิดข้อผิดพลาดภายใน auth service
)

// ─── Quotation (11xxx) ───────────────────────────────────────────

const (
	TraceQTIdRequired        = "qt-id-required"          // 11001 — ไม่ได้ส่ง quotation id
	TraceQTNotFound          = "qt-not-found"            // 11002 — ไม่พบ quotation
	TraceQTInternalError     = "qt-internal-error"       // 11003 — เกิดข้อผิดพลาดภายใน quotation service
	TraceQTCustomerRequired  = "qt-customer-id-required" // 11004 — ไม่ได้ส่ง customerId
	TraceQTListInternalError = "qt-list-internal-error"  // 11005 — เกิดข้อผิดพลาดขณะดึงรายการ quotation
)

// ─── CMI (12xxx) ─────────────────────────────────────────────────

const (
	TraceCMIJobIdRequired = "cmi-job-id-required" // 12001 — ไม่ได้ส่ง job_id
	TraceCMIJobNotFound   = "cmi-job-not-found"   // 12002 — ไม่พบ job
	TraceCMIInternalError = "cmi-internal-error"  // 12003 — เกิดข้อผิดพลาดภายใน CMI service
)

// ─── ExternalDB (13xxx) ──────────────────────────────────────────

const (
	TraceExtDBNameRequired = "extdb-name-required" // 13001 — ไม่ได้ส่ง database name
	TraceExtDBNotFound     = "extdb-not-found"     // 13002 — ไม่พบ database ที่ระบุ
	TraceExtDBUnhealthy    = "extdb-unhealthy"     // 13003 — database ไม่สามารถเชื่อมต่อได้
)

// ─── Webhook (14xxx) ─────────────────────────────────────────────

const (
	TraceWHInvalidSignature = "wh-invalid-signature" // 14001 — GitHub signature ไม่ถูกต้อง
	TraceWHProcessFailed    = "wh-process-failed"    // 14002 — ประมวลผล webhook ล้มเหลว
)
