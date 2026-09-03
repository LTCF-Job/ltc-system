-- 還原約束為搬遷前的列舉值；資料無法還原回 maintenance/retired/resigned，
-- 已收斂為 inactive 的資料列會保留 inactive（與 000015 down 的破壞性還原慣例一致）。

ALTER TABLE caregivers DROP COLUMN IF EXISTS status;

ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS vehicles_status_check;
ALTER TABLE vehicles ADD CONSTRAINT vehicles_status_check CHECK (status IN ('active', 'maintenance', 'retired'));

ALTER TABLE drivers DROP CONSTRAINT IF EXISTS drivers_status_check;
ALTER TABLE drivers ADD CONSTRAINT drivers_status_check CHECK (status IN ('active', 'resigned'));
