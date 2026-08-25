-- Migration: 000003_seed_demo_data.up.sql
-- Description: 載入全類型與全選項之完整 Demo 種子資料（包含合規 AES-GCM 身分證加密與 HMAC 雜湊）

-- 1. 據點種子資料
INSERT INTO sites (id, code, name, address, region, open_days, status) VALUES
('11111111-1111-1111-1111-111111111101', 'S001', '竹北日照中心', '新竹縣竹北市光明六路100號', 'hsinchu', '{1,2,3,4,5}', 'active'),
('11111111-1111-1111-1111-111111111102', 'S002', '竹南日照據點', '苗栗縣竹南鎮中正路200號', 'miaoli', '{1,2,3,4,5}', 'active'),
('11111111-1111-1111-1111-111111111103', 'S003', '湖口長照據點', '新竹縣湖口鄉成功路50號', 'hsinchu', '{1,3,5}', 'active'),
('11111111-1111-1111-1111-111111111104', 'S004', '苗栗市社區據點', '苗栗縣苗栗市府前路1號', 'miaoli', '{1,2,3,4,5}', 'active'),
('11111111-1111-1111-1111-111111111105', 'S005', '新竹縣輔具資源中心', '新竹縣竹北市光明九路235號', 'hsinchu', '{1,2,3,4,5,6}', 'active'),
('11111111-1111-1111-1111-111111111106', 'S006', '竹南身障日間作業據點', '苗栗縣竹南鎮博愛街120號', 'miaoli', '{2,4}', 'active')
ON CONFLICT DO NOTHING;

-- 2. 車輛種子資料
INSERT INTO vehicles (id, plate_no, display_name, region, status) VALUES
('22222222-2222-2222-2222-222222222201', 'BZG-7915', '竹北一車', 'hsinchu', 'active'),
('22222222-2222-2222-2222-222222222202', 'ABC-1234', '竹北二車', 'hsinchu', 'active'),
('22222222-2222-2222-2222-222222222203', 'DEF-5678', '竹南1車', 'miaoli', 'active'),
('22222222-2222-2222-2222-222222222204', 'GHI-9012', '竹南2車', 'miaoli', 'active'),
('22222222-2222-2222-2222-222222222205', 'JKL-3456', '竹東一車', 'hsinchu', 'maintenance'),
('22222222-2222-2222-2222-222222222206', 'MNO-7890', '苗栗市1車', 'miaoli', 'active')
ON CONFLICT DO NOTHING;

-- 3. 司機種子資料
INSERT INTO drivers (id, code, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked, email, region, status) VALUES
('33333333-3333-3333-3333-333333333301', 'D0001', '郭澤威', '郭澤威', '\x7da887ae52a21c20061f8d47dd4242f4b29efe0f170aa4566e41a9fb4826a9708cee68f43353', '\x625efe721ab81b570831bcd2380ebc64ab13ec47f360337a2a124aca62f76fec', 'G12***6465', 'driver1@ltc.example.com', 'hsinchu', 'active'),
('33333333-3333-3333-3333-333333333302', 'D0002', '林志豪', '林志豪', '\x4c07d34925dfb92225311c3ae385d97c58f2e92952be7eaf7247764cab4a4d3f32d4f4aa9982', '\xf593d69f7ea123fb1e5970baa319314adda8ba061f966d9d55dc14b9625a8a3c', 'J12***1223', 'driver2@ltc.example.com', 'hsinchu', 'active'),
('33333333-3333-3333-3333-333333333303', 'D0003', '陳國華', '陳國華', '\x6137ede43dab5310147c39e11e3557fbaaefbe9a9c32da812a9c1ad536c03211d62300a48e70', '\xf6ac65c422d6463d2397cde1cc9f37f3a61ff59c1f383706a0925b54bdde1593', 'K12***8177', 'driver3@ltc.example.com', 'miaoli', 'active'),
('33333333-3333-3333-3333-333333333304', 'D0004', '曾建宏', '曾建宏', '\xdf807e73e919644962721aab3188972c17353e5fd34babb38b764281d0e95f24ad7ecd39aae5', '\x14aa515996f819010ecde9c61304913833ef880dcd0da8b1079138dd2bc2d442', 'O12***3221', 'driver4@ltc.example.com', 'miaoli', 'active'),
('33333333-3333-3333-3333-333333333305', 'D0005', '吳秀珠', '吳秀珠', '\x1686f958748ad0540b102bd3b21b910dfcbb00ee452f1086134bbb119ea001d06ce3b4cd5bc8', '\x76c1fe826937133304d2020bab3896a04719fdefe14368b2dc9c5f6fcf947b09', 'J22***7881', 'driver5@ltc.example.com', 'miaoli', 'active'),
('33333333-3333-3333-3333-333333333306', 'D0006', '黃建民', '黃建民', '\x178ffd104128faf92b615ece1d12cf1dec00197038db6cfae58042e36b0e995e0ae4070c971a', '\x7e1c09dc77638298b9eb6b5a16c5c885d9b0a9a6e107f9a7b065146270382736', 'H12***4552', 'driver6@ltc.example.com', 'hsinchu', 'resigned')
ON CONFLICT DO NOTHING;

