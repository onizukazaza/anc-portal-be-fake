-- 000002_create_insurer_tables.up.sql
-- ตาราง insurer และ insurer_installment สำหรับข้อมูลบริษัทประกัน
CREATE TABLE IF NOT EXISTS insurer (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    STATUS VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_insurer_code ON insurer (code);

CREATE INDEX IF NOT EXISTS idx_insurer_status ON insurer (STATUS);

CREATE TABLE IF NOT EXISTS insurer_installment (
    id SERIAL PRIMARY KEY,
    insurer_code VARCHAR(50) NOT NULL,
    installment_month INT NOT NULL,
    interest_rate DECIMAL(10, 4) NOT NULL DEFAULT 0,
    STATUS VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_insurer_installment UNIQUE (insurer_code, installment_month),
    CONSTRAINT fk_insurer_installment_insurer FOREIGN KEY (insurer_code) REFERENCES insurer (code) ON DELETE CASCADE
);