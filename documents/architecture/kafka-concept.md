# Kafka Concept — Event-Driven Architecture

## ภาพรวม

ระบบใช้ Kafka เป็น message broker สำหรับ **event-driven communication** ระหว่าง API Server กับ Worker Process

| Component | หน้าที่ | Entry Point |
|-----------|---------|-------------|
| **Producer** | ส่ง event ไปยัง Kafka topic | `cmd/api/main.go` |
| **Consumer** | รับ event จาก Kafka topic + retry + DLQ | `cmd/worker/main.go` |
| **Router** | dispatch message ไปหา handler ตาม event type | Worker process |

---

## ทำไมต้อง Kafka — ประโยชน์หลัก

| ประโยชน์ | อธิบาย |
|----------|--------|
| **Decoupling** | Producer ไม่ต้องรู้จัก Consumer — เพิ่ม/ลด service ได้อิสระ |
| **Async Processing** | API ตอบ client ทันที (202) แล้วให้ Worker ทำงานหนักทีหลัง |
| **Buffering** | Kafka เก็บ message ไว้ใน topic — ถ้า Consumer ล่ม message ไม่หาย |
| **Scalability** | เพิ่ม Consumer instance ได้ง่าย (consumer group แชร์ partition) |
| **Replay** | อ่าน message ซ้ำได้ — reset offset แล้ว reprocess ใหม่ |
| **Ordering** | message ใน partition เดียวกันเรียงตามลำดับ |
| **Throughput** | รองรับ message หลักแสน–ล้าน msg/sec |

---

## เหมาะกับงานแบบไหน

### ✅ เหมาะ

| Use Case | ตัวอย่างในโปรเจกต์ |
|----------|-------------------|
| **Event-driven** — แจ้งเตือนเมื่อเกิดเหตุการณ์ | `order.created` → ส่ง notification |
| **Background job** — งานที่ใช้เวลานาน | สร้าง PDF, sync ข้อมูล, ส่ง email |
| **Audit log** — บันทึกทุก action | เก็บ event ทั้งหมดใน topic ไว้ตรวจสอบ |
| **Cross-service communication** — คุยข้ามระบบ | API server → Worker process |
| **Data pipeline** — ส่งข้อมูลต่อเป็นทอด ๆ | ETL, sync ไป data warehouse |
| **Rate limiting / Load leveling** — คุม traffic | Kafka เป็น buffer กันระบบปลายทางล่ม |

### ❌ ไม่เหมาะ

| Use Case | ใช้อะไรแทน |
|----------|-----------|
| **Request-response แบบ sync** — ต้องการคำตอบทันที | HTTP / gRPC |
| **ข้อมูลน้อยมาก + latency ต่ำมาก** | direct call / in-memory |
| **Simple CRUD** — ไม่มี side effect | DB ตรง ๆ ไม่ต้องผ่าน Kafka |
| **Exactly-once ที่ strict มาก** | ต้องออกแบบ idempotency เพิ่มเอง |

---

## โครงสร้างไฟล์

```
pkg/kafka/
├── producer.go       # Producer — ส่ง message พร้อม tracing + retry
├── consumer.go       # Consumer — รับ message + DLQ support
├── router.go         # Router — dispatch event type → handler
├── message.go        # Message envelope (Type, Key, Payload, Metadata)
├── tracing.go        # W3C Trace Context propagation ผ่าน Kafka headers
├── message_test.go   # Unit tests: message, router
└── tracing_test.go   # Unit tests: trace inject/extract roundtrip
```

---

## แนวคิดหลัก

### 1. Message Envelope — โครงสร้างข้อมูลกลาง

```go
type Message struct {
    Type       string              // event type เช่น "order.created", "debug.message"
    Key        string              // key สำหรับ partitioning (optional)
    Payload    json.RawMessage     // event-specific data (JSON)
    Metadata   map[string]string   // metadata เพิ่มเติม
    OccurredAt time.Time           // เวลาที่ event เกิดขึ้น (UTC)
}
```

**Validation Rules:**
- `Type` — ต้องไม่ว่าง
- `Payload` — ต้องมี JSON data
- `OccurredAt` — ต้องไม่เป็น zero time

**ใช้งาน:**
```go
msg, err := kafka.NewMessage("order.created", "order-123", payload, metadata)
```

---

### 2. Producer — ฝั่งส่ง

```go
type Producer struct {
    writer *kafka.Writer  // segmentio/kafka-go
}
```

| คุณสมบัติ | ค่า | เหตุผล |
|-----------|-----|--------|
| **ACK Policy** | `RequireAll` | รอ replica ทั้งหมดยืนยัน — ไม่สูญหาย |
| **Write Mode** | Synchronous | blocking write — รู้ผลทันที |
| **Balancer** | `LeastBytes` | กระจาย message ตาม partition size |
| **Retry** | 3 attempts + constant 1s backoff | ใช้ `retry.Do()` |
| **Tracing** | W3C TraceContext inject | ส่ง trace parent ผ่าน Kafka header |

