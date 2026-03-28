# GitHub Webhook → Discord Notification

> **v2.0** — Last updated: March 2026
>
> แนวทางการออกแบบระบบแจ้งเตือน GitHub events ไปยัง Discord ผ่าน Golang service
>
> เอกสารนี้เป็นแนวคิดเพื่อต่อยอด — อาจปรับเปลี่ยนตามโครงสร้างจริง

---

## สารบัญ

1. [ภาพรวม](#1-ภาพรวม)
2. [เป้าหมาย](#2-เป้าหมาย)
3. [สถาปัตยกรรม](#3-สถาปัตยกรรม)
4. [ทำไมต้องมี Golang Service](#4-ทำไมต้องมี-golang-service)
5. [โครงสร้างโปรเจคที่แนะนำ](#5-โครงสร้างโปรเจคที่แนะนำ)

---

## 1. ภาพรวม

เชื่อมต่อ **GitHub Webhook** เข้ากับ **Discord Webhook** ผ่าน **Golang Service**
เพื่อให้ทีมได้รับแจ้งเตือนอัตโนมัติเมื่อมี event จาก GitHub

### Use Cases

- แจ้งเตือนการ push code เข้า branch สำคัญ (`main`, `develop`)
- แสดงข้อมูล: ใคร push, commit message, branch, เวลา
- ปรับรูปแบบข้อความ Discord ให้เหมาะกับทีม
- ขยายต่อไปยัง event อื่นได้ (Pull Request, Review, Comment)

---

## 2. เป้าหมาย

ให้ทีมเห็นความเปลี่ยนแปลงจาก GitHub ได้ทันทีโดยไม่ต้องเปิด repository

### ตัวอย่างข้อความ Discord

```
🚀 New push to main
Repo: anc-portal-be
By: ancbroker
Time: 2026-03-23 09:38
Commit: 9226c93
Message: Update README with GitHub webhook notification details
```

### ประโยชน์

- ทีมรู้ทันทีว่ามีการ push code เข้า branch สำคัญ
- ตรวจสอบได้ว่าใครเป็นผู้ push
- เห็น commit ล่าสุดแบบย่อโดยไม่ต้องเปิด GitHub
- Dev / Reviewer ติดตาม activity ของ repository ได้สะดวก
- ต่อยอดไปยัง notification อื่นได้ง่าย

---

## 3. สถาปัตยกรรม

```
GitHub Repository
     │
     │  webhook event (HTTP POST)
     ▼
Golang Webhook Service
     │
     ├── รับ payload
     ├── verify signature
     ├── parse ข้อมูลที่จำเป็น
     ├── apply business rules
     ├── build notification message
     │
     │  HTTP POST (Discord Webhook URL)
     ▼
Discord Channel
     └── แสดง notification ตามรูปแบบที่กำหนด
```

---

## 4. ทำไมต้องมี Golang Service

GitHub ส่ง webhook event ได้ และ Discord รับ webhook message ได้
แต่ถ้าต้องการ **format ข้อความแบบ custom** เช่น:

- แสดงชื่อคน push
- แสดง short commit SHA
- แสดงเวลาในรูปแบบที่อ่านง่าย
- แสดงเฉพาะบาง branch
- แสดงหลาย commits ใน push เดียว

**ต้องมี service เป็นตัวกลาง** — GitHub/Discord ตั้งค่าเหล่านี้โดยตรงไม่ได้

---

## 5. โครงสร้างโปรเจคที่แนะนำ

แยก responsibility ออกเป็น layer ให้ชัด — ไม่ยัดปนกับ business domain หลัก

```
internal/
├── githubwebhook/
│   ├── handler.go          ← HTTP handler รับ webhook event
│   ├── service.go          ← Business logic (verify, parse, dispatch)
│   ├── model.go            ← GitHub payload struct
│   └── mapper.go           ← แปลง GitHub payload → Discord message
│
└── notification/
    ├── discord.go           ← Discord webhook client
    └── message_builder.go   ← สร้าง Discord embed message

config/
└── config.go                ← Discord webhook URL, GitHub secret

cmd/
└── api/
    └── main.go              ← Register webhook route
```

---

> **v2.0** — March 2026 | ANC Portal Backend Team
