# ISO Standards Checklist — Software Development

> **v1.0** — Last updated: March 2026
>
> สิ่งที่ต้องมีเพื่อให้โปรเจกต์ซอฟต์แวร์เข้ามาตรฐาน ISO ที่เกี่ยวข้อง

---

## สารบัญ

- [ISO Standards Checklist — Software Development](#iso-standards-checklist--software-development)
  - [สารบัญ](#สารบัญ)
  - [1. ISO ที่เกี่ยวข้องกับ Software](#1-iso-ที่เกี่ยวข้องกับ-software)
  - [2. ISO 9001 — Quality Management](#2-iso-9001--quality-management)
    - [สิ่งที่ต้องมี](#สิ่งที่ต้องมี)
  - [3. ISO/IEC 27001 — Information Security](#3-isoiec-27001--information-security)
    - [สิ่งที่ต้องมี](#สิ่งที่ต้องมี-1)
  - [4. ISO/IEC 25010 — Software Quality Model](#4-isoiec-25010--software-quality-model)
    - [8 Quality Characteristics](#8-quality-characteristics)
  - [5. ISO/IEC 12207 — Software Lifecycle](#5-isoiec-12207--software-lifecycle)
    - [สิ่งที่ต้องมี](#สิ่งที่ต้องมี-2)
  - [6. Checklist รวม — เทียบกับโปรเจกต์ปัจจุบัน](#6-checklist-รวม--เทียบกับโปรเจกต์ปัจจุบัน)
    - [✅ สิ่งที่มีแล้ว](#-สิ่งที่มีแล้ว)
    - [❌ สิ่งที่ยังขาด](#-สิ่งที่ยังขาด)
  - [7. Roadmap แนะนำ](#7-roadmap-แนะนำ)
    - [Phase 1 — Quick Wins (ทำได้เลย)](#phase-1--quick-wins-ทำได้เลย)
    - [Phase 2 — Medium Effort (1-2 สัปดาห์)](#phase-2--medium-effort-1-2-สัปดาห์)
    - [Phase 3 — Long Term (1 เดือน+)](#phase-3--long-term-1-เดือน)
    - [เป้าหมายรวม](#เป้าหมายรวม)

---

## 1. ISO ที่เกี่ยวข้องกับ Software

| มาตรฐาน | ชื่อเต็ม | เน้นเรื่อง |
|----------|----------|------------|
| **ISO 9001** | Quality Management Systems | ระบบบริหารคุณภาพ — process, documentation, continuous improvement |
| **ISO/IEC 27001** | Information Security Management | ความปลอดภัยของข้อมูล — access control, encryption, incident response |
| **ISO/IEC 25010** | Software Quality Model | คุณภาพซอฟต์แวร์ — reliability, security, performance, maintainability |
| **ISO/IEC 12207** | Software Lifecycle Processes | กระบวนการพัฒนา — requirement, design, coding, testing, deployment |

---

## 2. ISO 9001 — Quality Management

> **หัวใจ:** "ทำในสิ่งที่เขียน เขียนในสิ่งที่ทำ" — ทุกอย่างต้องมี process และ documentation

### สิ่งที่ต้องมี

| # | หัวข้อ | รายละเอียด | ตัวอย่าง |
|---|--------|-----------|---------|
| 1 | **Document Control** | เอกสารทุกฉบับต้องมี version, วันที่, ผู้อนุมัติ | README, Architecture docs มี version header |
| 2 | **Process Documentation** | ขั้นตอนการทำงานต้องเขียนไว้ชัดเจน | CI/CD guide, deployment guide |
| 3 | **Change Management** | การเปลี่ยนแปลงต้องมีขั้นตอน review/approve | Git branching strategy, PR review process |
| 4 | **Quality Objectives** | ตั้งเป้าหมายคุณภาพที่วัดได้ | Coverage ≥ 70%, Zero critical bugs in production |
| 5 | **Internal Audit** | ตรวจสอบว่าทำตาม process หรือไม่ | Code review checklist, sprint retrospective |
| 6 | **Corrective Action** | เมื่อเจอปัญหา ต้องมี root cause analysis + แก้ไข | Post-mortem document, bug tracking |
| 7 | **Continuous Improvement** | มีกระบวนการปรับปรุงอย่างต่อเนื่อง | Retrospective → action items → track progress |
| 8 | **Training Records** | บันทึกการฝึกอบรมของทีม | Onboarding checklist, knowledge sharing log |
| 9 | **Customer Feedback** | รวบรวม feedback จากผู้ใช้งาน | Bug reports, feature requests, satisfaction survey |
| 10 | **Management Review** | ผู้บริหารทบทวนระบบคุณภาพ | Monthly quality report |

---

## 3. ISO/IEC 27001 — Information Security

> **หัวใจ:** ปกป้องข้อมูล 3 ด้าน — **Confidentiality** (ลับ), **Integrity** (ถูกต้อง), **Availability** (พร้อมใช้)

### สิ่งที่ต้องมี

| # | หัวข้อ | รายละเอียด | ตัวอย่าง |
|---|--------|-----------|---------|
| 1 | **Access Control** | ควบคุมสิทธิ์การเข้าถึงระบบและข้อมูล | RBAC, JWT authentication, API keys |
| 2 | **Data Encryption** | เข้ารหัสข้อมูลทั้ง at-rest และ in-transit | TLS/SSL, bcrypt password hash, DB encryption |
| 3 | **Secret Management** | ข้อมูลลับต้องไม่อยู่ใน source code | Environment variables, K8s secrets, vault |
| 4 | **Vulnerability Management** | ตรวจสอบช่องโหว่อย่างสม่ำเสมอ | `govulncheck`, dependency scanning |
| 5 | **Incident Response Plan** | แผนรับมือเมื่อเกิดเหตุด้านความปลอดภัย | Runbook: "เมื่อถูก hack ทำอะไรก่อน" |
| 6 | **Audit Logging** | บันทึกกิจกรรมสำคัญ | Access log, authentication log, change log |
| 7 | **Backup & Recovery** | มีระบบสำรองและกู้คืนข้อมูล | Database backup schedule, DR plan, RTO/RPO |
| 8 | **Network Security** | ป้องกันระดับ network | Firewall, rate limiting, CORS policy |
| 9 | **Security Awareness** | ทีมต้องรู้เรื่อง security | OWASP Top 10 training, secure coding guide |
| 10 | **Risk Assessment** | ประเมินความเสี่ยงและจัดลำดับ | Risk register (ระบุ threat → impact → mitigation) |
| 11 | **Asset Inventory** | รู้ว่ามี asset อะไรบ้าง ใครดูแล | Server list, database list, API list + owner |
| 12 | **Third-Party Security** | ตรวจสอบความปลอดภัยของ dependency | License audit, supply chain security |

---

## 4. ISO/IEC 25010 — Software Quality Model

> **หัวใจ:** คุณภาพซอฟต์แวร์ 8 ด้าน

### 8 Quality Characteristics

| # | คุณสมบัติ | คำอธิบาย | วิธีวัด/ทำ |
|---|----------|----------|-----------|
| 1 | **Functional Suitability** | ทำงานถูกต้องตาม requirement | Unit test, integration test, acceptance test |
| 2 | **Performance Efficiency** | ทำงานเร็ว ใช้ resource น้อย | Load test, benchmark, response time monitoring |
| 3 | **Compatibility** | ทำงานร่วมกับระบบอื่นได้ | API versioning, backward compatibility test |
| 4 | **Usability** | ใช้งานง่าย | API documentation (Swagger), clear error messages |
| 5 | **Reliability** | ทำงานได้อย่างต่อเนื่อง ไม่ล่ม | Health check, graceful shutdown, retry mechanism |
| 6 | **Security** | ป้องกันการเข้าถึงที่ไม่ได้รับอนุญาต | Auth, authorization, input validation, OWASP |
| 7 | **Maintainability** | แก้ไข/เพิ่มเติมง่าย | Clean architecture, code coverage, linting |
| 8 | **Portability** | ย้ายแพลตฟอร์มได้ง่าย | Docker, K8s, environment-based config |

---

## 5. ISO/IEC 12207 — Software Lifecycle

> **หัวใจ:** ครบทุกขั้นตอนตั้งแต่เริ่ม → ส่งมอบ → บำรุงรักษา

### สิ่งที่ต้องมี

| # | กระบวนการ | รายละเอียด | สิ่งที่ต้องทำ |
|---|----------|-----------|-------------|
| 1 | **Requirement** | เก็บ requirement ชัดเจน | Requirement document, user stories |
| 2 | **Design** | ออกแบบก่อน code | Architecture document, ERD, API design |
| 3 | **Implementation** | Coding ตาม standard | Coding convention, code review, linting |
| 4 | **Testing** | ทดสอบทุกระดับ | Unit → Integration → E2E → UAT |
| 5 | **Deployment** | Deploy อย่างเป็นระบบ | CI/CD pipeline, release process |
| 6 | **Maintenance** | บำรุงรักษาหลัง deploy | Bug fix process, monitoring, alerting |
| 7 | **Configuration Management** | จัดการ config และ version | Git, semantic versioning, changelog |
| 8 | **Quality Assurance** | ตรวจสอบคุณภาพ | Code review, coverage threshold, lint rules |
| 9 | **Verification & Validation** | ยืนยันว่าทำถูกและทำสิ่งที่ต้องการ | Test reports, acceptance criteria |
| 10 | **Documentation** | เอกสารครบถ้วน | API docs, architecture docs, runbook |

---

## 6. Checklist รวม — เทียบกับโปรเจกต์ปัจจุบัน

### ✅ สิ่งที่มีแล้ว

| # | หัวข้อ | ISO | หลักฐาน |
|---|--------|-----|---------|
| 1 | Architecture Documentation | 9001, 12207 | `documents/architecture/` |
| 2 | CI/CD Pipeline | 12207, 25010 | `run.ps1 ci`, Makefile, GitHub Actions |
| 3 | Unit Testing (174+ tests) | 25010, 12207 | `*_test.go` (27 files) |
| 4 | Linting (17 linters) | 25010 | `.golangci.yml` |
| 5 | Vulnerability Scanning | 27001 | `govulncheck` in CI |
| 6 | Authentication & Authorization | 27001, 25010 | JWT, API keys, RBAC |
| 7 | Secret Management | 27001 | Environment variables, K8s secrets |
| 8 | Input Validation | 27001, 25010 | `internal/shared/validator/` |
| 9 | Health Checks | 25010 | Liveness + readiness + startup probes |
| 10 | Rate Limiting | 27001 | Server rate limit config |
| 11 | Logging & Observability | 27001, 25010 | OpenTelemetry, structured logging |
| 12 | Containerization | 25010 | Dockerfile, K8s manifests |
| 13 | Auto-Scaling | 25010 | HPA (2-6 pods) |
| 14 | Graceful Shutdown | 25010 | Server graceful shutdown |
| 15 | Database Migration | 12207 | `cmd/migrate/`, versioned SQL files |
| 16 | API Documentation | 25010, 12207 | Swagger/OpenAPI |
| 17 | Deployment Guide | 9001, 12207 | `documents/infrastructure/deployment-guide.md` |
| 18 | Branch Strategy | 9001, 12207 | `develop` → `main` workflow |
| 19 | TLS/SSL Support | 27001 | Database TLS config |
| 20 | Password Hashing | 27001 | bcrypt |

### ❌ สิ่งที่ยังขาด

| # | หัวข้อ | ISO | ความสำคัญ | หมายเหตุ |
|---|--------|-----|----------|---------|
| 1 | **Code Coverage Reporting** | 25010, 9001 | 🔴 สูง | ยังไม่มี `--coverprofile` ใน CI |
| 2 | **Coverage Threshold** | 25010, 9001 | 🔴 สูง | ยังไม่มีเกณฑ์ขั้นต่ำ (เช่น ≥ 70%) |
| 3 | **Integration Test** | 25010, 12207 | 🟡 กลาง | มีบางส่วน แต่ยังไม่ครบ |
| 4 | **Load/Performance Test** | 25010 | 🟡 กลาง | ยังไม่มี benchmark/load test |
| 5 | **Risk Assessment Document** | 27001 | 🟡 กลาง | ยังไม่มี risk register |
| 6 | **Incident Response Plan** | 27001 | 🟡 กลาง | ยังไม่มี runbook เมื่อเกิดเหตุ |
| 7 | **Backup & Recovery Plan** | 27001 | 🟡 กลาง | ยังไม่มี DR plan |
| 8 | **Changelog** | 9001, 12207 | 🟢 ต่ำ | ยังไม่มี CHANGELOG.md |
| 9 | **Requirement Document** | 12207 | 🟢 ต่ำ | ยังไม่มี formal requirement spec |
| 10 | **Training Records** | 9001 | 🟢 ต่ำ | ยังไม่มี onboarding checklist |

---

## 7. Roadmap แนะนำ

### Phase 1 — Quick Wins (ทำได้เลย)

```
1. เพิ่ม Code Coverage ใน CI → make test-cover
2. ตั้ง Coverage Threshold ≥ 70%
3. สร้าง CHANGELOG.md
4. สร้าง Incident Response Runbook
```

### Phase 2 — Medium Effort (1-2 สัปดาห์)

```
5. เพิ่ม Integration Test ให้ครบทุก module
6. สร้าง Risk Assessment Document
7. สร้าง Backup & Recovery Plan
8. เพิ่ม Load Test (k6 หรือ vegeta)
```

### Phase 3 — Long Term (1 เดือน+)

```
9.  Formal Requirement Document
10. Training & Onboarding Program
11. Internal Audit Process
12. Management Review Cycle
```

### เป้าหมายรวม

```
ISO Readiness Score ปัจจุบัน:  20/30 (67%)
เป้าหมาย Phase 1:            24/30 (80%)
เป้าหมาย Phase 2:            28/30 (93%)
เป้าหมาย Phase 3:            30/30 (100%)
```

---

> **สรุป:** โปรเจกต์ปัจจุบันมีพื้นฐานดี (20/30) — สิ่งที่ขาดหลัก ๆ คือ **coverage reporting**, **risk assessment**, **incident response plan** ซึ่งทำเพิ่มได้ไม่ยาก
