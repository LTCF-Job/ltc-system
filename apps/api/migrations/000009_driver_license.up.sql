-- Migration: 000009_driver_license.up.sql
-- Description: 司機主檔新增駕照類別與駕照有效日期。政府申報的「服務駕駛清冊」
-- 需要逐位駕駛列出這兩欄，既有司機資料留 NULL 代表尚未補登。

ALTER TABLE drivers ADD COLUMN license_class TEXT;
ALTER TABLE drivers ADD COLUMN license_expiry_date DATE;
ALTER TABLE drivers ADD CONSTRAINT chk_driver_license_class
    CHECK (license_class IS NULL OR license_class IN ('sedan', 'truck', 'bus', 'trailer'));
