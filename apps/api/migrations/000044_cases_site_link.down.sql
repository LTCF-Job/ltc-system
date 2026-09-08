-- Migration: 000044_cases_site_link.down.sql
-- Description: 還原個案據點關聯欄位與 view。不可逆：哨兵值與回填結果無法還原成 migration 前
-- 「無據點欄位」的狀態，down 只還原 schema 形狀。

DROP VIEW IF EXISTS case_pending_status;

CREATE VIEW case_pending_status AS
SELECT c.id AS case_id,
       (c.profile_pending OR COALESCE(p.link_pending, false)) AS is_pending
FROM cases c
LEFT JOIN case_transport_preferences p ON p.case_id = c.id;

DROP INDEX IF EXISTS ix_cases_site_pending;
DROP INDEX IF EXISTS ix_cases_site_id;

ALTER TABLE cases DROP COLUMN IF EXISTS site_pending;
ALTER TABLE cases DROP CONSTRAINT IF EXISTS ck_cases_site_present;
ALTER TABLE cases DROP COLUMN IF EXISTS site_name_raw;
ALTER TABLE cases DROP COLUMN IF EXISTS site_id;
