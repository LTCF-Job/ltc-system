-- Seed: 0001_baseline.up.sql
-- Purpose: Demo data-plane（ltc_demo）基準資料集。虛構的臺灣長照交通接送測試資料，
-- 涵蓋個案、排班、車輛、司機、接送、匯入、通知、出勤、維修、報表/稽核等模組，
-- 供 Demo 環境重置時還原一致的基準狀態。
--
-- 規則：
--   * 全部使用固定 UUID，搭配 ON CONFLICT 讓本檔案可重複執行（idempotent）。
--   * 不寫入 auth.* — Demo 測試帳號由共用 Supabase Auth 資料庫另外建立。
--   * 不重複寫入 regions — 已由 000002_seed_reference_data 對所有環境一致播種。
--   * national_id_cipher / national_id_hmac 為 AES-256-GCM 加密與 HMAC 索引，
--     只有執行中的 Go 應用程式持有加密金鑰才能產生真正的密文。這裡改用可讀的
--     佔位位元組（demo-cipher-<code> / demo-hmac-<code> 轉成的 bytea），僅滿足
--     NOT NULL / UNIQUE 與畫面遮罩顯示需求；依賴解密明文的政府申報匯出功能在
--     Demo 環境下無法還原真實身分證字號，此為刻意簡化。

-- 系統管理員 UUID（對應 000002 補種的 Supabase Auth 帳號），僅作為 created_by /
-- actor_id 等稽核追蹤欄位的示範值，這些欄位皆為裸 UUID，未對 auth.users 建 FK。
-- ltc_demo 若沒有掛載共用 Auth 資料庫，此 UUID 純粹是佔位值。

-- ============================================================
-- 1. 單位 (sites)
-- ============================================================
INSERT INTO sites (id, name, address, region, open_days, status) VALUES
('10000000-0000-4000-8000-000000000001', '新竹縣站', '新竹縣竹北市光明六路100號', '新竹縣', '{1,2,3,4,5}', 'active'),
('10000000-0000-4000-8000-000000000002', '苗栗縣站', '苗栗縣苗栗市自治路50號', '苗栗縣', '{1,2,3,4,5}', 'active'),
('10000000-0000-4000-8000-000000000003', '新竹市站', '新竹市東區中央路200號', '新竹市', '{1,2,3,4,5}', 'active')
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name, address = EXCLUDED.address,
  region = EXCLUDED.region, open_days = EXCLUDED.open_days, status = EXCLUDED.status;

-- ============================================================
-- 2. 車輛 (vehicles)
-- ============================================================
INSERT INTO vehicles (id, plate_no, display_name, status, site_id, brand, model, manufacture_ym,
    compulsory_insurance_expiry, passenger_insurance_expiry, third_party_insurance_expiry,
    last_inspection_date, wheelchair_accessible) VALUES
('20000000-0000-4000-8000-000000000001', 'AAA-1688', '新竹一號車', 'active', '10000000-0000-4000-8000-000000000001', '豐田', 'Hiace', '2021-05', '2027-05-01', '2027-05-01', '2027-05-01', '2026-03-15', true),
('20000000-0000-4000-8000-000000000002', 'AAB-2288', '新竹二號車', 'active', '10000000-0000-4000-8000-000000000001', '福特', 'Transit', '2020-09', '2027-02-01', '2027-02-01', '2027-02-01', '2026-02-20', false),
('20000000-0000-4000-8000-000000000003', 'ABC-3399', '苗栗一號車', 'active', '10000000-0000-4000-8000-000000000002', '豐田', 'Hiace', '2022-03', '2027-08-01', '2027-08-01', '2027-08-01', '2026-04-10', true),
('20000000-0000-4000-8000-000000000004', 'ABD-4410', '苗栗二號車', 'maintenance', '10000000-0000-4000-8000-000000000002', '日產', 'NV350', '2019-11', '2026-11-01', '2026-11-01', '2026-11-01', '2025-12-01', false),
('20000000-0000-4000-8000-000000000005', 'ABE-5521', '竹市一號車', 'active', '10000000-0000-4000-8000-000000000003', '豐田', 'Hiace', '2023-01', '2028-01-01', '2028-01-01', '2028-01-01', '2026-06-01', true),
('20000000-0000-4000-8000-000000000006', 'OLD-9001', '除役備用車', 'retired', NULL, '福特', 'Transit', '2015-06', NULL, NULL, NULL, NULL, false)
ON CONFLICT (id) DO UPDATE SET
  plate_no = EXCLUDED.plate_no, display_name = EXCLUDED.display_name, status = EXCLUDED.status,
  site_id = EXCLUDED.site_id, brand = EXCLUDED.brand, model = EXCLUDED.model,
  manufacture_ym = EXCLUDED.manufacture_ym,
  compulsory_insurance_expiry = EXCLUDED.compulsory_insurance_expiry,
  passenger_insurance_expiry = EXCLUDED.passenger_insurance_expiry,
  third_party_insurance_expiry = EXCLUDED.third_party_insurance_expiry,
  last_inspection_date = EXCLUDED.last_inspection_date,
  wheelchair_accessible = EXCLUDED.wheelchair_accessible;

-- ============================================================
-- 3. 司機 (drivers)
-- ============================================================
INSERT INTO drivers (id, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked,
    email, region, status, license_class, license_expiry_date) VALUES
