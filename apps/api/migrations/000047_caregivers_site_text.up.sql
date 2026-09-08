-- Migration: 000047_caregivers_site_text.up.sql
-- Description: 據點歸屬重構第四步。照護人員不再關聯據點主檔，改為自由輸入文字欄位
-- （非必填）。既有資料優先以主檔名稱回填，找不到主檔的既有 site_name_raw 值接著補上；
-- caregivers.is_pending 定義不含 site，不受影響。

ALTER TABLE caregivers ADD COLUMN site_name TEXT;

UPDATE caregivers c
SET site_name = s.name
FROM sites s
WHERE s.id = c.site_id;

UPDATE caregivers
SET site_name = site_name_raw
WHERE site_name IS NULL AND site_name_raw IS NOT NULL;

DROP INDEX IF EXISTS idx_caregivers_site_id;

ALTER TABLE caregivers
    DROP COLUMN site_id,
    DROP COLUMN site_name_raw;
