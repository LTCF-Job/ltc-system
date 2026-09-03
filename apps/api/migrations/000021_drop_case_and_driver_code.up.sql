-- 徹底移除個案與司機的 code 欄位，以及 export_job_files 的 case_code 欄位，
-- 避免前端未輸入 code 時產生空字串重複違反局部唯一約束。

DROP INDEX IF EXISTS uq_cases_code_live;
ALTER TABLE cases DROP COLUMN IF EXISTS code;

DROP INDEX IF EXISTS uq_drivers_code_live;
ALTER TABLE drivers DROP COLUMN IF EXISTS code;

ALTER TABLE export_job_files DROP COLUMN IF EXISTS case_code;