('30000000-0000-4000-8000-000000000001', '林建成', '林建成', convert_to('demo-cipher-DRV-001','UTF8'), convert_to('demo-hmac-DRV-001','UTF8'), 'A12***0001', 'driver001@ltc.example.com', '新竹縣', 'active', 'sedan', '2028-06-30'),
('30000000-0000-4000-8000-000000000002', '陳美惠', '陳美惠', convert_to('demo-cipher-DRV-002','UTF8'), convert_to('demo-hmac-DRV-002','UTF8'), 'A22***0002', 'driver002@ltc.example.com', '新竹縣', 'active', 'bus', '2027-11-20'),
('30000000-0000-4000-8000-000000000003', '黃志豪', '黃志豪', convert_to('demo-cipher-DRV-003','UTF8'), convert_to('demo-hmac-DRV-003','UTF8'), 'A32***0003', 'driver003@ltc.example.com', '苗栗縣', 'active', 'sedan', '2026-12-01'),
('30000000-0000-4000-8000-000000000004', '張淑芬', '張淑芬', convert_to('demo-cipher-DRV-004','UTF8'), convert_to('demo-hmac-DRV-004','UTF8'), 'A42***0004', 'driver004@ltc.example.com', '苗栗縣', 'active', NULL, NULL),
('30000000-0000-4000-8000-000000000005', '吳國棟', '吳國棟', convert_to('demo-cipher-DRV-005','UTF8'), convert_to('demo-hmac-DRV-005','UTF8'), 'A52***0005', 'driver005@ltc.example.com', '新竹市', 'resigned', 'sedan', '2025-01-01')
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name, name_normalized = EXCLUDED.name_normalized,
  national_id_cipher = EXCLUDED.national_id_cipher, national_id_hmac = EXCLUDED.national_id_hmac,
  national_id_masked = EXCLUDED.national_id_masked, email = EXCLUDED.email, region = EXCLUDED.region,
  status = EXCLUDED.status, license_class = EXCLUDED.license_class, license_expiry_date = EXCLUDED.license_expiry_date;

-- ============================================================
-- 4. 司機車輛指派 (driver_assignments)
-- 示範一車多司機（v3 由 d3、d4 同期共用）與同一司機的歷史交接（d3：v3 舊/新兩段）。
-- ============================================================
INSERT INTO driver_assignments (id, driver_id, vehicle_id, effective_range) VALUES
('31000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000001', '[2024-01-01,)'),
('31000000-0000-4000-8000-000000000002', '30000000-0000-4000-8000-000000000002', '20000000-0000-4000-8000-000000000002', '[2024-01-01,)'),
('31000000-0000-4000-8000-000000000003', '30000000-0000-4000-8000-000000000003', '20000000-0000-4000-8000-000000000003', '[2024-01-01,2026-06-01)'),
('31000000-0000-4000-8000-000000000004', '30000000-0000-4000-8000-000000000003', '20000000-0000-4000-8000-000000000003', '[2026-06-01,)'),
('31000000-0000-4000-8000-000000000005', '30000000-0000-4000-8000-000000000004', '20000000-0000-4000-8000-000000000003', '[2025-01-01,)'),
('31000000-0000-4000-8000-000000000006', '30000000-0000-4000-8000-000000000005', '20000000-0000-4000-8000-000000000005', '[2023-01-01,2025-06-01)')
ON CONFLICT (id) DO UPDATE SET
  driver_id = EXCLUDED.driver_id, vehicle_id = EXCLUDED.vehicle_id, effective_range = EXCLUDED.effective_range;

-- ============================================================
-- 5. 個案 (cases)
-- ============================================================
INSERT INTO cases (id, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked,
    home_address, region, ltc_level, service_category, service_usage_type, claim_end_date, status,
    household_type, gender, birth_date, care_contact_role, care_contact_name, registered_address, remarks) VALUES
('40000000-0000-4000-8000-000000000001', '王秀琴', '王秀琴', convert_to('demo-cipher-C-0001','UTF8'), convert_to('demo-hmac-C-0001','UTF8'), 'A12***0001', '新竹縣竹北市文興路一段1號', '新竹縣', 'CMS 2', 1, 2, NULL, 'active', '一般戶', 'F', '1948-03-12', '女兒', '王小美', '新竹縣竹北市文興路一段1號', NULL),
('40000000-0000-4000-8000-000000000002', '李國華', '李國華', convert_to('demo-cipher-C-0002','UTF8'), convert_to('demo-hmac-C-0002','UTF8'), 'A22***0002', '新竹縣竹北市中正西路20號', '新竹縣', 'CMS 4', 2, 1, NULL, 'active', '中低收入戶', 'M', '1955-07-22', '兒子', '李文彬', '新竹縣竹北市中正西路20號', NULL),
('40000000-0000-4000-8000-000000000003', '張阿蘭', '張阿蘭', convert_to('demo-cipher-C-0003','UTF8'), convert_to('demo-hmac-C-0003','UTF8'), 'A32***0003', '苗栗縣苗栗市中山路30號', '苗栗縣', 'CMS 3', 1, 3, NULL, 'active', '一般戶', 'F', '1950-11-05', '媳婦', '張林月', '苗栗縣苗栗市中山路30號', NULL),
('40000000-0000-4000-8000-000000000004', '陳福來', '陳福來', convert_to('demo-cipher-C-0004','UTF8'), convert_to('demo-hmac-C-0004','UTF8'), 'A42***0004', '苗栗縣頭份市中央路40號', '苗栗縣', 'CMS 3', 1, 2, NULL, 'suspended', '一般戶', 'M', '1952-02-18', '女兒', '陳小玉', '苗栗縣頭份市中央路40號', '暫停服務：住院治療中'),
('40000000-0000-4000-8000-000000000005', '劉月娥', '劉月娥', convert_to('demo-cipher-C-0005','UTF8'), convert_to('demo-hmac-C-0005','UTF8'), 'A52***0005', '新竹市東區中央路50號', '新竹市', 'CMS 2', 2, 4, '2026-06-30', 'closed', '一般戶', 'F', '1945-09-30', '配偶', '劉大山', '新竹市東區中央路50號', '個案已轉住機構，結案'),
('40000000-0000-4000-8000-000000000006', '許進財', '許進財', convert_to('demo-cipher-C-0006','UTF8'), convert_to('demo-hmac-C-0006','UTF8'), 'A62***0006', '新竹市北區中山路60號', '新竹市', NULL, NULL, NULL, NULL, 'active', '一般戶', 'M', '1958-12-01', '女兒', '許小芬', '新竹市北區中山路60號', '服務類別待社工評估後補齊'),
('40000000-0000-4000-8000-000000000007', '楊淑惠', '楊淑惠', convert_to('demo-cipher-C-0007','UTF8'), convert_to('demo-hmac-C-0007','UTF8'), 'A72***0007', '新竹縣竹北市光明一路70號', '新竹縣', 'CMS 4', 1, 1, NULL, 'active', '一般戶', 'F', '1949-04-25', '兒子', '楊文宏', '新竹縣竹北市光明一路70號', NULL),
('40000000-0000-4000-8000-000000000008', '蔡明宗', '蔡明宗', NULL, NULL, NULL, '苗栗縣竹南鎮中正路80號', '苗栗縣', NULL, 2, 2, NULL, 'active', '一般戶', 'M', '1960-08-14', '配偶', '蔡林秀', NULL, '新申請個案，尚未排班')
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name, name_normalized = EXCLUDED.name_normalized,
  national_id_cipher = EXCLUDED.national_id_cipher, national_id_hmac = EXCLUDED.national_id_hmac,
  national_id_masked = EXCLUDED.national_id_masked, home_address = EXCLUDED.home_address,
  region = EXCLUDED.region, ltc_level = EXCLUDED.ltc_level, service_category = EXCLUDED.service_category,
  service_usage_type = EXCLUDED.service_usage_type, claim_end_date = EXCLUDED.claim_end_date,
  status = EXCLUDED.status, household_type = EXCLUDED.household_type, gender = EXCLUDED.gender,
  birth_date = EXCLUDED.birth_date, care_contact_role = EXCLUDED.care_contact_role,
  care_contact_name = EXCLUDED.care_contact_name, registered_address = EXCLUDED.registered_address,
  remarks = EXCLUDED.remarks;

