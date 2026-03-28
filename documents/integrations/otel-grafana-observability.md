# OpenTelemetry + Grafana — Quick Start

> **v2.0** — Last updated: March 2026
>
> คู่มือ Quick Start สำหรับเปิดใช้ Observability stack ใน local dev
>
> ดูรายละเอียดเพิ่มเดิม: [OTel Tracing Guide (ฉบับเต็ม)](otel-tracing-guide.md)

---

## สถาปัตยกรรม

```
Go App ──OTLP/HTTP──▶ OTel Collector ──▶ Tempo (traces)
  │                        │
  │                        └──▶ Prometheus (metrics via remote write)
  │
  └──/metrics──▶ Prometheus (scrape)
                       │
                       └──▶ Grafana (dashboards)
```

| Component | Port | หน้าที่ |
|---|---|---|
| Go App | 20000 | ส่ง traces/metrics ผ่าน OTLP/HTTP |
| OTel Collector | 4318 | รับ OTLP/HTTP → forward ไป Tempo/Prometheus |
| Grafana Tempo | 3200 | เก็บ traces |
| Prometheus | 9090 | เก็บ metrics |
| Grafana | 3001 | Dashboard + Explore |

---

## Quick Start (4 ขั้นตอน)

### 1. เปิด Observability Stack

```bash
cd deployments/observability
docker compose up -d
```

### 2. เปิด OTel ใน .env.local

```env
OTEL_ENABLED=true
OTEL_SERVICE_NAME=anc-portal-dev
OTEL_EXPORTER_URL=localhost:4318
OTEL_SAMPLE_RATIO=1.0
```

### 3. รัน Go App

```powershell
.\run.ps1 dev
```

### 4. ดู Traces & Metrics

| เครื่องมือ | URL | วิธีใช้ |
|---|---|---|
| **Grafana** | http://localhost:3001 | Explore → Tempo → search by service name |
| **Prometheus** | http://localhost:9090 | Query metrics |
| **Raw metrics** | http://localhost:20000/metrics | Prometheus scrape endpoint |

---

## สิ่งที่ถูก Trace อัตโนมัติ

| Layer | Library | สิ่งที่บันทึก |
|---|---|---|
| HTTP Request | pkg/otel | method, route, status_code, duration |
| PostgreSQL Query | otelpgx | SQL query, duration, errors |
| Redis Command | redisotel | command, key, duration |
| Kafka Publish/Consume | pkg/kafka | topic, partition, W3C propagation |
| Outgoing HTTP | pkg/httpclient | method, URL, status, duration |

---

## Environment Variables

| Variable | Default | คำอธิบาย |
|---|---|---|
| `OTEL_ENABLED` | `false` | เปิด/ปิด OTel ทั้งระบบ |
| `OTEL_SERVICE_NAME` | `anc-portal-dev` | ชื่อ service ใน traces |
| `OTEL_EXPORTER_URL` | `localhost:4318` | OTel Collector endpoint |
| `OTEL_SAMPLE_RATIO` | `1.0` | อัตรา sampling (0.0 – 1.0) |

---

## ไฟล์ที่เกี่ยวข้อง

```
pkg/otel/
├── otel.go              ← Bootstrap TracerProvider + MeterProvider
├── middleware.go         ← Fiber HTTP tracing middleware
└── tracername.go         ← Central Tracer Name Registry

deployments/observability/
├── docker-compose.yaml  ← Observability stack
├── otel-collector.yaml  ← OTel Collector config
├── prometheus.yaml      ← Prometheus scrape config
├── tempo.yaml           ← Tempo storage config
└── grafana/provisioning/
    └── datasources/     ← Auto-provision Tempo + Prometheus
```

---

> **v2.0** — March 2026 | ANC Portal Backend Team

---

## เปิด/ปิด OTel

เมื่อ `OTEL_ENABLED=false`:
- TracerProvider + MeterProvider ใช้ noop (ไม่มี overhead)
- `/metrics` endpoint จะไม่ถูก register
- Fiber middleware จะไม่ถูก mount
- Redis tracing hooks จะไม่ถูกเพิ่ม

**ไม่มีผลกระทบต่อ application logic** — ทุกอย่างทำงานปกติโดยไม่ต้องมี OTel Collector

---

## Production Tips

1. **Sample Ratio**: ลดเป็น `0.1` – `0.5` ใน production เพื่อลด overhead
2. **TLS**: เปลี่ยน `WithInsecure()` เป็น TLS config ที่เหมาะสม
3. **Collector**: Deploy OTel Collector แยกเป็น sidecar หรือ DaemonSet ใน K8s
4. **Retention**: ตั้ง retention policy ใน Tempo + Prometheus ตามความเหมาะสม
