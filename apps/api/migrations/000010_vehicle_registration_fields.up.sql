-- 車輛主檔改以「所屬單位」歸屬（原本只有區域，無法對應到實際服務單位），並補齊
-- 服務車輛清冊需要的車籍、保險與檢驗欄位。

ALTER TABLE vehicles
    ADD COLUMN site_id UUID REFERENCES sites(id) ON DELETE RESTRICT,
    ADD COLUMN owner_name TEXT,
    ADD COLUMN brand TEXT,
    ADD COLUMN model TEXT,
    -- 清冊只到年月，格式固定為 YYYY-MM
    ADD COLUMN manufacture_ym TEXT,
    ADD COLUMN compulsory_insurance_expiry DATE,
    ADD COLUMN passenger_insurance_expiry DATE,
    ADD COLUMN third_party_insurance_expiry DATE,
    ADD COLUMN last_inspection_date DATE,
    ADD COLUMN wheelchair_accessible BOOLEAN;

ALTER TABLE vehicles
    ADD CONSTRAINT ck_vehicle_manufacture_ym CHECK (manufacture_ym IS NULL OR manufacture_ym ~ '^\d{4}-(0[1-9]|1[0-2])$');

CREATE INDEX idx_vehicles_site ON vehicles(site_id);

-- 車輛的區域改由所屬單位帶出，不再自存。既有資料沒有可靠的單位對應來源，
-- site_id 一律留空，由使用者在車輛管理逐台補齊。
ALTER TABLE vehicles DROP COLUMN region;
