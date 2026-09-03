ALTER TABLE cases ADD COLUMN code TEXT;
CREATE UNIQUE INDEX uq_cases_code_live ON cases (code) WHERE deleted_at IS NULL;

ALTER TABLE drivers ADD COLUMN code TEXT;
CREATE UNIQUE INDEX uq_drivers_code_live ON drivers (code) WHERE deleted_at IS NULL;

ALTER TABLE export_job_files ADD COLUMN case_code TEXT;
