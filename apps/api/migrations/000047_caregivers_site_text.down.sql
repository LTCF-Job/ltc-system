-- Migration: 000047_caregivers_site_text.down.sql
-- Description: 還原 schema 形狀。不可逆：以名稱盡力回連主檔，比對不到的一律落回
-- site_name_raw，無法保證與 migration 前的關聯一致。

ALTER TABLE caregivers
    ADD COLUMN site_id UUID REFERENCES sites(id),
    ADD COLUMN site_name_raw TEXT;

UPDATE caregivers c
SET site_id = s.id
FROM sites s
WHERE s.name = c.site_name;

UPDATE caregivers
SET site_name_raw = site_name
WHERE site_id IS NULL AND site_name IS NOT NULL;

CREATE INDEX idx_caregivers_site_id ON caregivers(site_id);

ALTER TABLE caregivers DROP COLUMN site_name;
