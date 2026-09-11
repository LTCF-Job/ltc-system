-- 清除 tests/qa-crud 產生的測試資料（名稱一律帶 QA 前綴或 QA 標記）。
-- 用法：docker exec -i ltc-postgres psql -U postgres -d ltc_system < tests/qa-crud/cleanup.sql
-- 有外鍵引用的資料一律先刪子表再刪主表；刪不掉代表測試留下了真實關聯，需人工確認。

BEGIN;

DELETE FROM maintenance_logs
WHERE vehicle_id IN (SELECT id FROM vehicles WHERE display_name LIKE 'QA%' OR plate_no LIKE 'QA%');

DELETE FROM fuel_logs
WHERE vehicle_id IN (SELECT id FROM vehicles WHERE display_name LIKE 'QA%' OR plate_no LIKE 'QA%')
   OR driver_id IN (SELECT id FROM drivers WHERE name LIKE 'QA%');

DELETE FROM attendance_records WHERE driver_id IN (SELECT id FROM drivers WHERE name LIKE 'QA%');
DELETE FROM driver_assignments
WHERE driver_id IN (SELECT id FROM drivers WHERE name LIKE 'QA%')
   OR vehicle_id IN (SELECT id FROM vehicles WHERE display_name LIKE 'QA%' OR plate_no LIKE 'QA%');

DELETE FROM driver_report_forms
WHERE vehicle_id IN (SELECT id FROM vehicles WHERE display_name LIKE 'QA%' OR plate_no LIKE 'QA%');

DELETE FROM case_schedules WHERE case_id IN (SELECT id FROM cases WHERE name LIKE 'QA%');
DELETE FROM case_import_idempotency WHERE case_id IN (SELECT id FROM cases WHERE name LIKE 'QA%');
DELETE FROM case_import_duplicate_rows WHERE name LIKE 'QA%';

DELETE FROM cases WHERE name LIKE 'QA%';
DELETE FROM caregivers WHERE name LIKE 'QA%';
DELETE FROM drivers WHERE name LIKE 'QA%';
DELETE FROM vehicles WHERE display_name LIKE 'QA%' OR plate_no LIKE 'QA%' OR plate_no ~ '^[0-9]{4}-000[0-9]$';
DELETE FROM sites WHERE name LIKE 'QA%';
-- regions 主檔已於 migration 000048 移除（區域改為 sites.region 自由文字欄位），不再需要單獨清理。
DELETE FROM holidays WHERE name LIKE 'QA%';
DELETE FROM notification_recipients WHERE email LIKE 'qa-%';
DELETE FROM roles WHERE is_system = false AND name LIKE 'QA%';

COMMIT;