-- ============================================================
-- 6. 個案排班設定 (case_schedules) — C-0008 刻意不建排班，示範待排班個案
-- ============================================================
INSERT INTO case_schedules (id, case_id, site_id, effective_range, weekdays, trip_pattern, unit_price, distance_km, service_duration_min, service_code) VALUES
('41000000-0000-4000-8000-000000000001', '40000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001', '[2025-01-01,)', '{1,2,3,4,5}', 4, 115.00, 8.50, 15, 'BD03'),
('41000000-0000-4000-8000-000000000002', '40000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000001', '[2025-01-01,)', '{1,3,5}', 2, 115.00, 6.20, 10, 'BD03'),
('41000000-0000-4000-8000-000000000003', '40000000-0000-4000-8000-000000000003', '10000000-0000-4000-8000-000000000002', '[2025-01-01,)', '{2,4}', 2, 115.00, 5.00, 10, 'BD03'),
('41000000-0000-4000-8000-000000000004', '40000000-0000-4000-8000-000000000004', '10000000-0000-4000-8000-000000000002', '[2025-01-01,)', '{1,2,3,4,5}', 4, 115.00, 7.30, 15, 'BD03'),
('41000000-0000-4000-8000-000000000005', '40000000-0000-4000-8000-000000000005', '10000000-0000-4000-8000-000000000003', '[2025-01-01,2026-06-30)', '{1,3,5}', 2, 115.00, 4.80, 10, 'BD03'),
('41000000-0000-4000-8000-000000000006', '40000000-0000-4000-8000-000000000006', '10000000-0000-4000-8000-000000000003', '[2025-06-01,)', '{1,2,3,4,5}', 1, 115.00, 3.90, 10, 'BD03'),
('41000000-0000-4000-8000-000000000007', '40000000-0000-4000-8000-000000000007', '10000000-0000-4000-8000-000000000001', '[2025-01-01,)', '{2,4}', 4, 115.00, 9.10, 15, 'BD03')
ON CONFLICT (id) DO UPDATE SET
  case_id = EXCLUDED.case_id, site_id = EXCLUDED.site_id, effective_range = EXCLUDED.effective_range,
  weekdays = EXCLUDED.weekdays, trip_pattern = EXCLUDED.trip_pattern, unit_price = EXCLUDED.unit_price,
  distance_km = EXCLUDED.distance_km, service_duration_min = EXCLUDED.service_duration_min,
  service_code = EXCLUDED.service_code;

-- ============================================================
-- 7. 個案交通偏好設定 (case_transport_preferences)
-- ============================================================
INSERT INTO case_transport_preferences (case_id, site_id, outbound_vehicle_id, inbound_vehicle_id, site_name_raw) VALUES
('40000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000001', NULL),
('40000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000002', '20000000-0000-4000-8000-000000000002', NULL),
('40000000-0000-4000-8000-000000000003', '10000000-0000-4000-8000-000000000002', '20000000-0000-4000-8000-000000000003', '20000000-0000-4000-8000-000000000003', NULL),
('40000000-0000-4000-8000-000000000004', '10000000-0000-4000-8000-000000000002', '20000000-0000-4000-8000-000000000003', NULL, NULL),
('40000000-0000-4000-8000-000000000005', '10000000-0000-4000-8000-000000000003', '20000000-0000-4000-8000-000000000005', '20000000-0000-4000-8000-000000000005', NULL),
('40000000-0000-4000-8000-000000000006', NULL, NULL, NULL, '竹市舊站（待維護）'),
('40000000-0000-4000-8000-000000000007', '10000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000002', NULL)
ON CONFLICT (case_id) DO UPDATE SET
  site_id = EXCLUDED.site_id, outbound_vehicle_id = EXCLUDED.outbound_vehicle_id,
  inbound_vehicle_id = EXCLUDED.inbound_vehicle_id, site_name_raw = EXCLUDED.site_name_raw;

