-- Migration: 000055_drop_case_transport_preferences.down.sql
-- Description: 還原 case_transport_preferences 的表結構（依 000001／000003／000045 的形狀）
-- 與 case_import_duplicate_rows 的車輛欄位。只能還原結構，個案原本指定的去/回程車輛資料
-- 已在 up 遺失，down 後全數為空，不會重建任何列。

ALTER TABLE case_import_duplicate_rows
    ADD COLUMN outbound_vehicle_id UUID REFERENCES vehicles(id) ON DELETE SET NULL,
    ADD COLUMN outbound_vehicle_name_raw TEXT,
    ADD COLUMN inbound_vehicle_id UUID REFERENCES vehicles(id) ON DELETE SET NULL,
    ADD COLUMN inbound_vehicle_name_raw TEXT;

DROP VIEW IF EXISTS case_pending_status;

CREATE TABLE case_transport_preferences (
    case_id UUID PRIMARY KEY REFERENCES cases(id) ON DELETE CASCADE,
    outbound_vehicle_id UUID REFERENCES vehicles(id) ON DELETE SET NULL,
    inbound_vehicle_id UUID REFERENCES vehicles(id) ON DELETE SET NULL,
    outbound_vehicle_name_raw TEXT,
    inbound_vehicle_name_raw TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE case_transport_preferences ADD COLUMN link_pending BOOLEAN GENERATED ALWAYS AS (
    (outbound_vehicle_id IS NULL AND outbound_vehicle_name_raw IS NOT NULL)
 OR (inbound_vehicle_id IS NULL AND inbound_vehicle_name_raw IS NOT NULL)
) STORED;

CREATE INDEX ix_case_transport_preferences_link_pending ON case_transport_preferences (case_id) WHERE link_pending;

CREATE VIEW case_pending_status AS
SELECT c.id AS case_id,
       (c.profile_pending OR c.site_pending OR c.caregiver_pending OR COALESCE(p.link_pending, false)) AS is_pending
FROM cases c
LEFT JOIN case_transport_preferences p ON p.case_id = c.id;
