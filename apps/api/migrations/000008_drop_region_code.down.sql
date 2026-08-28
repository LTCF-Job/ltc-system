-- Migration: 000008_drop_region_code.down.sql
-- Description: 復原 regions 區域資料表的 code 欄位

ALTER TABLE regions ADD COLUMN IF NOT EXISTS code TEXT;
