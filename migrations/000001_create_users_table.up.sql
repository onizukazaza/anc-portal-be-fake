-- 000001_create_users_table.up.sql
-- สร้างตาราง users สำหรับ authentication module
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(50) PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    roles JSONB NOT NULL DEFAULT '[]' :: jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index สำหรับ login query (FindByUsername)
CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);