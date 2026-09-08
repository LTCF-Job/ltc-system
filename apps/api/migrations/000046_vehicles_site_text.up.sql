-- Migration: 000046_vehicles_site_text.up.sql
-- Description: 據點歸屬重構第三步。車輛不再關聯據點主檔，改為該筆車輛自己的自由輸入
-- 文字欄位（非必填）。既有資料以據點名稱回填，車輛所屬的「區域」欄位（原本由 sites JOIN
-- 出的唯讀欄位）同時失去來源，隨本 migration 一併移除，前端相關篩選另於程式碼調整。

ALTER TABLE vehicles ADD COLUMN site_name TEXT;

UPDATE vehicles v
SET site_name = s.name
FROM sites s
WHERE s.id = v.site_id;

DROP INDEX IF EXISTS idx_vehicles_site;

ALTER TABLE vehicles DROP COLUMN site_id;
