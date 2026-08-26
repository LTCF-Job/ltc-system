-- Migration: 000003_seed_demo_data.down.sql
-- Description: 回滾 Demo 種子資料

DELETE FROM cases WHERE code IN ('C0001', 'C0002', 'C0003', 'C0004', 'C0005', 'C0006', 'C0007', 'C0008', 'C0009', 'C0010');
DELETE FROM drivers WHERE code IN ('D0001', 'D0002', 'D0003', 'D0004', 'D0005', 'D0006');
DELETE FROM vehicles WHERE display_name IN ('竹北一車', '竹北二車', '竹南1車', '竹南2車', '竹東一車', '苗栗市1車');
DELETE FROM sites WHERE code IN ('S001', 'S002', 'S003', 'S004', 'S005', 'S006');
