-- based_on_fingerprint 於 migration 000026 新增時未補 NOT NULL DEFAULT，既有列因此留 NULL。
-- 應用層一律以空字串代表「尚無更正依據」（app.RideRecord.BasedOnFingerprint 為 string，非
-- *string），查詢掃到 NULL 會直接失敗，讓匯入等寫入路徑整筆回滾。這裡先把既有 NULL 補齊，
-- 再補上約束避免同樣的資料狀態重新出現。
UPDATE ride_records SET based_on_fingerprint = '' WHERE based_on_fingerprint IS NULL;

ALTER TABLE ride_records ALTER COLUMN based_on_fingerprint SET DEFAULT '';
ALTER TABLE ride_records ALTER COLUMN based_on_fingerprint SET NOT NULL;
