-- Migration: 000012_case_service_type_nullable.down.sql
-- Description: 還原「服務類別」「服務使用類型」為 NOT NULL 並補回舊版預設值；
-- 還原前先把既有 NULL 值回填，避免違反 NOT NULL 約束。

UPDATE cases SET service_category = 1 WHERE service_category IS NULL;
UPDATE cases SET service_usage_type = 2 WHERE service_usage_type IS NULL;

ALTER TABLE cases DROP CONSTRAINT IF EXISTS cases_service_category_check;
ALTER TABLE cases ADD CONSTRAINT cases_service_category_check
    CHECK (service_category IN (1, 2));

ALTER TABLE cases DROP CONSTRAINT IF EXISTS cases_service_usage_type_check;
ALTER TABLE cases ADD CONSTRAINT cases_service_usage_type_check
    CHECK (service_usage_type IN (1, 2, 3, 4));

ALTER TABLE cases
    ALTER COLUMN service_category SET DEFAULT 1,
    ALTER COLUMN service_category SET NOT NULL,
    ALTER COLUMN service_usage_type SET DEFAULT 2,
    ALTER COLUMN service_usage_type SET NOT NULL;
