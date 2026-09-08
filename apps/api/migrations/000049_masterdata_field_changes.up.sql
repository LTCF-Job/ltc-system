-- Migration: 000049_masterdata_field_changes.up.sql
-- Description: 主檔欄位調整
--   - 據點主檔移除開放星期 (sites.open_days)，據點地址保留。
--   - 司機主檔移除驗車日 (drivers.inspection_date)；驗車資訊屬於車輛，車輛已有 last_inspection_date。
--   - 車輛主檔新增四項證件持有註記（行照、汽車買賣合約書、領牌登記書、異動登記書）。

ALTER TABLE sites DROP COLUMN IF EXISTS open_days;

ALTER TABLE drivers DROP COLUMN IF EXISTS inspection_date;

ALTER TABLE vehicles
    ADD COLUMN IF NOT EXISTS has_vehicle_license BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS has_purchase_contract BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS has_plate_registration BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS has_transfer_registration BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN vehicles.has_vehicle_license IS '是否持有行照';
COMMENT ON COLUMN vehicles.has_purchase_contract IS '是否持有汽車買賣合約書';
COMMENT ON COLUMN vehicles.has_plate_registration IS '是否持有領牌登記書';
COMMENT ON COLUMN vehicles.has_transfer_registration IS '是否持有異動登記書';
