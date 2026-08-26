-- Migration: 000004_create_regions.up.sql
-- Description: 建立 regions 區域資料表、放寬舊表 region 約束，並預載全台灣 22 縣市種子資料

CREATE TABLE IF NOT EXISTS regions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 放寬舊有資料表上的靜態 region CHECK 約束
ALTER TABLE sites DROP CONSTRAINT IF EXISTS sites_region_check;
ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS vehicles_region_check;
ALTER TABLE drivers DROP CONSTRAINT IF EXISTS drivers_region_check;
ALTER TABLE cases DROP CONSTRAINT IF EXISTS cases_region_check;
ALTER TABLE export_jobs DROP CONSTRAINT IF EXISTS export_jobs_region_check;
ALTER TABLE holidays DROP CONSTRAINT IF EXISTS holidays_region_check;

-- 預先載入台灣 22 縣市種子資料（包含既有使用的 hsinchu 與 miaoli 代碼）
INSERT INTO regions (code, name, description, status, sort_order) VALUES
('hsinchu', '新竹縣', '新竹縣營運區域', 'active', 1),
('hsinchu_city', '新竹市', '新竹市營運區域', 'active', 2),
('miaoli', '苗栗縣', '苗栗縣營運區域', 'active', 3),
('taipei', '臺北市', '臺北市營運區域', 'active', 4),
('new_taipei', '新北市', '新北市營運區域', 'active', 5),
('keelung', '基隆市', '基隆市營運區域', 'active', 6),
('taoyuan', '桃園市', '桃園市營運區域', 'active', 7),
('taichung', '臺中市', '臺中市營運區域', 'active', 8),
('changhua', '彰化縣', '彰化縣營運區域', 'active', 9),
('nantou', '南投縣', '南投縣營運區域', 'active', 10),
('yunlin', '雲林縣', '雲林縣營運區域', 'active', 11),
('chiayi_city', '嘉義市', '嘉義市營運區域', 'active', 12),
('chiayi', '嘉義縣', '嘉義縣營運區域', 'active', 13),
('tainan', '臺南市', '臺南市營運區域', 'active', 14),
('kaohsiung', '高雄市', '高雄市營運區域', 'active', 15),
('pingtung', '屏東縣', '屏東縣營運區域', 'active', 16),
('yilan', '宜蘭縣', '宜蘭縣營運區域', 'active', 17),
('hualien', '花蓮縣', '花蓮縣營運區域', 'active', 18),
('taitung', '臺東縣', '臺東縣營運區域', 'active', 19),
('penghu', '澎湖縣', '澎湖縣營運區域', 'active', 20),
('kinmen', '金門縣', '金門縣營運區域', 'active', 21),
('lienchiang', '連江縣', '連江縣營運區域', 'active', 22)
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  sort_order = EXCLUDED.sort_order;
