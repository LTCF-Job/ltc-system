-- Migration: 000055_drop_case_transport_preferences.up.sql
-- Description: 個案不再需要關聯車輛。排班、搭乘、報表、匯出皆不依賴
-- case_transport_preferences，且它是車輛待維護（link_pending）唯一的來源，一併移除。
-- 不可逆：既有個案指定的去/回程車輛與比對不到主檔的原始名稱會直接遺失，down 只能還原
-- 表結構，無法還原資料。

DROP VIEW IF EXISTS case_pending_status;

DROP TABLE IF EXISTS case_transport_preferences;

ALTER TABLE case_import_duplicate_rows
    DROP COLUMN IF EXISTS outbound_vehicle_id,
    DROP COLUMN IF EXISTS outbound_vehicle_name_raw,
    DROP COLUMN IF EXISTS inbound_vehicle_id,
    DROP COLUMN IF EXISTS inbound_vehicle_name_raw;

CREATE VIEW case_pending_status AS
SELECT c.id AS case_id,
       (c.profile_pending OR c.site_pending OR c.caregiver_pending) AS is_pending
FROM cases c;
