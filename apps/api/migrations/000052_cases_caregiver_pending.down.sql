-- Migration: 000052_cases_caregiver_pending.down.sql
-- Description: 回滾照護人員待維護旗標與待裁決列的照護人員關聯。
-- cases.caregiver_id 的回填不還原：那是補齊既有資料的正確關聯，清掉只會再造成資料遺失。

ALTER TABLE case_import_duplicate_rows
    DROP COLUMN IF EXISTS caregiver_id;

DROP VIEW IF EXISTS case_pending_status;

CREATE VIEW case_pending_status AS
SELECT c.id AS case_id,
       (c.profile_pending OR c.site_pending OR COALESCE(p.link_pending, false)) AS is_pending
FROM cases c
LEFT JOIN case_transport_preferences p ON p.case_id = c.id;

DROP INDEX IF EXISTS ix_cases_caregiver_pending;

ALTER TABLE cases DROP COLUMN IF EXISTS caregiver_pending;