**Flow:**
```
PublishMessage(ctx, msg)
    → injectTraceHeaders()    // ฝัง trace context ใน header
    → retry.Do(3 attempts)    // retry ถ้า broker ไม่ตอบ
    → writer.WriteMessages()  // ส่งไป Kafka broker
```

---

### 3. Consumer — ฝั่งรับ

```go
type Consumer struct {
    reader  *kafka.Reader     // Kafka reader (consumer group)
    dlq     *Producer         // DLQ producer (optional)
    cfg     ConsumerConfig
    healthy atomic.Bool       // health probe flag
}
```

| คุณสมบัติ | ค่า | เหตุผล |
|-----------|-----|--------|
| **Consumer Group** | configurable `GroupID` | หลาย instance แชร์ partition |
| **Auto Commit** | 1s interval | reader จัดการ offset |
| **MaxBytes** | 10 MB (default) | จำกัดขนาด fetch |
| **Retry** | exponential backoff (configurable) | processWithRetry() |
| **DLQ** | ส่ง message ที่ fail เกิน max retry ไป DLQ topic | ไม่ block consumer |
| **Health Probe** | `IsHealthy()` | true หลัง fetch สำเร็จครั้งแรก |

**Flow:**
```
StartMessages(ctx, handler)         // polling loop
    → FetchMessage()                 // ดึง message จาก broker
    → DecodeMessage()                // parse JSON envelope
    → extractTraceContext()          // restore parent span จาก producer
    → processWithRetry(handler)      // execute handler + retry
        → Success: CommitMessages()  // mark offset เป็น consumed
        → Failure: sendToDLQ()       // ส่งไป dead letter queue
```

---

### 4. Router — dispatch event type

```go
type Router struct {
    handlers map[string]Handler    // event_type → handler function
    fallback Handler               // handler สำหรับ event ที่ไม่มี handler
}

type Handler func(ctx context.Context, msg Message) error
```

**ใช้งาน:**
```go
router := kafka.NewRouter()

// ลงทะเบียน handler ตาม event type
router.Register("order.created", handleOrderCreated)
router.Register("payment.completed", handlePaymentCompleted)

// fallback สำหรับ event ที่ไม่รู้จัก
router.SetFallback(func(ctx context.Context, msg kafka.Message) error {
    log.Warn().Str("event_type", msg.Type).Msg("unhandled event — skipped")
    return nil
})

// Consumer ใช้ router เป็น handler
consumer.StartMessages(ctx, func(ctx context.Context, msg kafka.Message) error {
    return router.Dispatch(ctx, msg)
})
```

**กฎ:**
- ✅ Register handler 1 ตัวต่อ 1 event type (ป้องกัน duplicate)
- ✅ ใช้ fallback สำหรับ event ที่ยังไม่มี handler
- ❌ ห้าม register event type ซ้ำ — return error

---

### 5. Dead Letter Queue (DLQ)

เมื่อ message fail เกิน `MaxRetries` → ส่งไป DLQ topic อัตโนมัติ

```
Message fail
    → retry 1 (backoff)
    → retry 2 (backoff)
    → retry 3 (backoff)
    → MAX_RETRIES reached
    → sendToDLQ()
```

**DLQ Payload:**
```go
type dlqPayload struct {
    OriginalMessage string    // raw message ที่ fail
    Error           string    // error message
    Retries         int       // จำนวน retry ที่ทำ
    FailedAt        time.Time // เวลาที่ fail
}
```

**DLQ Message:**
- Type: `"dlq." + original_type` (เช่น `dlq.order.created`)
- Headers: `source_topic`, `error`

**ประโยชน์:**
- ✅ Consumer ไม่ถูก block จาก message ที่ fail ซ้ำ
- ✅ สามารถ monitor DLQ topic แยกต่างหาก
- ✅ replay failed messages ได้ภายหลัง

---

### 6. W3C Trace Context Propagation

ระบบส่ง **trace context** ผ่าน Kafka headers เพื่อเชื่อม trace ระหว่าง producer ↔ consumer

```
API Server (Producer)                     Worker (Consumer)
┌──────────────────────┐                  ┌──────────────────────┐
│ HTTP Request         │                  │ Kafka Message        │
│   └─ Publish span    │                  │   └─ Process span    │
│       └─ inject ─────┼── traceparent ──►│       └─ extract     │
│         headers      │   Kafka header   │         parent span  │
└──────────────────────┘                  └──────────────────────┘
         Same Trace ID linked across processes
```

- Producer: `injectTraceHeaders(ctx)` → W3C `traceparent` header
- Consumer: `extractTraceContext(ctx, headers)` → restore parent span
- ผลลัพธ์: ดู trace เดียวกันใน Grafana/Tempo ได้ตั้งแต่ HTTP request → Kafka → handler

---

## Feature Toggle

Kafka เปิด/ปิดได้ผ่าน `KAFKA_ENABLED` — ไม่ crash ถ้าปิด

| `KAFKA_ENABLED` | API Server | Worker |
|-----------------|------------|--------|
| `true` | สร้าง Producer, ใช้งานได้ | เริ่ม Consumer loop |
| `false` | `producer = nil`, ข้าม publish | log แล้ว exit ทันที |

