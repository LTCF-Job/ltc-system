-- Migration: 000044_cases_site_link.up.sql
-- Description: 據點歸屬重構第一步。個案主檔新增據點關聯（site_id + site_name_raw），
-- 取代原本掛在交通偏好與排班上的據點。既有資料回填優先序：交通偏好 -> 排班（取最新生效範圍）。
-- 兩者皆無據點來源的個案，以哨兵值填入 site_name_raw 使其落入待維護清單，由使用者手動補齊，
-- 不由 migration 猜測資料。site_pending 與既有 profile_pending 語意不同（生日/身分證格式錯誤），
-- 刻意分開兩個 generated column，case_pending_status view 合併兩者供既有查詢共用。

ALTER TABLE cases
    ADD COLUMN site_id UUID REFERENCES sites(id) ON DELETE RESTRICT,
    ADD COLUMN site_name_raw TEXT;

-- 回填 1：交通偏好目前是既有的據點來源。
UPDATE cases c
SET site_id = p.site_id,
    site_name_raw = p.site_name_raw
FROM case_transport_preferences p
WHERE p.case_id = c.id
  AND (p.site_id IS NOT NULL OR p.site_name_raw IS NOT NULL);

-- 回填 2：交通偏好未提供時，補排班中最新生效版本的據點。
UPDATE cases c
SET site_id = s.site_id
FROM (
    SELECT DISTINCT ON (case_id) case_id, site_id
    FROM case_schedules
    ORDER BY case_id, lower(effective_range) DESC
) s
WHERE s.case_id = c.id
  AND c.site_id IS NULL
  AND c.site_name_raw IS NULL;

-- 兩者皆無據點來源的個案（含已軟刪除者），以哨兵值落入待維護清單由使用者補齊。
DO $$
DECLARE
    missing_count INT;
BEGIN
    SELECT count(*) INTO missing_count
    FROM cases
    WHERE site_id IS NULL AND site_name_raw IS NULL;

    RAISE NOTICE '據點回填後仍無據點來源的個案筆數：%（已標記待維護，需人工於待維護工作台補齊）', missing_count;
END $$;

UPDATE cases
SET site_name_raw = '（待補齊據點）'
WHERE site_id IS NULL AND site_name_raw IS NULL;

ALTER TABLE cases ADD CONSTRAINT ck_cases_site_present
    CHECK (site_id IS NOT NULL OR site_name_raw IS NOT NULL);

-- 據點比對不到主檔：全站隱藏，待使用者於待維護工作台手動補齊（與 link_pending 同一模式）。
ALTER TABLE cases ADD COLUMN site_pending BOOLEAN GENERATED ALWAYS AS (
    site_id IS NULL AND site_name_raw IS NOT NULL
) STORED;

CREATE INDEX ix_cases_site_id ON cases (site_id);
CREATE INDEX ix_cases_site_pending ON cases (id) WHERE site_pending;

DROP VIEW case_pending_status;

CREATE VIEW case_pending_status AS
SELECT c.id AS case_id,
       (c.profile_pending OR c.site_pending OR COALESCE(p.link_pending, false)) AS is_pending
FROM cases c
LEFT JOIN case_transport_preferences p ON p.case_id = c.id;
