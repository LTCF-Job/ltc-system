-- Migration: 000052_cases_caregiver_pending.up.sql
-- Description: 個案的「個管or照專」原本只有 care_contact_role / care_contact_name 兩個純文字欄位，
-- 與 caregivers 主檔脫節；匯入匯出改為以 cases.caregiver_id 為單一來源後，既有資料需回填關聯，
-- 回填不到的個案比照據點（site_pending）落入待維護，由使用者於待維護工作台補齊。
-- 回填規則與匯入端 resolveCaregiver 完全一致：以姓名為主，同名多筆時才用角色消歧。

-- 回填 1：姓名同名唯一命中即採用，角色以主檔為準（不看 care_contact_role）。
UPDATE cases c
SET caregiver_id = m.id
FROM (
    SELECT btrim(name) AS name, (array_agg(id))[1] AS id, count(*) AS cnt
    FROM caregivers
    GROUP BY btrim(name)
) m
WHERE c.caregiver_id IS NULL
  AND c.care_contact_name IS NOT NULL
  AND btrim(c.care_contact_name) = m.name
  AND m.name <> ''
  AND m.cnt = 1;

-- 回填 2：同名多筆時才用 care_contact_role 消歧，過濾後仍唯一才採用；
-- 角色空白或過濾後仍多筆者一律留空，不由 migration 猜測。
UPDATE cases c
SET caregiver_id = m.id
FROM (
    SELECT btrim(name) AS name, type, (array_agg(id))[1] AS id, count(*) AS cnt
    FROM caregivers
    GROUP BY btrim(name), type
) m
WHERE c.caregiver_id IS NULL
  AND c.care_contact_name IS NOT NULL
  AND btrim(c.care_contact_name) = m.name
  AND m.name <> ''
  AND m.cnt = 1
  AND m.type = CASE btrim(c.care_contact_role)
                   WHEN '個管' THEN 'case_manager'
                   WHEN '照專' THEN 'specialist'
                   WHEN '專護' THEN 'specialist'
                   ELSE NULL
               END;

DO $$
DECLARE
    missing_count INT;
BEGIN
    SELECT count(*) INTO missing_count
    FROM cases
    WHERE caregiver_id IS NULL
      AND care_contact_name IS NOT NULL
      AND btrim(care_contact_name) <> '';

    RAISE NOTICE '照護人員回填後仍無關聯的個案筆數：%（已標記待維護，需人工於待維護工作台補齊）', missing_count;
END $$;

-- 照護人員比對不到主檔：全站隱藏，待使用者手動補齊（與 site_pending 同一模式）。
-- 從未填過照護人員姓名的個案不算待維護，避免把「本來就沒有這筆資料」誤判成待補正。
ALTER TABLE cases ADD COLUMN caregiver_pending BOOLEAN GENERATED ALWAYS AS (
    caregiver_id IS NULL AND care_contact_name IS NOT NULL AND btrim(care_contact_name) <> ''
) STORED;

CREATE INDEX ix_cases_caregiver_pending ON cases (id) WHERE caregiver_pending;

DROP VIEW case_pending_status;

CREATE VIEW case_pending_status AS
SELECT c.id AS case_id,
       (c.profile_pending OR c.site_pending OR c.caregiver_pending OR COALESCE(p.link_pending, false)) AS is_pending
FROM cases c
LEFT JOIN case_transport_preferences p ON p.case_id = c.id;

-- 待裁決暫存列也保住已比對到的關聯，裁決成新個案時才不會退回未關聯狀態。
ALTER TABLE case_import_duplicate_rows
    ADD COLUMN caregiver_id UUID REFERENCES caregivers(id) ON DELETE SET NULL;
