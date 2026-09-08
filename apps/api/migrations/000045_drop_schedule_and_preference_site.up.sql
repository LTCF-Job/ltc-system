-- Migration: 000045_drop_schedule_and_preference_site.up.sql
-- Description: 據點歸屬重構第二步。個案主檔已於 000044 取得自己的據點關聯，排班
-- (case_schedules) 與交通偏好 (case_transport_preferences) 上重複的據點欄位移除，
-- 一律改看個案本身的據點。link_pending 重建為只保留去/回程車輛兩個條件（單位/據點條件
-- 已搬到 cases.site_pending，此處移除避免重複判定）。
-- 不可逆：case_schedules.site_id 原本允許「同一個案不同排班掛不同據點」，drop 後該歷史
-- 差異無法還原。

DROP VIEW IF EXISTS case_pending_status;

ALTER TABLE case_transport_preferences DROP COLUMN link_pending;
ALTER TABLE case_transport_preferences
    DROP COLUMN site_id,
    DROP COLUMN site_name_raw;

ALTER TABLE case_transport_preferences ADD COLUMN link_pending BOOLEAN GENERATED ALWAYS AS (
    (outbound_vehicle_id IS NULL AND outbound_vehicle_name_raw IS NOT NULL)
 OR (inbound_vehicle_id IS NULL AND inbound_vehicle_name_raw IS NOT NULL)
) STORED;

CREATE INDEX ix_case_transport_preferences_link_pending ON case_transport_preferences (case_id) WHERE link_pending;

ALTER TABLE case_schedules DROP COLUMN site_id;

CREATE VIEW case_pending_status AS
SELECT c.id AS case_id,
       (c.profile_pending OR c.site_pending OR COALESCE(p.link_pending, false)) AS is_pending
FROM cases c
LEFT JOIN case_transport_preferences p ON p.case_id = c.id;