-- ============================================================
-- 8. 排班趟次時段明細 (schedule_legs)
-- ============================================================
INSERT INTO schedule_legs (id, schedule_id, leg_seq, direction, period, depart_time, vehicle_id) VALUES
('42000000-0000-4000-8000-000000000001', '41000000-0000-4000-8000-000000000001', 1, 'outbound', 'am', '07:30', '20000000-0000-4000-8000-000000000001'),
('42000000-0000-4000-8000-000000000002', '41000000-0000-4000-8000-000000000001', 2, 'inbound', 'am', '11:30', '20000000-0000-4000-8000-000000000001'),
('42000000-0000-4000-8000-000000000003', '41000000-0000-4000-8000-000000000001', 3, 'outbound', 'pm', '13:00', '20000000-0000-4000-8000-000000000001'),
('42000000-0000-4000-8000-000000000004', '41000000-0000-4000-8000-000000000001', 4, 'inbound', 'pm', '16:30', '20000000-0000-4000-8000-000000000001'),
('42000000-0000-4000-8000-000000000005', '41000000-0000-4000-8000-000000000002', 1, 'outbound', 'am', '08:00', '20000000-0000-4000-8000-000000000002'),
('42000000-0000-4000-8000-000000000006', '41000000-0000-4000-8000-000000000002', 2, 'inbound', 'pm', '16:00', '20000000-0000-4000-8000-000000000002'),
('42000000-0000-4000-8000-000000000007', '41000000-0000-4000-8000-000000000003', 1, 'outbound', 'am', '08:15', '20000000-0000-4000-8000-000000000003'),
('42000000-0000-4000-8000-000000000008', '41000000-0000-4000-8000-000000000003', 2, 'inbound', 'pm', '16:15', '20000000-0000-4000-8000-000000000003'),
('42000000-0000-4000-8000-000000000009', '41000000-0000-4000-8000-000000000004', 1, 'outbound', 'am', '07:45', '20000000-0000-4000-8000-000000000003'),
('42000000-0000-4000-8000-000000000010', '41000000-0000-4000-8000-000000000004', 2, 'inbound', 'am', '11:45', '20000000-0000-4000-8000-000000000003'),
('42000000-0000-4000-8000-000000000011', '41000000-0000-4000-8000-000000000004', 3, 'outbound', 'pm', '13:15', '20000000-0000-4000-8000-000000000003'),
('42000000-0000-4000-8000-000000000012', '41000000-0000-4000-8000-000000000004', 4, 'inbound', 'pm', '16:45', '20000000-0000-4000-8000-000000000003'),
('42000000-0000-4000-8000-000000000013', '41000000-0000-4000-8000-000000000005', 1, 'outbound', 'am', '08:30', '20000000-0000-4000-8000-000000000005'),
('42000000-0000-4000-8000-000000000014', '41000000-0000-4000-8000-000000000005', 2, 'inbound', 'pm', '16:30', '20000000-0000-4000-8000-000000000005'),
('42000000-0000-4000-8000-000000000015', '41000000-0000-4000-8000-000000000006', 1, 'outbound', 'am', '09:00', NULL),
('42000000-0000-4000-8000-000000000016', '41000000-0000-4000-8000-000000000007', 1, 'outbound', 'am', '07:50', '20000000-0000-4000-8000-000000000001'),
('42000000-0000-4000-8000-000000000017', '41000000-0000-4000-8000-000000000007', 2, 'inbound', 'am', '11:50', '20000000-0000-4000-8000-000000000001'),
('42000000-0000-4000-8000-000000000018', '41000000-0000-4000-8000-000000000007', 3, 'outbound', 'pm', '13:20', '20000000-0000-4000-8000-000000000002'),
('42000000-0000-4000-8000-000000000019', '41000000-0000-4000-8000-000000000007', 4, 'inbound', 'pm', '16:50', '20000000-0000-4000-8000-000000000002')
ON CONFLICT (id) DO UPDATE SET
  schedule_id = EXCLUDED.schedule_id, leg_seq = EXCLUDED.leg_seq, direction = EXCLUDED.direction,
  period = EXCLUDED.period, depart_time = EXCLUDED.depart_time, vehicle_id = EXCLUDED.vehicle_id;

-- ============================================================
-- 9. 照護人員 (caregivers)
-- ============================================================
INSERT INTO caregivers (id, site_id, site_name_raw, name, type, contact) VALUES
('50000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001', NULL, '周雅婷', 'case_manager', '03-1234567'),
('50000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000002', NULL, '謝文彬', 'specialist', '037-123456'),
('50000000-0000-4000-8000-000000000003', NULL, '舊竹南站（待維護）', '洪淑貞', 'case_manager', '037-654321')
ON CONFLICT (id) DO UPDATE SET
  site_id = EXCLUDED.site_id, site_name_raw = EXCLUDED.site_name_raw, name = EXCLUDED.name,
  type = EXCLUDED.type, contact = EXCLUDED.contact;

-- ============================================================
-- 10. 司機接送匯報表單 (driver_report_forms)
-- ============================================================
INSERT INTO driver_report_forms (id, vehicle_id, title, last_imported_at, status) VALUES
('60000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000001', '新竹一號車司機接送匯報表', '2026-08-04 20:00:00+08', 'active'),
('60000000-0000-4000-8000-000000000002', '20000000-0000-4000-8000-000000000003', '苗栗一號車司機接送匯報表', '2026-08-03 20:00:00+08', 'active')
ON CONFLICT (id) DO UPDATE SET
  vehicle_id = EXCLUDED.vehicle_id, title = EXCLUDED.title, last_imported_at = EXCLUDED.last_imported_at,
  status = EXCLUDED.status;

