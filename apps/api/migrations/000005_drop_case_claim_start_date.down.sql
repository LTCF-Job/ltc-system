-- Migration: 000005_drop_case_claim_start_date.down.sql
-- Description: 還原個案起聘申報日欄位。

ALTER TABLE cases ADD COLUMN claim_start_date DATE;
ALTER TABLE cases ADD CONSTRAINT chk_claim_date_range CHECK (claim_end_date IS NULL OR claim_start_date IS NULL OR claim_end_date >= claim_start_date);
