-- 注意：down 不會自動清除已軟刪除的資料列。cases／vehicles／drivers 被其他表以
-- ON DELETE RESTRICT 參照，硬刪不可行；若軟刪除的資料與現存資料代碼重複，
-- 重建下方 UNIQUE 約束會失敗，需人工先處理重複資料再執行 down。

DROP INDEX IF EXISTS idx_cases_live;
DROP INDEX IF EXISTS idx_vehicles_live;
DROP INDEX IF EXISTS idx_drivers_live;

DROP INDEX IF EXISTS uq_cases_code_live;
DROP INDEX IF EXISTS uq_cases_national_id_hmac_live;
ALTER TABLE cases ADD CONSTRAINT cases_code_key UNIQUE (code);
ALTER TABLE cases ADD CONSTRAINT cases_national_id_hmac_key UNIQUE (national_id_hmac);

DROP INDEX IF EXISTS uq_vehicles_plate_no_live;
DROP INDEX IF EXISTS uq_vehicles_display_name_live;
ALTER TABLE vehicles ADD CONSTRAINT vehicles_plate_no_key UNIQUE (plate_no);
ALTER TABLE vehicles ADD CONSTRAINT vehicles_display_name_key UNIQUE (display_name);

DROP INDEX IF EXISTS uq_drivers_code_live;
DROP INDEX IF EXISTS uq_drivers_national_id_hmac_live;
ALTER TABLE drivers ADD CONSTRAINT drivers_code_key UNIQUE (code);
ALTER TABLE drivers ADD CONSTRAINT drivers_national_id_hmac_key UNIQUE (national_id_hmac);

ALTER TABLE cases DROP COLUMN deleted_at;
ALTER TABLE cases DROP COLUMN deleted_by;
ALTER TABLE vehicles DROP COLUMN deleted_at;
ALTER TABLE vehicles DROP COLUMN deleted_by;
ALTER TABLE drivers DROP COLUMN deleted_at;
ALTER TABLE drivers DROP COLUMN deleted_by;