-- ============================================================
-- 11. 表單欄位對應 (form_columns)
-- ============================================================
INSERT INTO form_columns (id, form_id, column_index, column_header, cleaned_name, kind, mapping_status, case_id, leg_seq, suggested_case_id, suggestion_score) VALUES
('61000000-0000-4000-8000-000000000001', '60000000-0000-4000-8000-000000000001', 0, '時間戳記', NULL, 'meta', 'mapped', NULL, NULL, NULL, 0.0),
('61000000-0000-4000-8000-000000000002', '60000000-0000-4000-8000-000000000001', 1, '今日駕駛人', NULL, 'meta', 'mapped', NULL, NULL, NULL, 0.0),
('61000000-0000-4000-8000-000000000003', '60000000-0000-4000-8000-000000000001', 2, '王秀琴[去程]', '王秀琴', 'ride', 'mapped', '40000000-0000-4000-8000-000000000001', 1, NULL, 0.0),
('61000000-0000-4000-8000-000000000004', '60000000-0000-4000-8000-000000000001', 3, '王秀琴[回程]', '王秀琴', 'ride', 'mapped', '40000000-0000-4000-8000-000000000001', 2, NULL, 0.0),
('61000000-0000-4000-8000-000000000005', '60000000-0000-4000-8000-000000000001', 4, '李國華[去程]', '李國華', 'ride', 'mapped', '40000000-0000-4000-8000-000000000002', 1, NULL, 0.0),
('61000000-0000-4000-8000-000000000006', '60000000-0000-4000-8000-000000000001', 5, '問題回報', NULL, 'issue', 'mapped', NULL, NULL, NULL, 0.0),
('61000000-0000-4000-8000-000000000007', '60000000-0000-4000-8000-000000000001', 6, '楊淑惠2[去程]', '楊淑惠2', 'ride', 'pending', NULL, NULL, '40000000-0000-4000-8000-000000000007', 0.6),
('61000000-0000-4000-8000-000000000008', '60000000-0000-4000-8000-000000000002', 0, '時間戳記', NULL, 'meta', 'mapped', NULL, NULL, NULL, 0.0),
('61000000-0000-4000-8000-000000000009', '60000000-0000-4000-8000-000000000002', 1, '今日駕駛人', NULL, 'meta', 'mapped', NULL, NULL, NULL, 0.0),
('61000000-0000-4000-8000-000000000010', '60000000-0000-4000-8000-000000000002', 2, '張阿蘭[去程]', '張阿蘭', 'ride', 'mapped', '40000000-0000-4000-8000-000000000003', 1, NULL, 0.0),
('61000000-0000-4000-8000-000000000011', '60000000-0000-4000-8000-000000000002', 3, '張阿蘭[回程]', '張阿蘭', 'ride', 'mapped', '40000000-0000-4000-8000-000000000003', 2, NULL, 0.0),
('61000000-0000-4000-8000-000000000012', '60000000-0000-4000-8000-000000000002', 4, '陳福來[去程]', '陳福來', 'ride', 'mapped', '40000000-0000-4000-8000-000000000004', 1, NULL, 0.0)
ON CONFLICT (id) DO UPDATE SET
  form_id = EXCLUDED.form_id, column_index = EXCLUDED.column_index, column_header = EXCLUDED.column_header,
  cleaned_name = EXCLUDED.cleaned_name, kind = EXCLUDED.kind, mapping_status = EXCLUDED.mapping_status,
  case_id = EXCLUDED.case_id, leg_seq = EXCLUDED.leg_seq, suggested_case_id = EXCLUDED.suggested_case_id,
  suggestion_score = EXCLUDED.suggestion_score;

-- ============================================================
-- 12. 表單原始回報提交紀錄 (form_submissions)
-- ============================================================
INSERT INTO form_submissions (id, form_id, service_date, submitted_at, driver_name_raw, driver_id, source, payload, issue_text, anomaly_flags) VALUES
('62000000-0000-4000-8000-000000000001', '60000000-0000-4000-8000-000000000001', '2026-08-03', '2026-08-03 20:05:00+08', '林建成', '30000000-0000-4000-8000-000000000001', 'import', '{}'::jsonb, NULL, NULL),
('62000000-0000-4000-8000-000000000002', '60000000-0000-4000-8000-000000000001', '2026-08-04', '2026-08-04 20:10:00+08', '林建成', '30000000-0000-4000-8000-000000000001', 'import', '{}'::jsonb, NULL, NULL),
('62000000-0000-4000-8000-000000000003', '60000000-0000-4000-8000-000000000002', '2026-08-03', '2026-08-03 20:15:00+08', '黃志豪', '30000000-0000-4000-8000-000000000003', 'manual', '{}'::jsonb, '個案臨時取消', '{cancelled}')
ON CONFLICT (id) DO UPDATE SET
  form_id = EXCLUDED.form_id, service_date = EXCLUDED.service_date, submitted_at = EXCLUDED.submitted_at,
  driver_name_raw = EXCLUDED.driver_name_raw, driver_id = EXCLUDED.driver_id, source = EXCLUDED.source,
  payload = EXCLUDED.payload, issue_text = EXCLUDED.issue_text, anomaly_flags = EXCLUDED.anomaly_flags;

