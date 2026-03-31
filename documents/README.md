# Documents — สารบัญรวม

> เอกสารทั้งหมดของโปรเจกต์ ANC Portal Backend จัดเรียงตามหมวดหมู่

---

## Architecture — โครงสร้างระบบ

| เอกสาร | เนื้อหา |
|--------|---------|
| [README](architecture/README.md) | ภาพรวมสถาปัตยกรรม (Modular Monolith + Hexagonal) |
| [Project Structure](architecture/project-structure.md) | โครงสร้าง folder ทั้งโปรเจกต์ |
| [Auth Structure](architecture/auth-structure.md) | ระบบ Authentication & Authorization |
| [Database Concept](architecture/database-concept.md) | Database Layer (Multi-Driver: Postgres + MySQL) |
| [Kafka Concept](architecture/kafka-concept.md) | Event-Driven Architecture ด้วย Kafka |
| [Swagger Concept](architecture/swagger-concept.md) | API Documentation ด้วย Swagger/OpenAPI |
| [Microservice Readiness](architecture/microservice-readiness.md) | แนวทางแตก module เมื่อพร้อมเป็น microservice |

## CI/CD — Continuous Integration & Deployment

| เอกสาร | เนื้อหา |
|--------|---------|
| [CI/CD Guide](cicd/ci-cd-guide.md) | คู่มือหลัก — pipeline, linter, ทั้ง local และ GitHub Actions |
| [CI Pipeline Stages](cicd/ci-pipeline-stages.md) | รายละเอียด 7 stages ของ CI pipeline |
| [Workflow Concept](cicd/workflow-concept.md) | อธิบาย push → CI → deploy staging → tag → production |
| [GitHub Actions Setup](cicd/github-actions-setup.md) | วิธีตั้งค่า GitHub Actions + Secrets |
| [Dependabot Concept](cicd/dependabot-concept.md) | อธิบาย Dependabot คืออะไร + config |

## Observability — Monitoring & Tracing

| เอกสาร | เนื้อหา |
|--------|---------|
| [OTel Tracing Guide](observability/otel-tracing-guide.md) | วิธีใช้ OpenTelemetry tracing ในโปรเจกต์ |
| [OTel + Grafana Quickstart](observability/otel-grafana-quickstart.md) | ตั้ง Grafana + Tempo + Prometheus + OTel Collector |

## Infrastructure — Deployment & Services

| เอกสาร | เนื้อหา |
|--------|---------|
| [Deployment Guide](infrastructure/deployment-guide.md) | ขั้นตอน deploy ทั้ง Docker + Kubernetes |
| [Kubernetes Guide](infrastructure/kubernetes-guide.md) | Kubernetes concepts + manifests |
| [Resource Spec Guide](infrastructure/resource-spec-guide.md) | CPU/Memory requests & limits |
| [Redis Cache Guide](infrastructure/redis-cache-guide.md) | การใช้ Redis + Hybrid L1→L2 cache |
| [Discord Notification](infrastructure/discord-notification.md) | ตั้ง Discord Webhook สำหรับ CI/CD notification |

## Testing — การทดสอบ

| เอกสาร | เนื้อหา |
|--------|---------|
| [Unit Test Guide](testing/unit-test-guide.md) | คู่มือฉบับเต็ม — patterns, conventions, ตัวอย่าง |
| [Unit Test Cheatsheet](testing/unit-test-cheatsheet.md) | สรุปย่อ — patterns table, code examples, commands |
| [Code Coverage Concept](testing/code-coverage-concept.md) | อธิบาย coverage คืออะไร + วิธีวัด |

## Operations — Runbooks & Standards

| เอกสาร | เนื้อหา |
|--------|---------|
| [Incident Response Runbook](operations/incident-response-runbook.md) | ขั้นตอนจัดการ incident (SEV1-4) |
| [ISO Standards Checklist](operations/iso-standards-checklist.md) | Checklist มาตรฐาน ISO 27001 / 9001 |
