-- 主檔軟刪除：cases／vehicles／drivers 三張表加 deleted_at／deleted_by，
-- 並把欄位級 UNIQUE 改為 partial unique index，讓軟刪後同一身分證字號／車牌／代碼可重建。

ALTER TABLE cases ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE cases ADD COLUMN deleted_by UUID;
ALTER TABLE vehicles ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE vehicles ADD COLUMN deleted_by UUID;
ALTER TABLE drivers ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE drivers ADD COLUMN deleted_by UUID;

ALTER TABLE cases DROP CONSTRAINT IF EXISTS cases_code_key;
ALTER TABLE cases DROP CONSTRAINT IF EXISTS cases_national_id_hmac_key;
CREATE UNIQUE INDEX uq_cases_code_live ON cases (code) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_cases_national_id_hmac_live ON cases (national_id_hmac) WHERE deleted_at IS NULL;

ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS vehicles_plate_no_key;
ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS vehicles_display_name_key;
CREATE UNIQUE INDEX uq_vehicles_plate_no_live ON vehicles (plate_no) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_vehicles_display_name_live ON vehicles (display_name) WHERE deleted_at IS NULL;

ALTER TABLE drivers DROP CONSTRAINT IF EXISTS drivers_code_key;
ALTER TABLE drivers DROP CONSTRAINT IF EXISTS drivers_national_id_hmac_key;
CREATE UNIQUE INDEX uq_drivers_code_live ON drivers (code) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_drivers_national_id_hmac_live ON drivers (national_id_hmac) WHERE deleted_at IS NULL;

CREATE INDEX idx_cases_live ON cases (id) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicles_live ON vehicles (id) WHERE deleted_at IS NULL;
CREATE INDEX idx_drivers_live ON drivers (id) WHERE deleted_at IS NULL;