-- ============================================================
-- 13. 回報搭乘來源紀錄 (ride_sources)
-- ============================================================
INSERT INTO ride_sources (id, submission_id, case_id, service_date, leg_seq, vehicle_id, driver_id, reported, source_column_index) VALUES
('63000000-0000-4000-8000-000000000001', '62000000-0000-4000-8000-000000000001', '40000000-0000-4000-8000-000000000001', '2026-08-03', 1, '20000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001', 'boarded', 2),
('63000000-0000-4000-8000-000000000002', '62000000-0000-4000-8000-000000000001', '40000000-0000-4000-8000-000000000001', '2026-08-03', 2, '20000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001', 'boarded', 3),
('63000000-0000-4000-8000-000000000003', '62000000-0000-4000-8000-000000000002', '40000000-0000-4000-8000-000000000001', '2026-08-04', 1, '20000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001', 'absent', 2),
('63000000-0000-4000-8000-000000000004', '62000000-0000-4000-8000-000000000003', '40000000-0000-4000-8000-000000000003', '2026-08-03', 1, '20000000-0000-4000-8000-000000000003', '30000000-0000-4000-8000-000000000003', 'boarded', 2),
('63000000-0000-4000-8000-000000000005', '62000000-0000-4000-8000-000000000003', '40000000-0000-4000-8000-000000000004', '2026-08-03', 1, '20000000-0000-4000-8000-000000000003', '30000000-0000-4000-8000-000000000003', 'absent', 4)
ON CONFLICT (id) DO UPDATE SET
  submission_id = EXCLUDED.submission_id, case_id = EXCLUDED.case_id, service_date = EXCLUDED.service_date,
  leg_seq = EXCLUDED.leg_seq, vehicle_id = EXCLUDED.vehicle_id, driver_id = EXCLUDED.driver_id,
  reported = EXCLUDED.reported, source_column_index = EXCLUDED.source_column_index;

-- ============================================================
-- 14. 搭乘紀錄合併主表 (ride_records)
-- rr6：merged/effective 不一致且已由管理員更正 → 示範衝突已排除流程。
-- ============================================================
INSERT INTO ride_records (id, case_id, service_date, leg_seq, merged_status, effective_status, vehicle_id, driver_id,
    has_conflict, conflict_resolved_at, conflict_resolved_by, corrected_by, corrected_at, correction_reason) VALUES
('64000000-0000-4000-8000-000000000001', '40000000-0000-4000-8000-000000000001', '2026-08-03', 1, 'boarded', 'boarded', '20000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001', false, NULL, NULL, NULL, NULL, NULL),
('64000000-0000-4000-8000-000000000002', '40000000-0000-4000-8000-000000000001', '2026-08-03', 2, 'boarded', 'boarded', '20000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001', false, NULL, NULL, NULL, NULL, NULL),
('64000000-0000-4000-8000-000000000003', '40000000-0000-4000-8000-000000000001', '2026-08-04', 1, 'absent', 'absent', '20000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001', false, NULL, NULL, NULL, NULL, NULL),
('64000000-0000-4000-8000-000000000004', '40000000-0000-4000-8000-000000000002', '2026-08-03', 1, 'unreported', 'unreported', '20000000-0000-4000-8000-000000000002', '30000000-0000-4000-8000-000000000002', false, NULL, NULL, NULL, NULL, NULL),
('64000000-0000-4000-8000-000000000005', '40000000-0000-4000-8000-000000000003', '2026-08-03', 1, 'boarded', 'boarded', '20000000-0000-4000-8000-000000000003', '30000000-0000-4000-8000-000000000003', false, NULL, NULL, NULL, NULL, NULL),
('64000000-0000-4000-8000-000000000006', '40000000-0000-4000-8000-000000000004', '2026-08-03', 1, 'absent', 'boarded', '20000000-0000-4000-8000-000000000003', '30000000-0000-4000-8000-000000000003', true, '2026-08-05 09:00:00+08', '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002', '2026-08-05 09:00:00+08', '司機事後確認個案有搭乘，更正為已搭乘'),
('64000000-0000-4000-8000-000000000007', '40000000-0000-4000-8000-000000000007', '2026-08-03', 1, 'unreported', 'unreported', '20000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001', false, NULL, NULL, NULL, NULL, NULL)
ON CONFLICT (id) DO UPDATE SET
  case_id = EXCLUDED.case_id, service_date = EXCLUDED.service_date, leg_seq = EXCLUDED.leg_seq,
  merged_status = EXCLUDED.merged_status, effective_status = EXCLUDED.effective_status,
  vehicle_id = EXCLUDED.vehicle_id, driver_id = EXCLUDED.driver_id, has_conflict = EXCLUDED.has_conflict,
  conflict_resolved_at = EXCLUDED.conflict_resolved_at, conflict_resolved_by = EXCLUDED.conflict_resolved_by,
  corrected_by = EXCLUDED.corrected_by, corrected_at = EXCLUDED.corrected_at,
  correction_reason = EXCLUDED.correction_reason;

-- ============================================================
-- 15. 出勤紀錄 (attendance_records)
-- ============================================================
INSERT INTO attendance_records (id, driver_id, record_date, status, note) VALUES
('80000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001', '2026-08-03', 'work', NULL),
('80000000-0000-4000-8000-000000000002', '30000000-0000-4000-8000-000000000001', '2026-08-04', 'work', NULL),
('80000000-0000-4000-8000-000000000003', '30000000-0000-4000-8000-000000000002', '2026-08-03', 'leave', '特休'),
('80000000-0000-4000-8000-000000000004', '30000000-0000-4000-8000-000000000003', '2026-08-03', 'work', NULL),
('80000000-0000-4000-8000-000000000005', '30000000-0000-4000-8000-000000000003', '2026-08-05', 'sick', '感冒'),
('80000000-0000-4000-8000-000000000006', '30000000-0000-4000-8000-000000000004', '2026-08-03', 'off', NULL)
ON CONFLICT (id) DO UPDATE SET
  driver_id = EXCLUDED.driver_id, record_date = EXCLUDED.record_date, status = EXCLUDED.status, note = EXCLUDED.note;

