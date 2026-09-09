-- Migration: 000048_drop_region_master.up.sql
-- Description: 移除地區主檔（regions）與所有 region 外鍵關聯。
--   - sites.region 保留為使用者自由填寫的文字欄位，既有 code 回填成中文名稱。
--   - drivers / cases / holidays / export_jobs 的 region 欄位一併移除。
--   - 權限矩陣移除 masters_regions 模組。
--
-- 背景：地區管理功能下架後，區域不再是受控值域；各資料表自行保存文字即可。

-- 1. 先把 sites.region 由 regions.code 回填成顯示名稱，否則畫面會顯示 hsinchu 這類代碼。
--    regions 若已不存在（重複套用或既有環境已手動清理）就跳過。
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'regions')
       AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'regions' AND column_name = 'code')
    THEN
        UPDATE sites s SET region = r.name
        FROM regions r
        WHERE s.region = r.code AND r.name IS NOT NULL AND r.name <> '';
    END IF;
END $$;

-- 2. 移除五張業務表殘留的 region 外鍵。約束名以 000043 建立的命名為準，
--    但仍以查 pg_constraint 的方式移除，避免舊環境改過名而對不上。
DO $$
DECLARE r record;
BEGIN
    FOR r IN
        SELECT conrelid::regclass AS tbl, conname
        FROM pg_constraint
        WHERE contype = 'f'
          AND conrelid = ANY (ARRAY['sites', 'drivers', 'cases', 'export_jobs', 'holidays']::regclass[])
          AND pg_get_constraintdef(oid) ILIKE '%REFERENCES regions%'
    LOOP
        EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I', r.tbl, r.conname);
    END LOOP;
END $$;

-- 3. 移除不再使用的 region 欄位。sites.region 保留。
--    sites 的 (name, region) 唯一鍵維持不變：據點名稱仍可在不同區域重複。
ALTER TABLE drivers DROP COLUMN IF EXISTS region;
ALTER TABLE cases DROP COLUMN IF EXISTS region;
ALTER TABLE holidays DROP COLUMN IF EXISTS region;
ALTER TABLE export_jobs DROP COLUMN IF EXISTS region;
-- 匯入待裁決暫存列與匯出檔案快照的 region 隨個案申報區域一併移除。
ALTER TABLE case_import_duplicate_rows DROP COLUMN IF EXISTS region;
ALTER TABLE export_job_files DROP COLUMN IF EXISTS region;

-- 4. 移除地區主檔本體。seed（000002）寫入的 22 筆縣市一併消失。
DROP TABLE IF EXISTS regions;

-- 5. 權限矩陣移除 masters_regions；留著會讓角色設定頁出現沒有對應路由的模組。
UPDATE roles
SET permissions = permissions - 'masters_regions'
WHERE permissions ? 'masters_regions';
