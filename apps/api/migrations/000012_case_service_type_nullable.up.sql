-- Migration: 000012_case_service_type_nullable.up.sql
-- Description: 個案「服務類別」「服務使用類型」改為可留白，不再由後端靜默補上 1／2。
-- 未填寫時應在政府申報前置檢核報告中被抓出，交由使用者補齊，而非讓系統代填看似合理的資料。

ALTER TABLE cases
    ALTER COLUMN service_category DROP DEFAULT,
    ALTER COLUMN service_category DROP NOT NULL,
    ALTER COLUMN service_usage_type DROP DEFAULT,
    ALTER COLUMN service_usage_type DROP NOT NULL;

ALTER TABLE cases DROP CONSTRAINT IF EXISTS cases_service_category_check;
ALTER TABLE cases ADD CONSTRAINT cases_service_category_check
    CHECK (service_category IS NULL OR service_category IN (1, 2));

ALTER TABLE cases DROP CONSTRAINT IF EXISTS cases_service_usage_type_check;
ALTER TABLE cases ADD CONSTRAINT cases_service_usage_type_check
    CHECK (service_usage_type IS NULL OR service_usage_type IN (1, 2, 3, 4));
