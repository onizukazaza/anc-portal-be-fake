# Import Data Guide

> **v2.0** — Last updated: March 2026
>
> ระบบนำ Base Data / Master Data จากไฟล์ CSV เข้า Database อัตโนมัติผ่าน CLI

---

## สารบัญ

1. [วัตถุประสงค์](#1-วัตถุประสงค์)
2. [ข้อมูลที่รองรับ](#2-ข้อมูลที่รองรับ)
3. [วิธีใช้งาน](#3-วิธีใช้งาน)
4. [Execution Flow](#4-execution-flow)
5. [โครงสร้างไฟล์](#5-โครงสร้างไฟล์)
6. [ตัวอย่าง CSV](#6-ตัวอย่าง-csv)
7. [Database Constraint](#7-database-constraint)

---

## 1. วัตถุประสงค์

- ลดการ insert ข้อมูลด้วย SQL แบบ manual
- รองรับการ import ข้อมูลจำนวนมาก (Bulk Data)
- Setup ระบบใหม่ทำได้ง่าย
- Update master data สามารถทำซ้ำได้อย่างปลอดภัย (UPSERT)

---

## 2. ข้อมูลที่รองรับ

| Service Type | Description |
|---|---|
| `insurer` | ข้อมูลบริษัทประกัน |
| `insurer_installment` | ตารางผ่อนชำระของบริษัทประกัน |
| `province` | Master data จังหวัด |
| `user` | ข้อมูลผู้ใช้ / ตัวแทน |

---

## 3. วิธีใช้งาน

### คำสั่งหลัก

```bash
go run ./cmd/import/main.go --env .env.local --path ./base_data/users.csv --service_type user
```

### Parameters

| Parameter | Description |
|---|---|
| `--env` | path ของ environment config |
| `--path` | path ของไฟล์ CSV |
| `--service_type` | ประเภท importer ที่ต้องการใช้ |

### ตัวอย่างทั้งหมด

```bash
# Import insurer
go run ./cmd/import/main.go --env .env.local --path ./base_data/insurer.csv --service_type insurer

# Import insurer installment
go run ./cmd/import/main.go --env .env.local --path ./base_data/insurer_installment.csv --service_type insurer_installment

# Import province
go run ./cmd/import/main.go --env .env.local --path ./base_data/province.csv --service_type province

# Import user
go run ./cmd/import/main.go --env .env.local --path ./base_data/users.csv --service_type user
```

---

## 4. Execution Flow

```
cmd/import/main.go
     │
     ▼
Parse CLI Flags (--env, --path, --service_type)
     │
     ▼
Load Config (config.Load)
     │
     ▼
Connect Database (postgres.NewManager)
     │
     ▼
Run Importer (importer.Run)
     │
     ▼
Dispatch → Read CSV → Validate → UPSERT → Commit
```

---

## 5. โครงสร้างไฟล์

```
cmd/import/
├── main.go                         ← Entry point (CLI)
└── import_data_guide.md            ← เอกสารนี้

internal/import/
├── runner.go                       ← Dispatch importer ตาม service type
├── csv_reader.go                   ← ReadCSV() → CSVData{Header, Rows}
├── insurer_importer.go             ← Import บริษัทประกัน
├── insurer_installment_importer.go ← Import ตารางผ่อนชำระ
├── province_importer.go            ← Import จังหวัด
└── user_importer.go                ← Import ผู้ใช้ / ตัวแทน

base_data/
├── insurer_installment.csv
└── users.csv
```

---

## 6. ตัวอย่าง CSV

### insurer.csv

```csv
code,name,status
BKI,Bangkok Insurance,active
VBI,Viriyah Insurance,active
TIP,Thaivivat Insurance,active
```

### province.csv

```csv
code,name
10,Bangkok
20,Chonburi
30,Nakhon Ratchasima
```

### insurer_installment.csv

```csv
insurer_code,installment_month,interest_rate,status
BKI,3,0,active
BKI,6,1.5,active
VBI,3,0,active
```

---

## 7. Database Constraint

เพื่อให้ UPSERT ทำงานได้ถูกต้อง table ต้องมี UNIQUE constraint:

```sql
ALTER TABLE insurer
ADD CONSTRAINT uq_insurer_code UNIQUE (code);

ALTER TABLE province
ADD CONSTRAINT uq_province_code UNIQUE (code);

ALTER TABLE insurer_installment
ADD CONSTRAINT uq_installment UNIQUE (insurer_code, installment_month);
```

---

> **v2.0** — March 2026 | ANC Portal Backend Team
```

---

## 🔐 Transaction Control

ทุก importer ทำงานภายใต้ transaction — ป้องกัน partial insert

* สามารถ rollback ได้เมื่อเกิด error
* ข้อมูลมี consistency

---

## 🚨 Error Handling

ระบบจะหยุดทันทีเมื่อพบ error พร้อมแจ้งหมายเลขบรรทัด

```
line 14: invalid interest_rate
line 22: insurer_code is required
```

---

## 📌 Best Practice

* CSV ต้องมี header
* ใช้ lowercase column name
* validate ข้อมูลก่อน insert
* ใช้ transaction ทุกครั้ง
