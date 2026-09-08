-- Migration: 000042_relax_site_and_driver_fields.up.sql
-- Description: 單位管理：所屬區域 (region)、單位地址 (address) 改為非必填
-- 司機管理：所屬區域 (region) 改為非必填，並新增性別、生日、職業駕照、到職日、異動登記書、驗車日、備註等選填欄位

-- 1. 單位表放寬必填約束
ALTER TABLE sites
    ALTER COLUMN address DROP NOT NULL,
    ALTER COLUMN region DROP NOT NULL;

-- 2. 司機表放寬所屬區域必填約束
ALTER TABLE drivers
    ALTER COLUMN region DROP NOT NULL;

-- 3. 司機表新增管理清冊擴充欄位（皆為選填）
ALTER TABLE drivers
    ADD COLUMN IF NOT EXISTS gender TEXT,
    ADD COLUMN IF NOT EXISTS birth_date DATE,
    ADD COLUMN IF NOT EXISTS has_professional_license BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS employment_date DATE,
    ADD COLUMN IF NOT EXISTS has_transfer_cert BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS inspection_date DATE,
    ADD COLUMN IF NOT EXISTS remarks TEXT;
