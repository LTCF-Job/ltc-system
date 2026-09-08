-- Migration: 000048_drop_region_master.down.sql
-- Description: 還原地區主檔與 region 欄位。
--
-- 限制：drivers / cases / holidays / export_jobs 的原始 region 值在 up 已被刪除，
-- 無法還原，欄位一律回填為 NULL（export_jobs 的 NULL 代表全部地區）。
-- sites.region 在 up 被改寫成中文名稱，這裡改回 regions.code；對不上的自訂值保留原樣。

-- 1. 重建地區主檔（欄位比照 000001 + 000043 的終態）。
CREATE TABLE IF NOT EXISTS regions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO regions (code, name, sort_order)
VALUES
    ('taipei', '臺北市', 1), ('new_taipei', '新北市', 2), ('keelung', '基隆市', 3),
    ('taoyuan', '桃園市', 4), ('hsinchu_city', '新竹市', 5), ('hsinchu', '新竹縣', 6),
    ('miaoli', '苗栗縣', 7), ('taichung', '臺中市', 8), ('changhua', '彰化縣', 9),
    ('nantou', '南投縣', 10), ('yunlin', '雲林縣', 11), ('chiayi_city', '嘉義市', 12),
    ('chiayi', '嘉義縣', 13), ('tainan', '臺南市', 14), ('kaohsiung', '高雄市', 15),
    ('pingtung', '屏東縣', 16), ('yilan', '宜蘭縣', 17), ('hualien', '花蓮縣', 18),
    ('taitung', '臺東縣', 19), ('penghu', '澎湖縣', 20), ('kinmen', '金門縣', 21),
    ('lienchiang', '連江縣', 22)
ON CONFLICT (code) DO NOTHING;

-- 2. 還原被移除的 region 欄位（值無法復原）。
ALTER TABLE drivers ADD COLUMN IF NOT EXISTS region TEXT;
ALTER TABLE cases ADD COLUMN IF NOT EXISTS region TEXT;
ALTER TABLE holidays ADD COLUMN IF NOT EXISTS region TEXT;
ALTER TABLE export_jobs ADD COLUMN IF NOT EXISTS region TEXT;
ALTER TABLE case_import_duplicate_rows ADD COLUMN IF NOT EXISTS region TEXT;
-- export_job_files.region 原為 NOT NULL，值已無從復原，故以空字串回填。
ALTER TABLE export_job_files ADD COLUMN IF NOT EXISTS region TEXT NOT NULL DEFAULT '';

-- 3. sites.region 由中文名稱改回 code；自由填寫、對不到主檔的值先清成 NULL，
--    否則第 4 步建外鍵會失敗。
UPDATE sites s SET region = r.code FROM regions r WHERE s.region = r.name;
UPDATE sites SET region = NULL
WHERE region IS NOT NULL AND region NOT IN (SELECT code FROM regions);

-- 4. 重建五張表的 region 外鍵。
DO $$
DECLARE t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['sites', 'drivers', 'cases', 'export_jobs', 'holidays']
    LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conname = t || '_region_fkey' AND conrelid = t::regclass
        ) THEN
            EXECUTE format(
                'ALTER TABLE %I ADD CONSTRAINT %I FOREIGN KEY (region) REFERENCES regions(code)',
                t, t || '_region_fkey'
            );
        END IF;
    END LOOP;
END $$;

-- 5. 權限矩陣補回 masters_regions，軸向比照 000018 的既有規則。
UPDATE roles
SET permissions = permissions || jsonb_build_object(
    'masters_regions',
    jsonb_build_object(
        'view', true,
        'edit', base_role IN ('admin', 'staff'),
        'delete', base_role = 'admin'
    )
)
WHERE NOT (permissions ? 'masters_regions');
