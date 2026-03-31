# Incident Response Runbook — ANC Portal Backend

> **v1.0** — Last updated: March 2026
>
> คู่มือรับมือเหตุฉุกเฉิน — เมื่อเกิดเหตุ ทำตามขั้นตอนนี้ทันที
> ไม่ต้องคิดเอง ไม่ต้องจำ แค่ทำตาม checklist

---

## สารบัญ

1. [Severity Levels](#1-severity-levels)
2. [ใครทำอะไร (Roles)](#2-ใครทำอะไร-roles)
3. [Runbook 1 — API ล่ม (Service Down)](#3-runbook-1--api-ล่ม-service-down)
4. [Runbook 2 — Database ล่ม](#4-runbook-2--database-ล่ม)
5. [Runbook 3 — Response ช้าผิดปกติ](#5-runbook-3--response-ช้าผิดปกติ)
6. [Runbook 4 — ถูกโจมตี / Security Incident](#6-runbook-4--ถูกโจมตี--security-incident)
7. [Runbook 5 — Memory / CPU สูงผิดปกติ](#7-runbook-5--memory--cpu-สูงผิดปกติ)
8. [Runbook 6 — Deploy แล้วพัง](#8-runbook-6--deploy-แล้วพัง)
9. [Runbook 7 — Kafka / Worker หยุดทำงาน](#9-runbook-7--kafka--worker-หยุดทำงาน)
10. [Post-Incident Process](#10-post-incident-process)
11. [Contact List](#11-contact-list)
12. [เทมเพลต Post-Mortem](#12-เทมเพลต-post-mortem)

---

## 1. Severity Levels

| Level | ชื่อ | ตัวอย่าง | Response Time | ต้องทำ |
|-------|------|---------|--------------|--------|
| **SEV-1** | Critical | API ล่มทั้งระบบ, ข้อมูลรั่ว | < 15 นาที | แจ้งทีม + แก้ทันที |
| **SEV-2** | Major | บาง feature ใช้ไม่ได้, response ช้ามาก | < 1 ชั่วโมง | แจ้งทีม + หาสาเหตุ |
| **SEV-3** | Minor | Error เป็นระยะ, log ผิดปกติ | < 4 ชั่วโมง | สร้าง ticket + fix |
| **SEV-4** | Low | Warning ใน log, performance ลดเล็กน้อย | Next sprint | สร้าง ticket |

---

## 2. ใครทำอะไร (Roles)

| Role | ความรับผิดชอบ |
|------|-------------|
| **Incident Commander (IC)** | ตัดสินใจ, ประสานงาน, แจ้ง stakeholder |
| **On-Call Engineer** | วินิจฉัย + แก้ไขปัญหาเบื้องต้น |
| **Subject Matter Expert** | ผู้เชี่ยวชาญเฉพาะด้าน (DB, K8s, Security) |
| **Communicator** | อัพเดทสถานะให้ทีม/ลูกค้า |

---

## 3. Runbook 1 — API ล่ม (Service Down)

### อาการ

- Health check fail
- ผู้ใช้เข้าไม่ได้
- Monitoring แจ้ง pod restart หรือ CrashLoopBackOff

### ขั้นตอน

```
□ Step 1: ยืนยันว่าล่มจริง
  → curl health check endpoint
  → kubectl get pods -n <namespace>

□ Step 2: ดู pod status
  → kubectl describe pod <pod-name> -n <namespace>
  → kubectl logs <pod-name> -n <namespace> --tail=100

□ Step 3: ดู event
  → kubectl get events -n <namespace> --sort-by='.lastTimestamp'

□ Step 4: ตรวจสอบสาเหตุที่พบบ่อย
  → OOMKilled?         → เพิ่ม memory limit
  → CrashLoopBackOff?  → ดู log หา panic/error
  → ImagePullBackOff?  → ตรวจ image tag + registry
  → Readiness fail?    → ตรวจ DB connection / dependency

□ Step 5: แก้ไขเบื้องต้น
  → kubectl rollout restart deployment/api -n <namespace>
  → (ถ้า config ผิด) แก้ ConfigMap/Secret แล้ว restart

□ Step 6: ยืนยันว่ากลับมาปกติ
  → curl health check
  → ตรวจ pod status = Running
  → ตรวจ readiness = true
```

### Escalation

ถ้า 15 นาทีแก้ไม่ได้ → rollback ไป version ก่อนหน้า:

```bash
kubectl rollout undo deployment/api -n <namespace>
```

---

## 4. Runbook 2 — Database ล่ม

### อาการ

- API return 500 / connection refused
- Log แสดง `"connection refused"` หรือ `"timeout"`
- Health check fail ที่ DB endpoint

### ขั้นตอน

```
□ Step 1: ตรวจ DB สถานะ
  → pg_isready -h <host> -p 5432
  → (Cloud) ดู dashboard ของ DB provider

□ Step 2: ตรวจ connection pool
  → ดู log: "too many connections"?
  → SELECT count(*) FROM pg_stat_activity;

□ Step 3: สาเหตุที่พบบ่อย

  Connection limit เต็ม:
  → ตรวจ DB_MAX_CONNS ค่าปัจจุบัน (default: 20)
  → ลด connection หรือเพิ่ม limit ที่ DB

  DB disk เต็ม:
  → ตรวจ disk usage
  → ลบ log / VACUUM FULL

  Long-running query:
  → SELECT pid, now() - pg_stat_activity.query_start AS duration, query
    FROM pg_stat_activity
    WHERE state != 'idle'
    ORDER BY duration DESC;
  → pg_cancel_backend(<pid>) หรือ pg_terminate_backend(<pid>)

□ Step 4: ถ้า DB เข้าไม่ได้เลย
  → restart DB service
  → (Cloud) failover ไป replica
  → แจ้ง IC ทันที

□ Step 5: ยืนยัน
  → pg_isready
  → API health check ผ่าน
  → ตรวจ error rate กลับเป็น 0
```

---

## 5. Runbook 3 — Response ช้าผิดปกติ

### อาการ

- Response time > 5 วินาที (ปกติ < 500ms)
- Monitoring แจ้ง latency spike
- ผู้ใช้แจ้ง "ช้ามาก"

### ขั้นตอน

```
□ Step 1: ระบุขอบเขต
  → ช้าทุก endpoint หรือแค่บาง endpoint?
  → ช้าตั้งแต่เมื่อไหร่?

□ Step 2: ตรวจ resource
  → kubectl top pods -n <namespace>
  → CPU สูงไหม? Memory สูงไหม?

□ Step 3: ตรวจ DB
  → Slow query?
  → SELECT * FROM pg_stat_activity WHERE state = 'active';
  → Connection pool เต็ม?

□ Step 4: ตรวจ external dependency
  → Redis ตอบช้า?
  → External API (เช่น meprakun) ตอบช้า?
  → ดู tracing (Grafana Tempo) หา bottleneck

□ Step 5: แก้ไขตาม root cause
  → Slow query → เพิ่ม index / optimize query
  → CPU สูง → scale pod (HPA จะทำอัตโนมัติ)
  → External ช้า → เพิ่ม timeout / enable cache
  → Connection pool → เพิ่ม DB_MAX_CONNS

□ Step 6: ยืนยัน
  → Response time กลับมา < 500ms
  → ไม่มี timeout error ใน log
```

---

## 6. Runbook 4 — ถูกโจมตี / Security Incident

### อาการ

- Request rate สูงผิดปกติ (DDoS)
- Login attempt จาก IP แปลก ๆ จำนวนมาก
- พบ SQL injection / XSS attempt ใน log
- ข้อมูลถูกเข้าถึงโดยไม่ได้รับอนุญาต

### ขั้นตอน (ทำทันที — ไม่ต้องรอ approval)

```
□ Step 1: ประเมินความรุนแรง
  → ข้อมูลรั่วหรือยัง?
  → ระบบยังทำงานอยู่หรือไหม?
  → กระทบผู้ใช้กี่คน?

□ Step 2: Contain (หยุดความเสียหาย)
  → Block IP ที่น่าสงสัย (Firewall / WAF)
  → ถ้ามีข้อมูลรั่ว → ปิด endpoint ที่มีปัญหาทันที
  → Revoke compromised API keys / JWT tokens
  → เปลี่ยน password ทุก account ที่อาจถูกกระทบ

□ Step 3: Investigate
  → ดู access log: IP ไหน, endpoint ไหน, เมื่อไหร่
  → ดู auth log: login ที่สำเร็จจาก IP แปลก
  → ตรวจ DB: มี data ที่ถูกแก้ไขผิดปกติไหม

□ Step 4: Eradicate (กำจัดต้นเหตุ)
  → Patch vulnerability ที่เจอ
  → อัพเดท dependency ที่มีช่องโหว่
  → เพิ่ม rate limit / WAF rule

□ Step 5: แจ้ง
  → แจ้ง IC + ทีม
  → (ถ้าข้อมูลรั่ว) แจ้งผู้บริหาร + ฝ่ายกฎหมาย
  → บันทึก timeline ทุกอย่างที่ทำ

□ Step 6: Recovery
  → ยืนยันว่า vulnerability ถูกแก้แล้ว
  → restore data จาก backup (ถ้าจำเป็น)
  → monitor อย่างใกล้ชิด 24-48 ชม.
```

### สิ่งที่ห้ามทำ

- ❌ ห้ามลบ log (ต้องเก็บเป็นหลักฐาน)
- ❌ ห้ามแก้ไข DB โดยตรงโดยไม่บันทึก
- ❌ ห้ามปิดระบบทั้งหมดถ้าไม่จำเป็น (contain เฉพาะจุด)

---

## 7. Runbook 5 — Memory / CPU สูงผิดปกติ

### อาการ

- Pod OOMKilled (ถูก kill เพราะ memory เกิน)
- CPU throttling
- HPA scale pod จำนวนมากผิดปกติ

### ขั้นตอน

```
□ Step 1: ดู resource usage
  → kubectl top pods -n <namespace>
  → kubectl describe pod <pod> (ดู limits/requests)

□ Step 2: หา root cause

  Memory Leak:
  → ดู memory trend ใน Grafana
  → ค่อย ๆ เพิ่มเรื่อย ๆ ไม่ลดลงเลย?
  → goroutine leak? → go tool pprof

  CPU Spike:
  → endpoint ไหนถูกเรียกเยอะ?
  → มี infinite loop / expensive computation?
  → N+1 query?

□ Step 3: แก้ไขเบื้องต้น
  → Restart pod เพื่อคลาย memory ชั่วคราว
  → kubectl rollout restart deployment/api -n <namespace>

□ Step 4: แก้ไขถาวร
  → Fix memory leak ในโค้ด
  → เพิ่ม resource limits ถ้าจำเป็น
  → Optimize query / เพิ่ม cache

□ Step 5: ยืนยัน
  → Memory stable (ไม่เพิ่มขึ้นเรื่อย ๆ)
  → CPU < 80% sustained
  → Pod ไม่ restart
```

---

## 8. Runbook 6 — Deploy แล้วพัง

### อาการ

- Deploy version ใหม่แล้ว error เพิ่มขึ้น
- Feature ที่เคยทำงานได้กลับไม่ทำงาน
- New version CrashLoopBackOff

### ขั้นตอน

```
□ Step 1: ยืนยันว่า deploy เป็นสาเหตุ
  → ปัญหาเริ่มหลัง deploy เมื่อไร?
  → kubectl rollout history deployment/api -n <namespace>

□ Step 2: Rollback ทันที (ถ้า SEV-1/SEV-2)
  → kubectl rollout undo deployment/api -n <namespace>
  → ยืนยันว่ากลับมาปกติ

□ Step 3: วิเคราะห์สาเหตุ (หลัง rollback แล้ว)
  → ดู diff ระหว่าง version เก่า vs ใหม่
  → ดู log ของ version ที่พัง
  → Migration ผิดไหม?
  → Config/Secret เปลี่ยนหรือเปล่า?
  → Dependency version ใหม่มีปัญหา?

□ Step 4: Fix + Re-deploy
  → แก้ bug
  → เพิ่ม test ที่จับ bug นี้ได้
  → รัน CI ให้ผ่าน
  → Deploy ใหม่ + monitor

□ Step 5: ป้องกันซ้ำ
  → เพิ่ม test case สำหรับ scenario นี้
  → ปรับ CI pipeline (ถ้าจำเป็น)
  → อัพเดท runbook (ถ้าได้เรียนรู้อะไรใหม่)
```

---

## 9. Runbook 7 — Kafka / Worker หยุดทำงาน

### อาการ

- Message ค้างใน queue ไม่ถูก consume
- Worker pod ไม่ทำงาน / CrashLoop
- DLQ (Dead Letter Queue) message เพิ่มขึ้น

### ขั้นตอน

```
□ Step 1: ตรวจ Worker pod
  → kubectl get pods -l app=worker -n <namespace>
  → kubectl logs <worker-pod> --tail=100

□ Step 2: ตรวจ Kafka
  → Kafka broker ทำงานอยู่ไหม?
  → Consumer group lag เท่าไหร่?

□ Step 3: สาเหตุที่พบบ่อย

  Worker crash:
  → ดู log หา panic/error
  → fix bug → redeploy worker

  Kafka broker ล่ม:
  → restart broker
  → ตรวจ disk space

  Message format ผิด:
  → ดู DLQ message
  → fix producer / consumer schema

  Consumer group stuck:
  → reset consumer group offset (ระวัง — อาจ process ซ้ำ)

□ Step 4: แก้ไข + ยืนยัน
  → Worker pod Running
  → Consumer lag ลดลง
  → DLQ ไม่เพิ่มขึ้น
```

---

## 10. Post-Incident Process

**ทุก SEV-1 และ SEV-2** ต้องทำ Post-Incident Review ภายใน 48 ชั่วโมง

### ขั้นตอน

```
□ 1. Incident Commander เขียน Post-Mortem (ใช้เทมเพลตด้านล่าง)
□ 2. ประชุม Post-Incident Review กับทีม
□ 3. ระบุ Action Items + Owner + Deadline
□ 4. ติดตาม Action Items จนเสร็จ
□ 5. อัพเดท Runbook ถ้าเรียนรู้อะไรใหม่
```

### หลัก Blameless Culture

- ❌ ไม่โทษคน → ✅ หาจุดอ่อนของ **ระบบ/process**
- ❌ "ใครทำพัง" → ✅ "ทำไมระบบไม่จับได้ก่อน deploy"
- ❌ "ทำไมไม่ระวัง" → ✅ "เพิ่ม test/check อะไรเพื่อป้องกัน"

---

## 11. Contact List

> **หมายเหตุ:** เติมข้อมูลจริงของทีมในตารางนี้

| Role | ชื่อ | ติดต่อ | หมายเหตุ |
|------|------|--------|---------|
| Incident Commander | (เติม) | (เติม) | ตัดสินใจ + ประสาน |
| Backend Lead | (เติม) | (เติม) | API + Business logic |
| DevOps / Infra | (เติม) | (เติม) | K8s + DB + CI/CD |
| Security | (เติม) | (เติม) | Security incidents |
| Product Owner | (เติม) | (เติม) | แจ้ง business impact |

### ช่องทางแจ้งเหตุ

| ช่องทาง | ใช้เมื่อ |
|---------|---------|
| **Discord #incidents** | แจ้งทีมทันที (SEV-1, SEV-2) |
| **Discord #alerts** | Monitoring alerts อัตโนมัติ |
| **Phone** | SEV-1 ที่ต้องการคนเฉพาะ |

---

## 12. เทมเพลต Post-Mortem

```markdown
# Post-Mortem: [ชื่อเหตุการณ์]

## Summary
- **วันที่เกิด:** YYYY-MM-DD HH:MM
- **ระยะเวลา:** X ชั่วโมง Y นาที
- **Severity:** SEV-?
- **Impact:** [กระทบอะไร กี่คน]

## Timeline
| เวลา | เหตุการณ์ |
|-------|---------|
| HH:MM | ตรวจพบปัญหา ... |
| HH:MM | เริ่มวินิจฉัย ... |
| HH:MM | แก้ไขโดย ... |
| HH:MM | ยืนยันว่ากลับมาปกติ |

## Root Cause
[อธิบายสาเหตุที่แท้จริง]

## What Went Well
- [สิ่งที่ทำได้ดี]

## What Went Wrong
- [สิ่งที่ต้องปรับปรุง]

## Action Items
| # | Action | Owner | Deadline | Status |
|---|--------|-------|----------|--------|
| 1 | [สิ่งที่ต้องทำ] | [ใคร] | [เมื่อไหร่] | ☐ |
| 2 | ... | ... | ... | ☐ |

## Lessons Learned
- [บทเรียนที่ได้]
```

---

> **สรุป:** Runbook ไม่ใช่แค่เอกสาร — เป็นเครื่องมือที่ช่วยให้ทีม **แก้ปัญหาได้เร็ว ไม่ต้องคิดเอง** ในเวลาที่กดดัน
> อัพเดทหลังทุก incident เพื่อให้ดีขึ้นเรื่อย ๆ
