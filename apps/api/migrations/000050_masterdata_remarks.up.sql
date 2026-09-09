-- Migration: 000050_masterdata_remarks.up.sql
-- Description: 車輛主檔與據點主檔新增備註欄位 (remarks)

ALTER TABLE vehicles
    ADD COLUMN IF NOT EXISTS remarks TEXT;

ALTER TABLE sites
    ADD COLUMN IF NOT EXISTS remarks TEXT;

COMMENT ON COLUMN vehicles.remarks IS '車輛備註';
COMMENT ON COLUMN sites.remarks IS '據點備註';