-- 4. 司機車輛指派種子資料
INSERT INTO driver_assignments (driver_id, vehicle_id, is_primary, effective_range) VALUES
('33333333-3333-3333-3333-333333333301', '22222222-2222-2222-2222-222222222201', true, '[2026-01-01,infinity)'::daterange),
('33333333-3333-3333-3333-333333333302', '22222222-2222-2222-2222-222222222202', true, '[2026-01-01,infinity)'::daterange),
('33333333-3333-3333-3333-333333333303', '22222222-2222-2222-2222-222222222204', true, '[2026-01-01,infinity)'::daterange),
('33333333-3333-3333-3333-333333333304', '22222222-2222-2222-2222-222222222203', true, '[2026-02-01,infinity)'::daterange),
('33333333-3333-3333-3333-333333333305', '22222222-2222-2222-2222-222222222206', true, '[2026-03-01,infinity)'::daterange)
ON CONFLICT DO NOTHING;

-- 5. 個案主檔種子資料
INSERT INTO cases (id, code, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked, home_address, region, service_category, service_usage_type, claim_start_date, claim_end_date, status) VALUES
('55555555-5555-5555-5555-555555555501', 'C0001', '蔡曾切', '蔡曾切', '\x9b6834866e222c9faa993a1056c5bbaba4559168fd15ba1b7cae42b88280bf2ff753f2e56d3b', '\xaf82f3adaccea1412a8ac2de50a0b4b444196980f788a767a3e214e6e9f90115', 'A20***9750', '苗栗縣竹南鎮大營路123號', 'miaoli', 1, 2, '2026-07-01', NULL, 'active'),
('55555555-5555-5555-5555-555555555502', 'C0002', '葉秀珍', '葉秀珍', '\x35e23cef60cf4541dce73a441e8c4735c84ddc8f3dd8b84f0aaf38014920009c7f3644687e3d', '\xafdc6a9b55a4981578e8dbb7797c45ffbd568d6aba327700dda0d13cd52a887b', 'J22***3456', '新竹縣竹北市中正西路50號', 'hsinchu', 1, 1, '2026-07-01', NULL, 'active'),
('55555555-5555-5555-5555-555555555503', 'C0003', '吳𣵛桂', '吳𣵛桂', '\x7942f400a0d738df1c0a5e395c58f6e5e80c5539c9b7badc09426cc5062a857c43b9e12db001', '\x4d2d40806290fdb807fc53cb90f6e729da0b4e8fdf2e5ed4ab92048ee6dbc38b', 'H22***6543', '新竹縣竹北市福興東路二段88號', 'hsinchu', 1, 2, '2026-07-01', NULL, 'active'),
('55555555-5555-5555-5555-555555555504', 'C0004', '張詹竹妹', '張詹竹妹', '\xc89d252e9de44b0d1b0461ef8cb551c6e660d900b495f2d873f6d0361ab3966633a910ab7628', '\x15e73f51dbc913f54c993bbc4cb4146257bf6e189a585221f39dbb50b01f255f', 'O20***2334', '新竹縣竹北市三民路15號', 'hsinchu', 1, 2, '2026-07-01', NULL, 'suspended'),
('55555555-5555-5555-5555-555555555505', 'C0005', '李國盛', '李國盛', '\xba9762fde82a7bda6cce3c49b97205ba199006ed2cb4a8e4c50cdda928620535fd3cceab496e', '\x7b682563606c4fb4565cf60f8f272c18d6693ecd1efae8d9a34073a49f93380c', 'J12***9001', '新竹縣竹北市文興路一段200號', 'hsinchu', 2, 1, '2026-07-01', NULL, 'active'),
('55555555-5555-5555-5555-555555555506', 'C0006', '陳素貞', '陳素貞', '\x210bd72572a4eb0e9404aff8d4411632b87da9ddbb6f4fc5d7236e705ce0aee417df44abd663', '\x075e221aacf47bc11c2af31ce61bab37e9f7892c5debe04ed695910b1cb94e20', 'J22***1223', '新竹縣竹北市縣政九路80號', 'hsinchu', 1, 3, '2026-06-01', '2026-07-31', 'closed'),
('55555555-5555-5555-5555-555555555507', 'C0007', '黃天賜', '黃天賜', '\x464a91009e7ba7313f22bd57ca153d8265a161e2899249599758e9612512044f1acac2e50b2e', '\xf71a1031cfa4a30443af618b43eed781d859d9351bcee90a4e17e9cb63712c98', 'K12***0112', '苗栗縣竹南鎮延平路66號', 'miaoli', 1, 4, '2026-07-01', NULL, 'active'),
('55555555-5555-5555-5555-555555555508', 'C0008', '彭阿土', '彭阿土', '\x1710688a5a260242d36b2f320a7569f83d8786bcd6387e6bd6e24b49add2263d9fed5703f185', '\x06095a1a63ca92cbe55e4d560b49246ef68db7dbaa54b2e97070f7f3679aacf6', 'K10***3445', '苗栗縣苗栗市中正路500號', 'miaoli', 2, 4, '2026-07-01', NULL, 'active'),
('55555555-5555-5555-5555-555555555509', 'C0009', '邱美蘭', '邱美蘭', '\x467a36afefc2ebef3713cab821704ff8248f3d7a3da6f6d2dd02ca74806c041b54779b4cc6b1', '\x2f175d63a2a89f5b34e92bfb3513421ca1c89fbc495cf91ed2edb83f59d9b878', 'J20***7889', '新竹縣湖口鄉達生路33號', 'hsinchu', 1, 3, '2026-07-01', NULL, 'active'),
('55555555-5555-5555-5555-555555555510', 'C0010', '林阿祥', '林阿祥', '\x76162a62f88d63972683c10c8be3667cb7c9ac333366ede6f663414b6a5f5e32946cbf70cbe9', '\x1cef49a5ef0c37331231725b26a981b51354f983db0ad8248735211aaa4bd20d', 'K12***4567', '苗栗縣竹南鎮光復路88號', 'miaoli', 1, 2, '2026-07-01', NULL, 'active')
ON CONFLICT (id) DO UPDATE SET
  national_id_cipher = EXCLUDED.national_id_cipher,
  national_id_hmac = EXCLUDED.national_id_hmac,
  national_id_masked = EXCLUDED.national_id_masked;