-- ============================================================
-- 16. 車輛維修保養紀錄 (maintenance_logs)
-- ============================================================
INSERT INTO maintenance_logs (id, vehicle_id, service_date, mileage, items, vendor, cost, created_by) VALUES
('81000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000001', '2026-06-15', 45230.5, '定期保養、更換機油機油芯', '新竹汽車保養廠', 3200.00, '00000000-0000-0000-0000-000000000002'),
('81000000-0000-4000-8000-000000000002', '20000000-0000-4000-8000-000000000002', '2026-07-01', 38900.0, '煞車來令片更換', '福特原廠服務廠', 5600.00, '00000000-0000-0000-0000-000000000002'),
('81000000-0000-4000-8000-000000000003', '20000000-0000-4000-8000-000000000003', '2026-05-20', 62000.0, '定期保養', '苗栗汽車保養廠', 2800.00, '00000000-0000-0000-0000-000000000002'),
('81000000-0000-4000-8000-000000000004', '20000000-0000-4000-8000-000000000004', '2026-08-01', 71000.0, '故障排除中，暫停派車', NULL, 0.00, '00000000-0000-0000-0000-000000000002'),
('81000000-0000-4000-8000-000000000005', '20000000-0000-4000-8000-000000000005', '2026-07-10', 29800.0, '輪胎更換', '竹市輪胎行', 8000.00, '00000000-0000-0000-0000-000000000002')
ON CONFLICT (id) DO UPDATE SET
  vehicle_id = EXCLUDED.vehicle_id, service_date = EXCLUDED.service_date, mileage = EXCLUDED.mileage,
  items = EXCLUDED.items, vendor = EXCLUDED.vendor, cost = EXCLUDED.cost, created_by = EXCLUDED.created_by;

-- ============================================================
-- 17. 車輛油資紀錄 (fuel_logs)
-- ============================================================
INSERT INTO fuel_logs (id, vehicle_id, driver_id, fuel_date, liters, cost, created_by) VALUES
('82000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001', '2026-08-01', 45.5, 1450.00, '00000000-0000-0000-0000-000000000002'),
('82000000-0000-4000-8000-000000000002', '20000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001', '2026-08-08', 42.0, 1340.00, '00000000-0000-0000-0000-000000000002'),
('82000000-0000-4000-8000-000000000003', '20000000-0000-4000-8000-000000000002', '30000000-0000-4000-8000-000000000002', '2026-08-02', 50.0, 1600.00, '00000000-0000-0000-0000-000000000002'),
('82000000-0000-4000-8000-000000000004', '20000000-0000-4000-8000-000000000003', '30000000-0000-4000-8000-000000000003', '2026-08-03', 38.5, 1230.00, '00000000-0000-0000-0000-000000000002'),
('82000000-0000-4000-8000-000000000005', '20000000-0000-4000-8000-000000000005', NULL, '2026-08-04', 40.0, 1280.00, '00000000-0000-0000-0000-000000000002')
ON CONFLICT (id) DO UPDATE SET
  vehicle_id = EXCLUDED.vehicle_id, driver_id = EXCLUDED.driver_id, fuel_date = EXCLUDED.fuel_date,
  liters = EXCLUDED.liters, cost = EXCLUDED.cost, created_by = EXCLUDED.created_by;

-- ============================================================
-- 18. 通知收件人管理 (notification_recipients) — 以 (topic, email) 為自然鍵
-- ============================================================
INSERT INTO notification_recipients (topic, email, display_name, active, created_by) VALUES
('missing_report', 'ops1@ltc.example.com', '業務窗口一', true, '00000000-0000-0000-0000-000000000002'),
('driver_leave', 'ops2@ltc.example.com', '業務窗口二', true, '00000000-0000-0000-0000-000000000002'),
('export_failed', 'it@ltc.example.com', 'IT支援', false, '00000000-0000-0000-0000-000000000002')
ON CONFLICT (topic, email) DO UPDATE SET
  display_name = EXCLUDED.display_name, active = EXCLUDED.active, created_by = EXCLUDED.created_by;

-- ============================================================
-- 19. 通知歷史日誌 (notification_log) — 固定 id（bigserial 允許指定值）
-- ============================================================
INSERT INTO notification_log (id, topic, channel, recipient_emails, subject, content_summary, status, error_message, triggered_by, triggered_by_name, sent_at) VALUES
(9000001, 'missing_report', 'email', '{ops1@ltc.example.com}', '[LTC] 8/4 王秀琴 個案漏報接送', '個案王秀琴 2026-08-04 去程無到離紀錄', 'sent', NULL, '00000000-0000-0000-0000-000000000002', '系統管理員', '2026-08-05 08:00:00+08'),
(9000002, 'export_failed', 'system', '{it@ltc.example.com}', '[LTC] 政府申報匯出失敗', NULL, 'failed', '個案身分證加密欄位缺漏', '00000000-0000-0000-0000-000000000002', '系統管理員', '2026-08-05 10:00:00+08'),
(9000003, 'month_end', 'email', '{ops1@ltc.example.com,ops2@ltc.example.com}', '[LTC] 7月結報提醒', '7月接送紀錄請於本週完成核對', 'sent', NULL, '00000000-0000-0000-0000-000000000002', '系統管理員', '2026-08-01 09:00:00+08')
ON CONFLICT (id) DO UPDATE SET
  topic = EXCLUDED.topic, channel = EXCLUDED.channel, recipient_emails = EXCLUDED.recipient_emails,
  subject = EXCLUDED.subject, content_summary = EXCLUDED.content_summary, status = EXCLUDED.status,
  error_message = EXCLUDED.error_message, triggered_by = EXCLUDED.triggered_by,
  triggered_by_name = EXCLUDED.triggered_by_name, sent_at = EXCLUDED.sent_at;

