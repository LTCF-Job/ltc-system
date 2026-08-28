-- Migration: 000005_drop_region_code.up.sql
-- Description: 移除 regions 區域資料表的 code 欄位

ALTER TABLE regions DROP COLUMN IF EXISTS code;
