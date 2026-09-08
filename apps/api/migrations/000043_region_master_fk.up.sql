-- Migration: 000043_region_master_fk.up.sql
-- Description: 收斂 region 值域 —— 移除殘留的靜態 region CHECK、regions 新增 code 欄位並回填，
-- 五張業務表的 region 改以外鍵參照 regions(code)，export_jobs.region 放寬為可空（NULL 代表全部地區）。
--
-- 背景：commit 4fc88e2 改寫了已套用的 000001 並重新編號 000002–000007，
-- 造成 4fc88e2 之前建立的資料庫仍帶著 region CHECK，而用現在的 migration 從零建的資料庫沒有。
-- 本 migration 以查 pg_constraint 的方式收斂，新舊環境跑起來都是同一個終態。

-- 1. 移除所有殘留的 region CHECK。約束名由 PostgreSQL 自動產生，
-- 舊環境若曾手動改名會對不上，因此不寫死名稱。
DO $$
DECLARE r record;
BEGIN
    FOR r IN
        SELECT conrelid::regclass AS tbl, conname
        FROM pg_constraint
        WHERE contype = 'c'
          AND conrelid = ANY (ARRAY['sites', 'vehicles', 'drivers', 'cases', 'export_jobs', 'holidays']::regclass[])
          AND pg_get_constraintdef(oid) ILIKE '%region%'
    LOOP
        EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I', r.tbl, r.conname);
    END LOOP;
END $$;

-- 2. regions 新增 code 欄位。外鍵目標必須是 UNIQUE 約束，因此 code 不可為 NULL。
ALTER TABLE regions ADD COLUMN IF NOT EXISTS code TEXT;

UPDATE regions AS r SET code = m.code
FROM (VALUES
    ('新竹縣', 'hsinchu'),
    ('新竹市', 'hsinchu_city'),
    ('苗栗縣', 'miaoli'),
    ('臺北市', 'taipei'),
    ('新北市', 'new_taipei'),
    ('基隆市', 'keelung'),
    ('桃園市', 'taoyuan'),
    ('臺中市', 'taichung'),
    ('彰化縣', 'changhua'),
    ('南投縣', 'nantou'),
    ('雲林縣', 'yunlin'),
    ('嘉義市', 'chiayi_city'),
    ('嘉義縣', 'chiayi'),
    ('臺南市', 'tainan'),
    ('高雄市', 'kaohsiung'),
    ('屏東縣', 'pingtung'),
    ('宜蘭縣', 'yilan'),
    ('花蓮縣', 'hualien'),
    ('臺東縣', 'taitung'),
    ('澎湖縣', 'penghu'),
    ('金門縣', 'kinmen'),
    ('連江縣', 'lienchiang')
) AS m(name, code)
WHERE r.name = m.name AND r.code IS NULL;

-- 自訂地區沒有固定對照，以 id 前 8 碼產生穩定且不會衝突的 code。
UPDATE regions
SET code = 'region_' || left(replace(id::text, '-', ''), 8)
WHERE code IS NULL;

ALTER TABLE regions ALTER COLUMN code SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'regions_code_key' AND conrelid = 'regions'::regclass
    ) THEN
        ALTER TABLE regions ADD CONSTRAINT regions_code_key UNIQUE (code);
    END IF;
END $$;

-- 3. 空字串不是合法的 region，會讓外鍵建立失敗；統一正規化成 NULL。
UPDATE sites SET region = NULL WHERE region = '';
UPDATE drivers SET region = NULL WHERE region = '';
UPDATE cases SET region = NULL WHERE region = '';
UPDATE holidays SET region = NULL WHERE region = '';

-- 4. export_jobs.region 放寬為可空：NULL 代表「全部地區」，
-- 這是應用層既有語意（ClaimScope 把空值視為不限區域），先前只能靠空字串表達而撞上 NOT NULL 與 CHECK。
-- DROP NOT NULL 必須先做：欄位仍是 NOT NULL 時把空字串更新成 NULL 會直接失敗。
ALTER TABLE export_jobs ALTER COLUMN region DROP NOT NULL;
UPDATE export_jobs SET region = NULL WHERE region = '';

-- 5. 業務資料表若含主檔沒有的 region，直接建外鍵會中止 migration 且留下
-- 「CHECK 已移除、FK 尚未建立」的中間態。先把孤兒值列出來讓失敗訊息可診斷。
DO $$
DECLARE orphans text;
BEGIN
    SELECT string_agg(DISTINCT tbl || '.' || region, ', ') INTO orphans
    FROM (
        SELECT 'sites' AS tbl, region FROM sites WHERE region IS NOT NULL
        UNION ALL SELECT 'drivers', region FROM drivers WHERE region IS NOT NULL
        UNION ALL SELECT 'cases', region FROM cases WHERE region IS NOT NULL
        UNION ALL SELECT 'export_jobs', region FROM export_jobs WHERE region IS NOT NULL
        UNION ALL SELECT 'holidays', region FROM holidays WHERE region IS NOT NULL
    ) used
    WHERE NOT EXISTS (SELECT 1 FROM regions r WHERE r.code = used.region);

    IF orphans IS NOT NULL THEN
        RAISE EXCEPTION '業務資料表含地區主檔沒有的 region 值，請先在地區管理補齊或修正後再套用：%', orphans;
    END IF;
END $$;

-- 6. 業務表的 region 改以外鍵參照 regions(code)，值域自此由地區主檔決定。
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
