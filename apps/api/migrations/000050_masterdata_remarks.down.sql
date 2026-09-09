-- Migration: 000050_masterdata_remarks.down.sql
-- Description: 回滾車輛主檔與據點主檔的備註欄位 (remarks)

ALTER TABLE vehicles DROP COLUMN IF EXISTS remarks;

ALTER TABLE sites DROP COLUMN IF EXISTS remarks;
