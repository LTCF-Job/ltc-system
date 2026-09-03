-- 統一主檔啟用/停用狀態：車輛與司機的狀態欄位收斂為 active/inactive，
-- 與 sites／regions 既有慣例一致；照護人員新增 status 欄位補齊同一套設計。

-- 回填必須在收緊 CHECK 約束之前執行，此時舊約束仍允許 maintenance/retired/resigned。
UPDATE vehicles SET status = 'inactive' WHERE status IN ('maintenance', 'retired');
UPDATE drivers SET status = 'inactive' WHERE status = 'resigned';

ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS vehicles_status_check;
ALTER TABLE vehicles ADD CONSTRAINT vehicles_status_check CHECK (status IN ('active', 'inactive'));

ALTER TABLE drivers DROP CONSTRAINT IF EXISTS drivers_status_check;
ALTER TABLE drivers ADD CONSTRAINT drivers_status_check CHECK (status IN ('active', 'inactive'));

ALTER TABLE caregivers ADD COLUMN status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive'));