-- ============================================================
-- 20. 匯出工作 (export_jobs)
-- ============================================================
INSERT INTO export_jobs (id, job_type, period_ym, region, format, filter_case_ids, status, storage_path, file_checksum, error_message, created_by, created_by_name, finished_at) VALUES
('70000000-0000-4000-8000-000000000001', 'gov_claim', '11507', '新竹縣', 'zip',
  ARRAY['40000000-0000-4000-8000-000000000001','40000000-0000-4000-8000-000000000002','40000000-0000-4000-8000-000000000007']::uuid[],
  'succeeded', '/exports/gov_claim/11507-hsinchu.zip', 'sha256:demo-checksum-0001', NULL,
  '00000000-0000-0000-0000-000000000002', '系統管理員', '2026-08-02 10:30:00+08'),
('70000000-0000-4000-8000-000000000002', 'trip_summary', '11508', '苗栗縣', 'xlsx',
  ARRAY['40000000-0000-4000-8000-000000000004']::uuid[],
  'failed', NULL, NULL, '個案陳福來身分證加密欄位缺漏，請補齊後重新匯出',
  '00000000-0000-0000-0000-000000000002', '系統管理員', '2026-09-01 11:00:00+08')
ON CONFLICT (id) DO UPDATE SET
  job_type = EXCLUDED.job_type, period_ym = EXCLUDED.period_ym, region = EXCLUDED.region,
  format = EXCLUDED.format, filter_case_ids = EXCLUDED.filter_case_ids, status = EXCLUDED.status,
  storage_path = EXCLUDED.storage_path, file_checksum = EXCLUDED.file_checksum,
  error_message = EXCLUDED.error_message, created_by = EXCLUDED.created_by,
  created_by_name = EXCLUDED.created_by_name, finished_at = EXCLUDED.finished_at;

-- ============================================================
-- 21. 匯出快照行明細 (export_lines)
-- ============================================================
INSERT INTO export_lines (id, job_id, line_no, case_id, national_id_masked, service_date_roc, raw_payload) VALUES
('71000000-0000-4000-8000-000000000001', '70000000-0000-4000-8000-000000000001', 1, '40000000-0000-4000-8000-000000000001', 'A12***0001', 1150801, '{}'::jsonb),
('71000000-0000-4000-8000-000000000002', '70000000-0000-4000-8000-000000000001', 2, '40000000-0000-4000-8000-000000000002', 'A22***0002', 1150802, '{}'::jsonb)
ON CONFLICT (id) DO UPDATE SET
  job_id = EXCLUDED.job_id, line_no = EXCLUDED.line_no, case_id = EXCLUDED.case_id,
  national_id_masked = EXCLUDED.national_id_masked, service_date_roc = EXCLUDED.service_date_roc,
  raw_payload = EXCLUDED.raw_payload;

-- ============================================================
-- 22. 匯出個案檔案 (export_job_files)
-- ============================================================
INSERT INTO export_job_files (id, job_id, case_id, seq, case_code, case_name, region, file_name, row_count, file_checksum) VALUES
('72000000-0000-4000-8000-000000000001', '70000000-0000-4000-8000-000000000001', '40000000-0000-4000-8000-000000000001', 1, 'C-0001', '王秀琴', '新竹縣', 'C-0001_11507.xlsx', 20, 'sha256:demo-checksum-file-0001'),
('72000000-0000-4000-8000-000000000002', '70000000-0000-4000-8000-000000000001', '40000000-0000-4000-8000-000000000002', 2, 'C-0002', '李國華', '新竹縣', 'C-0002_11507.xlsx', 18, 'sha256:demo-checksum-file-0002')
ON CONFLICT (id) DO UPDATE SET
  job_id = EXCLUDED.job_id, case_id = EXCLUDED.case_id, seq = EXCLUDED.seq, case_code = EXCLUDED.case_code,
  case_name = EXCLUDED.case_name, region = EXCLUDED.region, file_name = EXCLUDED.file_name,
  row_count = EXCLUDED.row_count, file_checksum = EXCLUDED.file_checksum;

-- ============================================================
-- 23. 系統稽核日誌 (audit_log) — 固定 id（bigserial 允許指定值）
-- ============================================================
INSERT INTO audit_log (id, actor_id, actor_role, action, entity_type, entity_id, before_data, after_data, ip_address, created_at) VALUES
(9100001, '00000000-0000-0000-0000-000000000002', 'admin', 'case.create', 'case', '40000000-0000-4000-8000-000000000001', NULL, '{"code":"C-0001","status":"active"}'::jsonb, '127.0.0.1', '2025-01-01 09:00:00+08'),
(9100002, '00000000-0000-0000-0000-000000000002', 'admin', 'case.update', 'case', '40000000-0000-4000-8000-000000000004', '{"status":"active"}'::jsonb, '{"status":"suspended"}'::jsonb, '127.0.0.1', '2026-07-20 09:00:00+08'),
(9100003, '00000000-0000-0000-0000-000000000002', 'admin', 'ride_record.correct', 'ride_record', '64000000-0000-4000-8000-000000000006', '{"effective_status":"absent"}'::jsonb, '{"effective_status":"boarded"}'::jsonb, '127.0.0.1', '2026-08-05 09:00:00+08'),
(9100004, '00000000-0000-0000-0000-000000000002', 'admin', 'export_job.run', 'export_job', '70000000-0000-4000-8000-000000000001', NULL, '{"status":"succeeded"}'::jsonb, '127.0.0.1', '2026-08-02 10:30:00+08'),
(9100005, '00000000-0000-0000-0000-000000000002', 'admin', 'driver.create', 'driver', '30000000-0000-4000-8000-000000000005', NULL, '{"code":"DRV-005","status":"active"}'::jsonb, '127.0.0.1', '2023-01-01 09:00:00+08'),
(9100006, '00000000-0000-0000-0000-000000000002', 'admin', 'vehicle.status_change', 'vehicle', '20000000-0000-4000-8000-000000000004', '{"status":"active"}'::jsonb, '{"status":"maintenance"}'::jsonb, '127.0.0.1', '2026-08-01 08:00:00+08')
ON CONFLICT (id) DO NOTHING;
