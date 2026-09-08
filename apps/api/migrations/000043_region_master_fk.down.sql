-- Migration: 000043_region_master_fk.down.sql
-- Description: 移除 region 外鍵與 regions.code，還原 export_jobs.region 的 NOT NULL。
--
-- 注意：不重建原本的 region CHECK。原 CHECK 只允許 miaoli／hsinchu，
-- 而本 migration 之後主檔可新增其他地區並被業務資料引用，重建會直接失敗。
-- 回滾後 region 值域改由應用層負責，資料本身不受影響。

DO $$
DECLARE t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['sites', 'drivers', 'cases', 'export_jobs', 'holidays']
    LOOP
        EXECUTE format('ALTER TABLE %I DROP CONSTRAINT IF EXISTS %I', t, t || '_region_fkey');
    END LOOP;
END $$;

ALTER TABLE regions DROP CONSTRAINT IF EXISTS regions_code_key;
ALTER TABLE regions DROP COLUMN IF EXISTS code;

-- export_jobs.region 還原 NOT NULL 前，把代表「全部地區」的 NULL 寫回空字串。
UPDATE export_jobs SET region = '' WHERE region IS NULL;
ALTER TABLE export_jobs ALTER COLUMN region SET NOT NULL;
