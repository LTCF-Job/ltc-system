-- Migration: 000051_cases_caregiver.up.sql
-- Description: 個案主檔新增照護人員關聯（caregiver_id），關聯至 caregivers(id)。

ALTER TABLE cases
    ADD COLUMN caregiver_id UUID REFERENCES caregivers(id) ON DELETE RESTRICT;

CREATE INDEX idx_cases_caregiver_id ON cases(caregiver_id);
