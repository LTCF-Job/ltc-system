-- Migration: 000009_driver_license.down.sql
-- Description: 還原司機主檔的駕照類別與駕照有效日期欄位。

ALTER TABLE drivers DROP CONSTRAINT IF EXISTS chk_driver_license_class;
ALTER TABLE drivers DROP COLUMN IF EXISTS license_expiry_date;
ALTER TABLE drivers DROP COLUMN IF EXISTS license_class;
