-- Migration: 000003_relax_case_required_fields.up.sql
-- Description: 個案除姓名外全欄位改為選填、移除身分證字號唯一鍵；新增個案備註與
-- 接送車輛/據點比對不到時的原始名稱保留欄位。

ALTER TABLE cases
    ALTER COLUMN national_id_cipher DROP NOT NULL,
    ALTER COLUMN national_id_hmac DROP NOT NULL,
    ALTER COLUMN national_id_masked DROP NOT NULL,
    ALTER COLUMN home_address DROP NOT NULL,
    ALTER COLUMN region DROP NOT NULL,
    ALTER COLUMN claim_start_date DROP NOT NULL;

ALTER TABLE cases DROP CONSTRAINT IF EXISTS cases_national_id_hmac_key;

ALTER TABLE cases ADD COLUMN remarks TEXT;

ALTER TABLE case_transport_preferences
    ADD COLUMN site_name_raw TEXT,
    ADD COLUMN outbound_vehicle_name_raw TEXT,
    ADD COLUMN inbound_vehicle_name_raw TEXT;
