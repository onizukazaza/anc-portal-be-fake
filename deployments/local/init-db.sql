-- =============================================================================
-- init-db.sql — สร้าง database สำหรับ local development
-- =============================================================================
-- ไฟล์นี้จะถูกรันอัตโนมัติเมื่อ PostgreSQL container เริ่มต้นครั้งแรก
-- (จาก docker-entrypoint-initdb.d)
-- =============================================================================
-- Main database
CREATE DATABASE "anc-portal";

-- External database (legacy ERP)
CREATE DATABASE "meprakun_local_v2";