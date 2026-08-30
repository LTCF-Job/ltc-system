-- Migration: 000005_drop_case_claim_start_date.up.sql
-- Description: 移除個案起聘申報日欄位。排班的生效區間已由 case_schedules
-- 逐月維護，claim_start_date 對趟次產生不再有實質作用。

ALTER TABLE cases DROP CONSTRAINT IF EXISTS chk_claim_date_range;
ALTER TABLE cases DROP COLUMN IF EXISTS claim_start_date;
