-- Migration: 000046_vehicles_site_text.down.sql
-- Description: 還原 schema 形狀。不可逆：以 sites.name 盡力依名稱回連，改過名稱或多筆
-- 同名據點時無法保證與 migration 前的關聯一致。

ALTER TABLE vehicles ADD COLUMN site_id UUID REFERENCES sites(id) ON DELETE RESTRICT;

UPDATE vehicles v
SET site_id = s.id
FROM sites s
WHERE s.name = v.site_name;

CREATE INDEX idx_vehicles_site ON vehicles(site_id);

ALTER TABLE vehicles DROP COLUMN site_name;