---

## Environment Configuration

| Variable | ค่าตัวอย่าง | หมายเหตุ |
|----------|-------------|----------|
| `KAFKA_ENABLED` | `true` | เปิด/ปิด Kafka |
| `KAFKA_BROKERS` | `localhost:9092` | comma-separated |
| `KAFKA_TOPIC` | `anc-portal-events` | main event topic |
| `KAFKA_GROUP_ID` | `anc-portal-worker` | consumer group |
| `KAFKA_DLQ_TOPIC` | `anc-portal-events-dlq` | dead letter topic (optional) |
| `KAFKA_WRITE_TIMEOUT` | `10s` | producer timeout |
| `KAFKA_MAX_BYTES` | `10485760` | 10 MB — consumer fetch limit |
| `KAFKA_MAX_RETRIES` | `3` | retry ก่อนส่ง DLQ |

**ตัวอย่าง .env:**
```env
KAFKA_ENABLED=true
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=anc-portal-events
KAFKA_GROUP_ID=anc-portal-worker
KAFKA_DLQ_TOPIC=anc-portal-events-dlq
KAFKA_WRITE_TIMEOUT=10s
KAFKA_MAX_BYTES=10485760
KAFKA_MAX_RETRIES=3
```

---

## Data Flow — End-to-End

```
┌─────────────────────────────────────────────────────────────────┐
│                        API Server (cmd/api)                     │
│                                                                 │
│  HTTP Request → Handler → kafka.NewMessage() → producer.Publish │
│                                    │                            │
│                            inject trace headers                 │
└────────────────────────────────────┼────────────────────────────┘
                                     │
                              Kafka Broker
                            ┌────────┴────────┐
                            │  anc-portal-     │
                            │  events (topic)  │
                            └────────┬────────┘
                                     │
┌────────────────────────────────────┼────────────────────────────┐
│                     Worker Process (cmd/worker)                  │
│                                    │                            │
│  consumer.StartMessages() → DecodeMessage() → extract trace     │
│                                    │                            │
│                          router.Dispatch(msg)                   │
│                          ┌─────────┼─────────┐                  │
│                    handler A    handler B    fallback            │
│                          │                                      │
│              ┌───────────┴───────────┐                          │
│              │ Success → Commit      │                          │
│              │ Failure → Retry → DLQ │                          │
│              └───────────────────────┘                          │
└─────────────────────────────────────────────────────────────────┘
                                     │
                              (if DLQ enabled)
                            ┌────────┴────────┐
                            │  anc-portal-     │
                            │  events-dlq      │
                            └─────────────────┘
```

---

## กฎการใช้งาน

### ✅ ควรทำ
- ใช้ `Message` envelope เสมอ — ไม่ส่ง raw bytes
- ตั้ง `event_type` ให้ชัดเจน เช่น `domain.action` (e.g. `order.created`)
- ลงทะเบียน handler ผ่าน `Router` — ไม่ handle ใน consumer โดยตรง
- ตั้ง fallback handler เพื่อ log event ที่ยังไม่มี handler
- เปิด DLQ สำหรับ production — ป้องกัน message สูญหาย
- ใช้ `KAFKA_ENABLED=false` ใน local dev ถ้าไม่ต้องการ Kafka

### ❌ ไม่ควรทำ
- ส่ง message โดยไม่ validate (`NewMessage` จัดการให้)
- Register handler ซ้ำ event type — จะ error
- ใช้ DLQ topic เดียวกับ main topic
- ปิด retry ใน production — ควรมี retry อย่างน้อย 3 ครั้ง
- ส่ง message ที่ใหญ่มาก — ควรส่งแค่ reference (ID) แล้วให้ consumer ดึงข้อมูลเอง

---

## Test Endpoint (Local Only)

เมื่อ `STAGE_STATUS=local` และ Kafka enabled → มี endpoint สำหรับทดสอบ:

```http
POST /v1/kafka/publish
Content-Type: application/json

{
  "key": "user-123",
  "eventType": "order.created",
  "message": "Order #456 created",
  "metadata": { "source": "api" }
}
```

Response: `202 Accepted`

---

## Worker Health Probe

Worker process เปิด HTTP server บน port `:20001` สำหรับ Kubernetes:

| Endpoint | ตรวจสอบ | ใช้กับ |
|----------|---------|--------|
| `/healthz` | `consumer.IsHealthy()` | liveness + readiness probe |

- `IsHealthy() = true` → consumer fetch สำเร็จอย่างน้อย 1 ครั้ง
- `IsHealthy() = false` → ยังไม่เคย fetch สำเร็จ หรือ disconnect

---

## Technology Stack

| Component | Library | Version |
|-----------|---------|---------|
| Kafka Client | `github.com/segmentio/kafka-go` | v0.4.50 |
| Retry | `pkg/retry` (internal) | — |
| Tracing | `go.opentelemetry.io/otel` | W3C TraceContext |
| Logging | `github.com/rs/zerolog` | — |