-- 6. 個案排班種子資料
INSERT INTO case_schedules (id, case_id, site_id, effective_range, weekdays, trip_pattern, unit_price, distance_km, service_duration_min, service_code) VALUES
('66666666-6666-6666-6666-666666666601', '55555555-5555-5555-5555-555555555501', '11111111-1111-1111-1111-111111111102', '[2026-07-01,infinity)'::daterange, '{1,2,3,4,5}', 2, 115.00, 4.5, 10, 'BD03'),
('66666666-6666-6666-6666-666666666602', '55555555-5555-5555-5555-555555555502', '11111111-1111-1111-1111-111111111101', '[2026-07-01,infinity)'::daterange, '{1,2,3,4,5}', 2, 115.00, 3.8, 10, 'BD03')
ON CONFLICT DO NOTHING;

-- 7. 個案時段種子資料
INSERT INTO schedule_legs (id, schedule_id, leg_seq, direction, period, depart_time, arrive_time, run_no, vehicle_id) VALUES
('77777777-7777-7777-7777-777777777701', '66666666-6666-6666-6666-666666666601', 1, 'outbound', 'am', '09:00:00', '09:10:00', 1, '22222222-2222-2222-2222-222222222203'),
('77777777-7777-7777-7777-777777777702', '66666666-6666-6666-6666-666666666601', 2, 'inbound', 'pm', '16:00:00', '16:10:00', 1, '22222222-2222-2222-2222-222222222203'),
('77777777-7777-7777-7777-777777777703', '66666666-6666-6666-6666-666666666602', 1, 'outbound', 'am', '09:30:00', '09:40:00', 1, '22222222-2222-2222-2222-222222222201'),
('77777777-7777-7777-7777-777777777704', '66666666-6666-6666-6666-666666666602', 2, 'inbound', 'pm', '15:30:00', '15:40:00', 1, '22222222-2222-2222-2222-222222222201')
ON CONFLICT DO NOTHING;

-- 8. 通知收件人種子資料
INSERT INTO notification_recipients (id, topic, email, display_name, active, created_by) VALUES
(1, 'missing_report', 'admin@ltc.example.com', '系統管理員', true, '00000000-0000-0000-0000-000000000001'),
(2, 'driver_leave', 'dispatch@ltc.example.com', '調度組', true, '00000000-0000-0000-0000-000000000001'),
(3, 'month_end', 'finance@ltc.example.com', '行政會計組', true, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (id) DO NOTHING;

-- 9. 通知歷史紀錄種子資料
INSERT INTO notification_log (id, topic, channel, recipient_emails, subject, content_summary, status, triggered_by, triggered_by_name, sent_at) VALUES
(1, 'missing_report', 'email', '{"admin@ltc.example.com"}', '【長照系統】未回報催報通知', '已發送未回報提醒', 'sent', '00000000-0000-0000-0000-000000000001', '系統管理員', now())
ON CONFLICT (id) DO NOTHING;
