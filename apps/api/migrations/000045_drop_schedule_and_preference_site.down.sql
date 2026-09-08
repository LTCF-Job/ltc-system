-- Migration: 000045_drop_schedule_and_preference_site.down.sql
-- Description: 還原 schema 形狀。不可逆：case_schedules.site_id 與 case_transport_preferences
-- 的 site 欄位原值已遺失，down 只能以 nullable 重建欄位，並盡力從 cases.site_id 回填交通偏好，
-- 不保證與 migration 前的實際值一致。

DROP VIEW IF EXISTS case_pending_status;

ALTER TABLE case_schedules ADD COLUMN site_id UUID REFERENCES sites(id) ON DELETE RESTRICT;

DROP INDEX IF EXISTS ix_case_transport_preferences_link_pending;
ALTER TABLE case_transport_preferences DROP COLUMN link_pending;

ALTER TABLE case_transport_preferences
    ADD COLUMN site_id UUID REFERENCES sites(id) ON DELETE SET NULL,
    ADD COLUMN site_name_raw TEXT;

UPDATE case_transport_preferences p
SET site_id = c.site_id,
    site_name_raw = c.site_name_raw
FROM cases c
WHERE c.id = p.case_id;

ALTER TABLE case_transport_preferences ADD COLUMN link_pending BOOLEAN GENERATED ALWAYS AS (
    (site_id IS NULL AND site_name_raw IS NOT NULL)
 OR (outbound_vehicle_id IS NULL AND outbound_vehicle_name_raw IS NOT NULL)
 OR (inbound_vehicle_id IS NULL AND inbound_vehicle_name_raw IS NOT NULL)
) STORED;

CREATE INDEX ix_case_transport_preferences_link_pending ON case_transport_preferences (case_id) WHERE link_pending;

CREATE VIEW case_pending_status AS
SELECT c.id AS case_id,
       (c.profile_pending OR c.site_pending OR COALESCE(p.link_pending, false)) AS is_pending
FROM cases c
LEFT JOIN case_transport_preferences p ON p.case_id = c.id;
