-- Migration: 000045_masterdata_field_changes.down.sql
-- Description: 還原主檔欄位調整。
--
-- 限制：sites.open_days 與 drivers.inspection_date 的原始值在 up 已被刪除，
-- 無法還原；open_days 回填為預設的週一到週五，inspection_date 回填為 NULL。

ALTER TABLE vehicles
    DROP COLUMN IF EXISTS has_vehicle_license,
    DROP COLUMN IF EXISTS has_purchase_contract,
    DROP COLUMN IF EXISTS has_plate_registration,
    DROP COLUMN IF EXISTS has_transfer_registration;

ALTER TABLE drivers ADD COLUMN IF NOT EXISTS inspection_date DATE;

ALTER TABLE sites ADD COLUMN IF NOT EXISTS open_days SMALLINT[] NOT NULL DEFAULT '{1,2,3,4,5}';
