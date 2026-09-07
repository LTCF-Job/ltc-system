-- Migration: 000041_pending_flag_columns.up.sql
-- Description: 「待維護」原本只有 CaseRepository.List 的 WHERE 子句認得，其餘讀取個案的查詢
-- （排班日曆、報表、政府申報、司機匯報比對、個案匯出）各自繞過，導致待維護資料外洩到其他頁面。
-- 這裡把判定下沉到資料庫成為單一來源：個案本體與交通偏好各自產生一個 generated column，
-- 再由 case_pending_status 這個 view 合併成一個 is_pending 供所有查詢共用。單位／車輛的原始名稱
-- 欄位位於 case_transport_preferences，因此無法只用 cases 上的單一欄位涵蓋。

-- 生日格式錯誤或身分證字號格式錯誤：條件與既有 List 查詢完全一致（IS NOT NULL / = true）。
ALTER TABLE cases ADD COLUMN profile_pending BOOLEAN GENERATED ALWAYS AS (
    birth_date_raw IS NOT NULL OR national_id_invalid
) STORED;

-- 單位／去程車輛／回程車輛比對不到主檔：同樣沿用既有 List 查詢的條件。
ALTER TABLE case_transport_preferences ADD COLUMN link_pending BOOLEAN GENERATED ALWAYS AS (
    (site_id IS NULL AND site_name_raw IS NOT NULL)
 OR (outbound_vehicle_id IS NULL AND outbound_vehicle_name_raw IS NOT NULL)
 OR (inbound_vehicle_id IS NULL AND inbound_vehicle_name_raw IS NOT NULL)
) STORED;

-- 照護人員的待維護判定改為「姓名或類型未填寫」；單位比對不到不再列入待維護。
-- btrim 讓手動 API 與既有資料的空白值，與匯入端的 TrimSpace 判定一致。
ALTER TABLE caregivers ADD COLUMN is_pending BOOLEAN GENERATED ALWAYS AS (
    btrim(name) = '' OR btrim(type) = ''
) STORED;

-- 沒有交通偏好列的個案視為連結無待維護，因此用 COALESCE 收斂 LEFT JOIN 的 NULL。
CREATE VIEW case_pending_status AS
SELECT c.id AS case_id,
       (c.profile_pending OR COALESCE(p.link_pending, false)) AS is_pending
FROM cases c
LEFT JOIN case_transport_preferences p ON p.case_id = c.id;

CREATE INDEX ix_cases_profile_pending ON cases (id) WHERE profile_pending;
CREATE INDEX ix_case_transport_preferences_link_pending ON case_transport_preferences (case_id) WHERE link_pending;
CREATE INDEX ix_caregivers_is_pending ON caregivers (id) WHERE is_pending;
