-- Migration: 000003_relax_case_required_fields.down.sql
-- Description: 還原個案必填欄位與身分證字號唯一鍵。
-- 注意：若資料表已存在 NULL 值，還原 NOT NULL 會失敗；本 down migration
-- 僅適用於空表或測試環境，正式環境還原前需先清理或補齊資料。

ALTER TABLE case_transport_preferences
    DROP COLUMN IF EXISTS site_name_raw,
    DROP COLUMN IF EXISTS outbound_vehicle_name_raw,
    DROP COLUMN IF EXISTS inbound_vehicle_name_raw;

ALTER TABLE cases DROP COLUMN IF EXISTS remarks;

ALTER TABLE cases ADD CONSTRAINT cases_national_id_hmac_key UNIQUE (national_id_hmac);

ALTER TABLE cases
    ALTER COLUMN national_id_cipher SET NOT NULL,
    ALTER COLUMN national_id_hmac SET NOT NULL,
    ALTER COLUMN national_id_masked SET NOT NULL,
    ALTER COLUMN home_address SET NOT NULL,
    ALTER COLUMN region SET NOT NULL,
    ALTER COLUMN claim_start_date SET NOT NULL;
