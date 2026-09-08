-- Migration: 000042_relax_site_and_driver_fields.down.sql
-- Description: 還原單位與司機必填欄位並移除司機擴充欄位。
-- 注意：若資料表已存在 NULL 值，還原 NOT NULL 會失敗；本 down migration
-- 僅適用於空表或測試環境，正式環境還原前需先清理或補齊資料。

ALTER TABLE drivers
    DROP COLUMN IF EXISTS gender,
    DROP COLUMN IF EXISTS birth_date,
    DROP COLUMN IF EXISTS has_professional_license,
    DROP COLUMN IF EXISTS employment_date,
    DROP COLUMN IF EXISTS has_transfer_cert,
    DROP COLUMN IF EXISTS inspection_date,
    DROP COLUMN IF EXISTS remarks;

UPDATE sites SET address = '' WHERE address IS NULL;
UPDATE sites SET region = '' WHERE region IS NULL;
UPDATE drivers SET region = '' WHERE region IS NULL;

ALTER TABLE sites
    ALTER COLUMN address SET NOT NULL,
    ALTER COLUMN region SET NOT NULL;

ALTER TABLE drivers
    ALTER COLUMN region SET NOT NULL;
