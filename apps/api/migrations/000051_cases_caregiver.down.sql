-- Migration: 000051_cases_caregiver.down.sql
-- Description: 回滾個案主檔的照護人員關聯（caregiver_id）。

DROP INDEX IF EXISTS idx_cases_caregiver_id;

ALTER TABLE cases
    DROP COLUMN IF EXISTS caregiver_id;
