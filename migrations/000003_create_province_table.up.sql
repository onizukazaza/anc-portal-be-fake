-- 000003_create_province_table.up.sql
-- ตาราง province สำหรับข้อมูลจังหวัด
CREATE TABLE IF NOT EXISTS province (
    id SERIAL PRIMARY KEY,
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    name_th VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_province_code ON province (code);